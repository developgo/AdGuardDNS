package dnssvc_test

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/AdGuardDNS/internal/agd"
	"github.com/AdguardTeam/AdGuardDNS/internal/agdtest"
	"github.com/AdguardTeam/AdGuardDNS/internal/dnsserver"
	"github.com/AdguardTeam/AdGuardDNS/internal/dnsserver/dnsservertest"
	"github.com/AdguardTeam/AdGuardDNS/internal/dnsserver/forward"
	"github.com/AdguardTeam/AdGuardDNS/internal/dnssvc"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	agdtest.DiscardLogOutput(m)
}

// type check
var _ agd.Refresher = (*forward.Handler)(nil)

// Test Mocks

// type check
var _ dnssvc.Listener = (*testListener)(nil)

// testListener is a dnssvc.Listener for tests.
type testListener struct {
	onStart     func(ctx context.Context) (err error)
	onShutdown  func(ctx context.Context) (err error)
	onLocalAddr func() (addr net.Addr)
}

// Start implements the dnsserver.Server interface for *testListener.
func (l *testListener) Start(ctx context.Context) (err error) {
	return l.onStart(ctx)
}

// Shutdown implements the dnsserver.Server interface for *testListener.
func (l *testListener) Shutdown(ctx context.Context) (err error) {
	return l.onShutdown(ctx)
}

// LocalAddr implements the dnsserver.Server interface for *testListener.
func (l *testListener) LocalAddr() (addr net.Addr) {
	return l.onLocalAddr()
}

// newTestListener returns a *testListener all of methods of which panic with
// a "not implemented" message.
func newTestListener() (tl *testListener) {
	return &testListener{
		onStart:     func(_ context.Context) (err error) { panic("not implemented") },
		onShutdown:  func(_ context.Context) (err error) { panic("not implemented") },
		onLocalAddr: func() (addr net.Addr) { panic("not implemented") },
	}
}

// newTestListenerFunc returns a new NewListenerFunc that returns the provided
// listener.
func newTestListenerFunc(tl *testListener) (f dnssvc.NewListenerFunc) {
	return func(
		_ *agd.Server,
		_ string,
		_ netip.AddrPort,
		_ dnsserver.Handler,
		_ http.Handler,
		_ agd.ErrorCollector,
	) (l dnssvc.Listener, err error) {
		return tl, nil
	}
}

// type check
var _ dnsserver.ResponseWriter = (*testResponseWriter)(nil)

// testResponseWriter is a simple dnsserver.ResponseWriter for tests.
type testResponseWriter struct {
	onLocalAddr  func() (a net.Addr)
	onRemoteAddr func() (a net.Addr)
	onWriteMsg   func(ctx context.Context, req, resp *dns.Msg) (err error)
}

// LocalAddr returns the net.Addr of the server.
func (rw *testResponseWriter) LocalAddr() (a net.Addr) {
	return rw.onLocalAddr()
}

// RemoteAddr returns the net.Addr of the client that sent the current request.
func (rw *testResponseWriter) RemoteAddr() (a net.Addr) {
	return rw.onRemoteAddr()
}

// WriteMsg writes a reply back to the client.
func (rw *testResponseWriter) WriteMsg(ctx context.Context, req, resp *dns.Msg) (err error) {
	return rw.onWriteMsg(ctx, req, resp)
}

// Tests

func TestService_Start(t *testing.T) {
	var numStart, numShutdown atomic.Uint64

	tl := newTestListener()
	tl.onStart = func(_ context.Context) (err error) {
		numStart.Add(1)

		return nil
	}
	tl.onShutdown = func(_ context.Context) (err error) {
		numShutdown.Add(1)

		return nil
	}

	srv := &agd.Server{
		Name:          "test_server",
		Protocol:      agd.ProtoDNSUDP,
		BindAddresses: []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:53")},
	}

	c := &dnssvc.Config{
		NewListener: newTestListenerFunc(tl),
		Handler:     dnsservertest.DefaultHandler(),
		ServerGroups: []*agd.ServerGroup{{
			Name:    "test_group",
			Servers: []*agd.Server{srv},
		}},
		Upstream: &agd.Upstream{
			Server: netip.MustParseAddrPort("8.8.8.8:53"),
			FallbackServers: []netip.AddrPort{
				netip.MustParseAddrPort("1.1.1.1:53"),
			},
		},
	}

	svc, err := dnssvc.New(c)
	require.NoError(t, err)

	require.NotPanics(t, func() {
		err = svc.Start()
		assert.NoError(t, err)
		assert.Equal(t, uint64(1), numStart.Load())
	})

	require.NotPanics(t, func() {
		err = svc.Shutdown(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, uint64(1), numShutdown.Load())
	})
}

func TestNew(t *testing.T) {
	srvs := []*agd.Server{{
		DNSCrypt:      nil,
		TLS:           nil,
		Name:          "test_server_dns_tcp",
		Protocol:      agd.ProtoDNSTCP,
		BindAddresses: []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:53")},
	}, {
		DNSCrypt:      nil,
		TLS:           nil,
		Name:          "test_server_dns_udp",
		Protocol:      agd.ProtoDNSUDP,
		BindAddresses: []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:53")},
	}, {
		DNSCrypt:      &agd.DNSCryptConfig{},
		TLS:           nil,
		Name:          "test_server_dnscrypt_tcp",
		Protocol:      agd.ProtoDNSCryptTCP,
		BindAddresses: []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:8853")},
	}, {
		DNSCrypt:      &agd.DNSCryptConfig{},
		TLS:           nil,
		Name:          "test_server_dnscrypt_udp",
		Protocol:      agd.ProtoDNSCryptUDP,
		BindAddresses: []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:8853")},
	}, {
		DNSCrypt:      nil,
		TLS:           &tls.Config{},
		Name:          "test_server_doh",
		Protocol:      agd.ProtoDoH,
		BindAddresses: []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:443")},
	}, {
		DNSCrypt:      nil,
		TLS:           &tls.Config{},
		Name:          "test_server_doq",
		Protocol:      agd.ProtoDoQ,
		BindAddresses: []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:853")},
	}, {
		DNSCrypt:      nil,
		TLS:           &tls.Config{},
		Name:          "test_server_dot",
		Protocol:      agd.ProtoDoT,
		BindAddresses: []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:853")},
	}}

	c := &dnssvc.Config{
		Handler: dnsservertest.DefaultHandler(),
		ServerGroups: []*agd.ServerGroup{{
			Name:    "test_group",
			Servers: srvs,
		}},
		Upstream: &agd.Upstream{
			Server: netip.MustParseAddrPort("8.8.8.8:53"),
			FallbackServers: []netip.AddrPort{
				netip.MustParseAddrPort("1.1.1.1:53"),
			},
		},
	}

	svc, err := dnssvc.New(c)
	require.NoError(t, err)
	require.NotNil(t, svc)
}
