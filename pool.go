// Package grpc_pool provides a gRPC connection pool.
package grpc_pool

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

var ErrConnShutdown = errors.New("grpc conn shutdown")

// Pool represents a gRPC connection pool.
type Pool struct {
	size int64
	next int64

	sync.Mutex
	conns    []*grpc.ClientConn
	dialFunc DialFunc
}

type DialFunc func() (*grpc.ClientConn, error)

// NewPool create a gRPC pool.
func NewPool(size int, dialFunc DialFunc) (*Pool, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid pool size")
	}

	p := &Pool{
		size:     int64(size),
		conns:    make([]*grpc.ClientConn, size),
		dialFunc: dialFunc,
	}

	for i := range p.conns {
		conn, err := p.dialFunc()
		if err != nil {
			return nil, err
		}
		p.conns[i] = conn
	}

	return p, nil
}

func (p *Pool) checkState(conn *grpc.ClientConn) error {
	state := conn.GetState()
	switch state {
	case connectivity.TransientFailure, connectivity.Shutdown:
		return ErrConnShutdown
	}

	return nil
}

// GetConn return an available gRPC connection.
func (p *Pool) GetConn() (*grpc.ClientConn, error) {
	var (
		idx  int64
		next int64

		err error
	)

	next = atomic.AddInt64(&p.next, 1)
	idx = next % p.size
	conn := p.conns[idx]
	if conn != nil && p.checkState(conn) == nil {
		return conn, nil
	}

	// gc old conn
	if conn != nil {
		conn.Close()
	}

	p.Lock()
	defer p.Unlock()

	// double check, already inited
	conn = p.conns[idx]
	if conn != nil && p.checkState(conn) == nil {
		return conn, nil
	}

	conn, err = p.dialFunc()
	if err != nil {
		return nil, err
	}

	p.conns[idx] = conn
	return conn, nil
}

// Close the pool.
func (p *Pool) Close() {
	p.Lock()
	defer p.Unlock()

	for _, conn := range p.conns {
		if conn != nil {
			conn.Close()
		}
	}
}
