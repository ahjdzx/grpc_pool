// Package grpc_pool provides a gRPC connection pool.
package grpc_pool

import (
	"sync"
	"time"

	"google.golang.org/grpc"
)

// Pool represents a gRPC connection pool.
type Pool struct {
	size int
	ttl  int64
	sync.Mutex
	conns map[string][]*Conn
}

// Conn wrapped a gRPC connection with create time.
type Conn struct {
	*grpc.ClientConn
	createdAt int64
}

// NewPool create a gRPC pool.
func NewPool(size int, ttl time.Duration) *Pool {
	return &Pool{
		size:  size,
		ttl:   int64(ttl.Seconds()),
		conns: make(map[string][]*Conn),
	}
}

// GetConn return an available gRPC connection.
func (p *Pool) GetConn(addr string, opts ...grpc.DialOption) (*Conn, error) {
	p.Lock()
	conns := p.conns[addr]
	now := time.Now().Unix()

	for len(conns) > 0 {
		conn := conns[len(conns)-1]
		conns = conns[:len(conns)-1]
		p.conns[addr] = conns

		// if conn is tool old then close it and move on
		if d := now - conn.createdAt; d > p.ttl {
			conn.Close()
			continue
		}

		// we got a available one
		p.Unlock()
		return conn, nil
	}

	p.Unlock()

	// create new conn
	c, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}

	return &Conn{
		ClientConn: c,
		createdAt:  time.Now().Unix(),
	}, nil
}

// Release a gRPC connection to pool.
func (p *Pool) Release(addr string, conn *Conn, err error) {
	if err != nil {
		conn.Close()
		return
	}

	p.Lock()
	conns := p.conns[addr]
	if len(conns) >= p.size {
		p.Unlock()
		conn.Close()
		return
	}

	p.conns[addr] = append(conns, conn)
	p.Unlock()
}

// Close the pool.
func (p *Pool) Close() {
	p.Lock()
	defer p.Unlock()

	for _, conns := range p.conns {
		for _, conn := range conns {
			conn.Close()
		}
	}
}
