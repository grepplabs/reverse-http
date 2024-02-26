package gost

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/grepplabs/reverse-http/pkg/logger"
)

type Service interface {
	Serve() error
	Addr() net.Addr
	Close() error
}

type serviceOptions struct {
	logger *logger.Logger
}

type ServiceOption func(opts *serviceOptions)

type defaultService struct {
	listener Listener
	handler  Handler
	options  serviceOptions
}

func NewService(ln Listener, h Handler, opts ...ServiceOption) Service {
	options := serviceOptions{
		logger: logger.GetInstance().WithFields(map[string]any{"kind": "service"}),
	}
	for _, opt := range opts {
		opt(&options)
	}
	s := &defaultService{
		listener: ln,
		handler:  h,
		options:  options,
	}
	return s
}

func (s *defaultService) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *defaultService) Serve() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var tempDelay time.Duration
	for {
		conn, e := s.listener.Accept()
		if e != nil {
			//nolint:staticcheck // SA1019 ignore this!
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if maxDelay := 5 * time.Second; tempDelay > maxDelay {
					tempDelay = maxDelay
				}
				s.options.logger.Warnf("accept: %v, retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			s.options.logger.Errorf("accept: %v", e)
			return e
		}

		if tempDelay > 0 {
			tempDelay = 0
		}
		go func() {
			if err := s.handler.Handle(ctx, conn); err != nil {
				s.options.logger.Error(err.Error())
			}
		}()
	}
}

func (s *defaultService) Close() error {
	if closer, ok := s.handler.(io.Closer); ok {
		_ = closer.Close()
	}
	return s.listener.Close()
}

func WithServiceLogger(logger *logger.Logger) ServiceOption {
	return func(opts *serviceOptions) {
		opts.logger = logger
	}
}
