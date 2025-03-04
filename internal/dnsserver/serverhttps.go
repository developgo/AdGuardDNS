package dnsserver

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/miekg/dns"
	"golang.org/x/net/http2"
)

const (
	// MimeTypeDoH is a Content-Type that DoH wireformat requests and responses
	// must use.
	MimeTypeDoH = "application/dns-message"
	// MimeTypeJSON is a Content-Type that DoH JSON requests and responses
	// must use.
	MimeTypeJSON = "application/x-javascript"
	// PathDoH is a relative path we use to accept DoH wireformat requests.
	PathDoH = "/dns-query"
	// PathJSON is a relative path we use to accept DoH JSON requests.
	PathJSON = "/resolve"

	httpReadTimeout  = 5 * time.Second
	httpWriteTimeout = 5 * time.Second
	httpIdleTimeout  = 120 * time.Second
)

// nextProtoDoH is a list of ALPN that we would add by default to the server
// *tls.Config if no NextProto is specified there.  Note, that with this order,
// we prioritize HTTP/2 over HTTP/1.1.
var nextProtoDoH = []string{http2.NextProtoTLS, "http/1.1"}

// ConfigHTTPS is a struct that needs to be passed to NewServerHTTPS to
// initialize a new ServerHTTPS instance.
type ConfigHTTPS struct {
	ConfigBase

	// TLSConfig is the TLS configuration for HTTPS.
	// if not set, we'll run a plain HTTP server.
	TLSConfig *tls.Config

	// NonDNSHandler handles requests with the path not equal to /dns-query.
	// If it is empty, the server will return 404 for requests like that.
	NonDNSHandler http.Handler
}

// ServerHTTPS is a DoH server implementation. It supports both DNS Wireformat
// and DNS JSON format.  Regular DoH (wireformat) will be available at the
// /dns-query location.  JSON format will be available at the "/resolve"
// location.
type ServerHTTPS struct {
	*ServerBase

	httpServer *http.Server

	conf ConfigHTTPS
}

// type check
var _ Server = (*ServerHTTPS)(nil)

// NewServerHTTPS creates a new ServerHTTPS instance.
func NewServerHTTPS(conf ConfigHTTPS) (s *ServerHTTPS) {
	tlsConfig := conf.TLSConfig
	if tlsConfig != nil && len(tlsConfig.NextProtos) == 0 {
		tlsConfig.NextProtos = nextProtoDoH
	}

	s = &ServerHTTPS{
		ServerBase: newServerBase(conf.ConfigBase),
		conf:       conf,
	}

	return s
}

// Start starts the ServerHTTPS.
func (s *ServerHTTPS) Start(ctx context.Context) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.started {
		return ErrServerAlreadyStarted
	}
	s.started = true

	log.Info("[%s]: Starting the server", s.addr)

	ctx = ContextWithServerInfo(ctx, ServerInfo{
		Name:  s.name,
		Addr:  s.addr,
		Proto: s.proto,
	})

	// Start the TLS or TCP listener
	err = s.listenTLS(ctx)
	if err != nil {
		return err
	}

	// Prepare and run the HTTP server
	handler := &httpHandler{
		srv:      s,
		listener: s.tcpListener,
	}

	s.httpServer = &http.Server{
		Handler:           handler,
		ReadTimeout:       httpReadTimeout,
		ReadHeaderTimeout: httpReadTimeout,
		WriteTimeout:      httpWriteTimeout,
		IdleTimeout:       httpIdleTimeout,
		ErrorLog:          log.StdLog("dnsserver/serverhttps: "+s.name, log.DEBUG),
	}

	// Start the server thread
	s.wg.Add(1)
	go s.serveHTTPS(ctx, s.httpServer, s.tcpListener)
	log.Info("[%s]: Server has been started", s.Name())

	return nil
}

// Shutdown stops the ServerHTTPS.
func (s *ServerHTTPS) Shutdown(ctx context.Context) (err error) {
	log.Info("[%s]: Stopping the server", s.Name())
	err = s.shutdown(ctx)
	if err != nil {
		log.Info("[%s]: Failed to shutdown: %v", s.Name(), err)

		return err
	}

	err = s.waitShutdown(ctx)
	log.Info("[%s]: Finished stopping the server", s.Name())

	return err
}

// shutdown marks the server as stopped and closes active listeners.
func (s *ServerHTTPS) shutdown(ctx context.Context) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if !s.started {
		return ErrServerNotStarted
	}

	s.started = false

	// First step, close the active listener right away
	s.closeListeners()

	// Second, shutdown the HTTP server
	err = s.httpServer.Shutdown(ctx)
	if err != nil {
		log.Debug("[%s]: http server shutdown: %v", s.Name(), err)
	}

	return nil
}

// serveHTTPS is a worker goroutine that serves HTTP.
func (s *ServerHTTPS) serveHTTPS(ctx context.Context, hs *http.Server, l net.Listener) {
	defer s.wg.Done()

	// Do not recover from panics here since if this goroutine panics, the
	// application won't be able to continue listening to DoH.
	defer s.handlePanicAndExit(ctx)

	scheme := "https"
	if s.conf.TLSConfig == nil {
		scheme = "http"
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   s.addr,
	}
	log.Info("[%s]: Start listening to %s", s.name, u)

	err := hs.Serve(l)
	if err != nil {
		log.Info("[%s]: Finished listening to %s due to %v", s.name, u, err)
	}
}

// httpHandler is a helper structure that implements http.Handler
// and holds pointers to ServerHTTPS, net.Listener.
type httpHandler struct {
	srv      *ServerHTTPS
	listener net.Listener
}

// type check
var _ http.Handler = (*httpHandler)(nil)

// ServeHTTP implements the http.Handler interface for *httpHandler.  It reads
// the DNS data from the request, resolves it, and sends a response.
//
// NOTE: r.Context() is only used to control cancellation.  To add values to the
// context, use the BaseContext of this handler's ServerHTTPS.
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := h.srv.requestContext()
	if dl, ok := r.Context().Deadline(); ok {
		var cancel func()
		ctx, cancel = context.WithDeadline(ctx, dl)
		defer cancel()
	}

	defer h.srv.handlePanicAndRecover(ctx)

	log.Debug("Received a request to %s", r.URL)

	isDNS, _, _ := isDoH(r)
	if isDNS {
		h.serveDoH(ctx, w, r)

		return
	}

	if h.srv.conf.NonDNSHandler != nil {
		h.srv.conf.NonDNSHandler.ServeHTTP(w, r)
	} else {
		h.srv.metrics.OnInvalidMsg(ctx)
		http.Error(w, "", http.StatusNotFound)
	}
}

// serveDoH processes the incoming DNS message and writes the response back to
// the client.
func (h *httpHandler) serveDoH(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	m, err := httpRequestToMsg(r)
	if err != nil {
		log.Debug("Failed to convert request to a DNS message: %v", err)
		h.srv.metrics.OnInvalidMsg(ctx)
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	rAddr := remoteAddr(r)
	lAddr := h.listener.Addr()
	rw := NewNonWriterResponseWriter(lAddr, rAddr)

	ci := ClientInfo{
		URL: netutil.CloneURL(r.URL),
	}
	if r.TLS != nil {
		ci.TLSServerName = strings.ToLower(r.TLS.ServerName)
	}
	ctx = ContextWithClientInfo(ctx, ci)

	// Serve the query
	written := h.srv.serveDNS(ctx, m, rw)

	// If no response were written, indicate it via an internal server error.
	if !written {
		log.Debug("No response has been written by the handler")
		http.Error(w, "No response", http.StatusInternalServerError)

		return
	}

	// Get the response that has been written.
	resp := rw.Msg()
	req := rw.req

	// Write the response to the client
	err = h.writeResponse(req, resp, r, w)
	if err != nil {
		log.Debug("[%d] Failed to write HTTP response: %v", req.Id, err)

		// Try writing an error response just in case.
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

// writeResponse writes the actual DNS response to the client and takes care of
// the response serialization, i.e. writes different content depending on the
// requested mime type (wireformat or JSON).
func (h *httpHandler) writeResponse(
	req *dns.Msg,
	resp *dns.Msg,
	r *http.Request,
	w http.ResponseWriter,
) (err error) {
	// normalize the response
	normalize(NetworkTCP, req, resp)

	isDNS, _, ct := isDoH(r)
	if !isDNS {
		return fmt.Errorf("invalid request path: %s", r.URL.Path)
	}

	var buf []byte
	switch ct {
	case MimeTypeDoH:
		buf, err = resp.Pack()
		w.Header().Set("Content-Type", MimeTypeDoH)
	case MimeTypeJSON:
		buf, err = dnsMsgToJSON(resp)
		w.Header().Set("Content-Type", MimeTypeJSON)
	default:
		return fmt.Errorf("invalid content type: %s", ct)
	}

	if err != nil {
		return err
	}

	// From RFC8484, Section 5.1:
	// DoH servers SHOULD assign an explicit HTTP freshness
	// lifetime (see Section 4.2 of [RFC7234]) so that the DoH client is
	// more likely to use fresh DNS data.
	maxAge := minimalTTL(resp)
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%f", maxAge.Seconds()))
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	w.WriteHeader(http.StatusOK)

	// Write the actual response
	log.Debug("[%d] Writing HTTP response", req.Id)
	_, err = w.Write(buf)
	return err
}

// listenTCP starts the TCP/TLS listener.
func (s *ServerHTTPS) listenTLS(ctx context.Context) (err error) {
	l, err := listenTCP(ctx, s.addr)
	if err != nil {
		return err
	}

	if s.conf.TLSConfig != nil {
		l = tls.NewListener(l, s.conf.TLSConfig)
	}

	s.tcpListener = l

	return nil
}

// httpRequestToMsg reads the DNS message from http.Request.
func httpRequestToMsg(req *http.Request) (b []byte, err error) {
	_, isJSON, _ := isDoH(req)
	if isJSON {
		return httpRequestToMsgJSON(req)
	}

	switch req.Method {
	case http.MethodGet:
		return httpRequestToMsgGet(req)
	case http.MethodPost:
		return httpRequestToMsgPost(req)
	default:
		return nil, fmt.Errorf("method not allowed: %s", req.Method)
	}
}

// httpRequestToMsgPost extracts the DNS message from a request body.
func httpRequestToMsgPost(req *http.Request) (b []byte, err error) {
	buf, err := io.ReadAll(req.Body)
	defer log.OnCloserError(req.Body, log.DEBUG)
	return buf, err
}

// httpRequestToMsgGet extracts the DNS message from a GET request.
func httpRequestToMsgGet(req *http.Request) (b []byte, err error) {
	values := req.URL.Query()
	b64, ok := values["dns"]
	if !ok {
		return nil, fmt.Errorf("no 'dns' query parameter found")
	}
	if len(b64) != 1 {
		return nil, fmt.Errorf("multiple 'dns' query values found")
	}

	return base64.RawURLEncoding.DecodeString(b64[0])
}

// remoteAddr gets HTTP request's remote address.
//
// TODO(a.garipov): Add trusted proxies and real IP extraction logic.  Perhaps
// just copy from module dnsproxy if that one fits.  Also, perhaps make that
// logic pluggable and put it into a new package in module golibs.
func remoteAddr(r *http.Request) (addr *net.TCPAddr) {
	// Consider that the http.Request.RemoteAddr documentation is correct and
	// that it is always a valid ip:port value.  Panic if it isn't so.
	ipStr, port, err := netutil.SplitHostPort(r.RemoteAddr)
	if err != nil {
		panic(err)
	}

	ip, err := netutil.ParseIP(ipStr)
	if err != nil {
		panic(err)
	}

	// TODO(a.garipov): Return a *net.UDPAddr when we're HTTP/3-ready.
	return &net.TCPAddr{IP: ip, Port: port}
}

// isDoH returns true if r.URL.Path contains DNS-over-HTTP paths, and also what
// content type is desired by the user.  isJSON is true if the user uses the
// JSON API.  ct can be either MimeTypeDoH or MimeTypeJSON.
func isDoH(r *http.Request) (ok, isJSON bool, ct string) {
	parts := strings.Split(path.Clean(r.URL.Path), "/")
	if parts[0] == "" {
		parts = parts[1:]
	}

	switch {
	case parts[0] == "":
		return false, false, ""
	case strings.HasSuffix(PathDoH, parts[0]):
		return true, false, MimeTypeDoH
	case strings.HasSuffix(PathJSON, parts[0]):
		desiredCt := r.URL.Query().Get("ct")
		if desiredCt == MimeTypeDoH {
			return true, true, MimeTypeDoH
		}

		return true, true, MimeTypeJSON
	default:
		return false, false, ""
	}
}
