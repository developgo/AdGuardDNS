package dnsserver

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/AdguardTeam/golibs/log"
)

// ConfigTLS is a struct that needs to be passed to NewServerTLS to
// initialize a new ServerTLS instance.
type ConfigTLS struct {
	ConfigDNS

	// TLSConfig is the TLS configuration for TLS.
	TLSConfig *tls.Config
}

// ServerTLS implements a DNS-over-TLS server.
// Note that it heavily relies on ServerDNS.
type ServerTLS struct {
	*ServerDNS

	conf ConfigTLS
}

// type check
var _ Server = (*ServerTLS)(nil)

// NewServerTLS creates a new ServerTLS instance.
func NewServerTLS(conf ConfigTLS) (s *ServerTLS) {
	srv := newServerDNS(conf.ConfigDNS)
	s = &ServerTLS{
		ServerDNS: srv,
		conf:      conf,
	}

	return s
}

// Start starts the TLS listener and starts processing queries.
func (s *ServerTLS) Start(ctx context.Context) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// TODO(ameshkov): Consider only setting s.started to true once the
	// listeners are up.
	if s.started {
		return ErrServerAlreadyStarted
	}
	s.started = true

	log.Info("[%s]: Starting the server", s.name)

	ctx = ContextWithServerInfo(ctx, ServerInfo{
		Name:  s.name,
		Addr:  s.addr,
		Proto: s.proto,
	})

	// Start listening to TCP on the specified addr
	err = s.listenTLS(ctx)
	if err != nil {
		return err
	}

	// Start the TLS server loop
	if s.tcpListener != nil {
		go s.startServeTCP(ctx)
	}

	log.Info("[%s]: Server has been started", s.Name())

	return nil
}

// startServeTCP starts the TCP listen loop and handles errors if any.
func (s *ServerTLS) startServeTCP(ctx context.Context) {
	// We do not recover from panics here since if this go routine panics
	// the application won't be able to continue listening to DoT
	defer s.handlePanicAndExit(ctx)

	log.Info("[%s]: Start listening to tls://%s", s.Name(), s.Addr())
	err := s.serveTCP(ctx, s.tcpListener)
	if err != nil {
		log.Info("[%s]: Finished listening to tls://%s due to %v", s.Name(), s.Addr(), err)
	}
}

// listenTLS creates the TLS listener for the ServerTLS.addr.
func (s *ServerTLS) listenTLS(ctx context.Context) (err error) {
	var l net.Listener
	l, err = listenTCP(ctx, s.addr)
	if err != nil {
		return err
	}

	s.tcpListener = newTLSListener(l, s.conf.TLSConfig)

	return nil
}
