// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package utils

import (
	"io"
	"log"
	"net"
	"sync"
)

// HandlerFunc defines the interface of a function that should be served by a Unix Domain Socket server
type HandlerFunc func(io.Reader, io.WriteCloser)

// UnixDomainSocketServer implements a Unix Domain Socket server
type UnixDomainSocketServer struct {
	listener net.Listener
	quit     chan interface{}
	wg       sync.WaitGroup
	handler  HandlerFunc
}

// NewUnixDomainSocketServer returns a new Unix Domain Socket server.
// The parameters define the server address and the handler func it serves
func NewUnixDomainSocketServer(addr string, handler HandlerFunc) (*UnixDomainSocketServer, error) {
	l, err := net.Listen("unix", addr)
	if err != nil {
		return nil, err
	}
	s := &UnixDomainSocketServer{
		quit:     make(chan interface{}),
		listener: l,
		handler:  handler,
	}
	return s, nil
}

// Start starts the server goroutine
func (s *UnixDomainSocketServer) Start() {
	s.wg.Add(1)
	go s.serve()
}

func (s *UnixDomainSocketServer) serve() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Println("accept error", err)
			}
		} else {
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				s.handler(conn, conn)
			}()
		}
	}
}

// Stop stops the server goroutine
func (s *UnixDomainSocketServer) Stop() {
	close(s.quit)
	if err := s.listener.Close(); err != nil {
		println(err)
	}
	s.wg.Wait()
}
