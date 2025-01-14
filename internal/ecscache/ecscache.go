// Package ecscache implements a EDNS Client Subnet (ECS) aware DNS cache that
// can be used as a dnsserver.Middleware.
package ecscache

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/AdguardTeam/AdGuardDNS/internal/agd"
	"github.com/AdguardTeam/AdGuardDNS/internal/agdnet"
	"github.com/AdguardTeam/AdGuardDNS/internal/dnsmsg"
	"github.com/AdguardTeam/AdGuardDNS/internal/dnsserver"
	"github.com/AdguardTeam/AdGuardDNS/internal/geoip"
	"github.com/AdguardTeam/AdGuardDNS/internal/metrics"
	"github.com/AdguardTeam/AdGuardDNS/internal/optlog"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/bluele/gcache"
	"github.com/miekg/dns"
)

// EDNS Client Subnet (ECS) Aware LRU Cache

// Middleware is a dnsserver.Middleware with ECS-aware caching.
type Middleware struct {
	// cache is the LRU cache for results indicating no support for ECS.
	cache gcache.Cache

	// ecsCache is the LRU cache for results indicating ECS support.
	ecsCache gcache.Cache

	// geoIP is used to get subnets for countries.
	geoIP geoip.Interface
}

// MiddlewareConfig is the configuration structure for NewMiddleware.
type MiddlewareConfig struct {
	// GeoIP is the GeoIP database used to get subnets for countries.  It must
	// not be nil.
	GeoIP geoip.Interface

	// Size is the number of entities to hold in the cache for hosts that don't
	// support ECS.  It must be greater than zero.
	Size int

	// ECSSize is the number of entities to hold in the cache for hosts that
	// support ECS.  It must be greater than zero.
	ECSSize int
}

// NewMiddleware initializes a new ECS-aware LRU caching middleware.  c must not
// be nil.
func NewMiddleware(c *MiddlewareConfig) (m *Middleware) {
	return &Middleware{
		cache:    gcache.New(c.Size).LRU().Build(),
		ecsCache: gcache.New(c.ECSSize).LRU().Build(),
		geoIP:    c.GeoIP,
	}
}

// type check
var _ dnsserver.Middleware = (*Middleware)(nil)

// Wrap implements the dnsserver.Middleware interface for *Middleware.
func (m *Middleware) Wrap(handler dnsserver.Handler) (wrapped dnsserver.Handler) {
	f := func(ctx context.Context, rw dnsserver.ResponseWriter, req *dns.Msg) (err error) {
		defer func() { err = errors.Annotate(err, "ecs-cache: %w") }()

		ri := agd.MustRequestInfoFromContext(ctx)

		// Try getting a cached result using the data from the request and the
		// subnet of the location.  If there is one, write, increment the
		// metrics, and return.  See also writeCachedResponse.
		ecsFam := ecsFamFromReq(ri)
		ctry, asn := locFromReq(ri)
		locSubnet, err := m.geoIP.SubnetByLocation(ctry, asn, ecsFam)
		if err != nil {
			return fmt.Errorf("getting subnet for country %s (family: %d): %w", ctry, ecsFam, err)
		}

		optlog.Debug3("ecscache: got ctry %s, asn %d, subnet %s", ctry, asn, locSubnet)

		reqDO := dnsmsg.IsDO(req)
		resp, found, hostHasECS := m.get(req, ri.Host, locSubnet, reqDO)
		if found {
			// Don't wrap the error, because it's informative enough as is.
			return writeCachedResponse(ctx, rw, req, resp, ri.ECS, ecsFam, hostHasECS)
		}

		// Perform an upstream request with the ECS data for the location.  If
		// successful, write, increment the metrics,  and return.  See also
		// writeUpstreamResponse.
		reqECS := &agd.ECS{
			Subnet: locSubnet,
			Scope:  0,
		}

		ecsReq := dnsmsg.Clone(req)
		err = setECS(ecsReq, reqECS, ecsFam, false)
		if err != nil {
			return fmt.Errorf("setting ecs for upstream req: %w", err)
		}

		nrw := dnsserver.NewNonWriterResponseWriter(rw.LocalAddr(), rw.RemoteAddr())
		err = handler.ServeDNS(ctx, nrw, ecsReq)
		if err != nil {
			return fmt.Errorf("requesting upstream: %w", err)
		}

		resp = nrw.Msg()
		if resp == nil {
			return nil
		}

		// Don't wrap the error, because it's informative enough as is.
		return m.writeUpstreamResponse(ctx, rw, req, resp, ri, locSubnet, ecsFam, reqDO)
	}

	return dnsserver.HandlerFunc(f)
}

// writeCachedResponse writes resp to rw replacing the ECS data and incrementing
// cache hit metrics if necessary.
func writeCachedResponse(
	ctx context.Context,
	rw dnsserver.ResponseWriter,
	req *dns.Msg,
	resp *dns.Msg,
	ecs *agd.ECS,
	ecsFam agdnet.AddrFamily,
	hostHasECS bool,
) (err error) {
	// Increment the hits metrics here, since we already know if the domain name
	// supports ECS or not from the cache data.  Increment the misses metrics in
	// writeResponse, where this information is retrieved from the upstream
	metrics.ECSCacheLookupTotalHits.Inc()

	if hostHasECS {
		metrics.ECSCacheLookupHasSupportHits.Inc()

		// Only set the ECS info if the request had it originally.
		if ecs != nil {
			err = setECS(resp, ecs, ecsFam, true)
			if err != nil {
				return fmt.Errorf("setting ecs for cached resp: %w", err)
			}
		}
	} else {
		metrics.ECSCacheLookupNoSupportHits.Inc()
	}

	// resp doesn't need additional filtering, since all hop-to-hop data has
	// been filtered when setting the cache, and the AD bit was set when resp
	// was being retrieved from the cache.
	err = rw.WriteMsg(ctx, req, resp)
	if err != nil {
		return fmt.Errorf("writing cached resp: %w", err)
	}

	return nil
}

// ecsFamFromReq returns the address family to use for the outgoing request from
// the request information using either the contents of the EDNS Client Subnet
// option or the real remote IP address.
func ecsFamFromReq(ri *agd.RequestInfo) (ecsFam agdnet.AddrFamily) {
	if ecs := ri.ECS; ecs != nil {
		if ecs.Subnet.Addr().Is4() {
			return agdnet.AddrFamilyIPv4
		}

		// Assume that families other than IPv4 and IPv6 have been filtered out
		// by dnsmsg.ECSFromMsg.
		return agdnet.AddrFamilyIPv6
	}

	// Set the address family parameter to the one of the client's address as
	// per RFC 7871.
	//
	// See https://datatracker.ietf.org/doc/html/rfc7871#section-7.1.1.
	if ri.RemoteIP.Is4() {
		return agdnet.AddrFamilyIPv4
	}

	return agdnet.AddrFamilyIPv6
}

// locFromReq returns the country and ASN from the request information using
// either the contents of the EDNS Client Subnet option or the real remote
// address.
func locFromReq(ri *agd.RequestInfo) (ctry agd.Country, asn agd.ASN) {
	if ecs := ri.ECS; ecs != nil && ecs.Location != nil {
		ctry = ecs.Location.Country
		asn = ecs.Location.ASN
	}

	if ctry == agd.CountryNone && ri.Location != nil {
		ctry = ri.Location.Country
		asn = ri.Location.ASN
	}

	return ctry, asn
}

// writeUpstreamResponse processes, caches, and writes the response to rw as
// well as updates cache metrics.
func (m *Middleware) writeUpstreamResponse(
	ctx context.Context,
	rw dnsserver.ResponseWriter,
	req *dns.Msg,
	resp *dns.Msg,
	ri *agd.RequestInfo,
	locSubnet netip.Prefix,
	ecsFam agdnet.AddrFamily,
	reqDO bool,
) (err error) {
	subnet, scope, err := dnsmsg.ECSFromMsg(resp)
	if err != nil {
		return fmt.Errorf("getting ecs from resp: %w", err)
	}

	optlog.Debug2("ecscache: upstream: %s/%d", subnet, scope)

	rmHopToHopData(resp, ri.QType, reqDO)

	metrics.ECSCacheLookupTotalMisses.Inc()

	hostHasECS := scope != 0 && subnet != (netip.Prefix{})
	if hostHasECS {
		metrics.ECSCacheLookupHasSupportMisses.Inc()
		metrics.ECSHasSupportCacheSize.Set(float64(m.ecsCache.Len(false)))
	} else {
		metrics.ECSCacheLookupNoSupportMisses.Inc()
		metrics.ECSNoSupportCacheSize.Set(float64(m.cache.Len(false)))

		locSubnet = agdnet.ZeroSubnet(ecsFam)
		ecsFam = agdnet.AddrFamilyNone
	}

	m.set(resp, ri.Host, locSubnet, hostHasECS, reqDO)

	// Set the AD bit and ECS information here, where it is safe to do so, since
	// a clone of the otherwise filtered response has already been set to cache.
	setRespAD(resp, req.AuthenticatedData, reqDO)
	if hostHasECS && ri.ECS != nil {
		err = setECS(resp, ri.ECS, ecsFam, true)
		if err != nil {
			return fmt.Errorf("responding with ecs: %w", err)
		}
	}

	err = rw.WriteMsg(ctx, req, resp)
	if err != nil {
		return fmt.Errorf("writing upstream resp: %w", err)
	}

	return nil
}
