package filter

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/AdguardTeam/AdGuardDNS/internal/agd"
	"github.com/AdguardTeam/AdGuardDNS/internal/dnsmsg"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter"
	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
)

// Safe Search

// safeSearch is a filter that enforces safe search.
type safeSearch struct {
	// resultCache contains cached results.
	resultCache *resultCache

	// rslvCache contains resolved IPs.
	rslvCache *resolveCache

	// flt is used to filter requests.
	flt *ruleListFilter

	// errColl is used to report rare errors.
	errColl agd.ErrorCollector
}

// safeSearchConfig contains configuration for the safe search filter.
type safeSearchConfig struct {
	rslvCache *resolveCache

	errColl agd.ErrorCollector

	list *agd.FilterList

	cacheDir string

	ttl time.Duration

	//lint:ignore U1000 TODO(a.garipov): Currently unused.  See AGDNS-398.
	cacheSize int
}

// newSafeSearch returns a new safe search filter.  c must not be nil.  The
// initial refresh should be called explicitly if necessary.
func newSafeSearch(c *safeSearchConfig) (f *safeSearch) {
	resCache := &resultCache{
		cache: cache.New(c.ttl, defaultResultCacheGC),
	}

	return &safeSearch{
		rslvCache:   c.rslvCache,
		resultCache: resCache,
		flt:         newRuleListFilter(c.list, c.cacheDir),
		errColl:     c.errColl,
	}
}

// type check
var _ qtHostFilter = (*safeSearch)(nil)

// filterReq implements the qtHostFilter interface for *safeSearch.  It modifies
// the response if host matches f.
func (f *safeSearch) filterReq(
	ctx context.Context,
	ri *agd.RequestInfo,
	req *dns.Msg,
) (r Result, err error) {
	qt := ri.QType
	network := dnsTypeToNetwork(qt)
	if network == "" {
		return nil, nil
	}

	host := ri.Host
	repHost, ok := f.safeSearchHost(host, qt)
	if !ok {
		log.Debug("filter %s: host %q is not on the list", f.flt.id(), host)

		return nil, nil
	}

	log.Debug("filter %s: found host %q", f.flt.id(), repHost)

	r, ok = f.resultCache.get(host, qt)
	if ok {
		return r.(*ResultModified).CloneForReq(req), nil
	}

	var result *dns.Msg
	ips, err := f.rslvCache.resolve(network, repHost)
	if err != nil {
		agd.Collectf(ctx, f.errColl, "filter %s: resolving: %w", f.flt.id(), err)

		result = ri.Messages.NewMsgSERVFAIL(req)
	} else {
		result, err = ri.Messages.NewIPRespMsg(req, ips...)
		if err != nil {
			return nil, fmt.Errorf("filter %s: creating modified result: %w", f.flt.id(), err)
		}
	}

	rm := &ResultModified{
		Msg:  result,
		List: f.flt.id(),
		Rule: agd.FilterRuleText(host),
	}

	// Copy the result to make sure that modifications to the result message
	// down the pipeline don't interfere with the cached value.
	//
	// See AGDNS-359.
	f.resultCache.set(host, qt, rm.Clone())

	return rm, nil
}

func (f *safeSearch) safeSearchHost(host string, qt dnsmsg.RRType) (ssHost string, ok bool) {
	switch qt {
	case dns.TypeA, dns.TypeAAAA:
		// Go on processing the request.
	default:
		return "", false
	}

	dnsReq := &urlfilter.DNSRequest{
		Hostname: host,
		DNSType:  qt,
		Answer:   false,
	}

	f.flt.mu.RLock()
	defer f.flt.mu.RUnlock()

	// Omit matching the result since it's always false for rewrite rules.
	dr, _ := f.flt.engine.MatchRequest(dnsReq)
	if dr == nil {
		return "", false
	}

	for _, nr := range dr.DNSRewrites() {
		drw := nr.DNSRewrite
		if drw.RCode != dns.RcodeSuccess {
			continue
		}

		if nc := drw.NewCNAME; nc != "" {
			return nc, true
		}

		// All the rules in safe search rule lists are expected to have either
		// A/AAAA or CNAME type.
		switch drw.RRType {
		case dns.TypeA, dns.TypeAAAA:
			return drw.Value.(net.IP).String(), true
		default:
			continue
		}
	}

	return "", false
}

// name implements the qtHostFilter interface for *safeSearch.
func (f *safeSearch) name() (n string) {
	if f == nil || f.flt == nil {
		return ""
	}

	return string(f.flt.id())
}

// refresh reloads the rule list data.  If acceptStale is true, and the cache
// file exists, the data is read from there regardless of its staleness.
func (f *safeSearch) refresh(ctx context.Context, acceptStale bool) (err error) {
	return f.flt.refresh(ctx, acceptStale)
}
