// ----------------------
// NOTE: this was branched out from "github.com/gomodule/redigo/redis/pool.go"
// There are some modifications in code.
// ----------------------

// Copyright 2012 Gary Burd
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package pool

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

var nowFunc = time.Now // for testing

// ErrPoolExhausted is returned from a pool connection method (Do, Send,
// Receive, Flush, Err) when the maximum number of database connections in the
// pool has been reached.
var ErrPoolExhausted = errors.New("pool_exhausted")

var (
	errPoolClosed = errors.New("pool_closed")
	errConnClosed = errors.New("pool_connection_closed")
)

// Pool maintains a pool of connections. The application calls the Get method
// to get a connection from the pool and the connection's Close method to
// return the connection's resources to the pool.
//
// The following example shows how to use a pool in a web application. The
// application creates a pool at application startup and makes it available to
// request handlers using a package level variable. The pool configuration used
// here is an example, not a recommendation.
//
//	func newPool(addr string) io.Closer {
//	  return &pool.Pool{
//	    MaxIdle: 3,
//	    IdleTimeout: 240 * time.Second,
//	    Dial: func () (io.Closer, error) { return redis.Dial("tcp", addr) },
//	  }
//	}
//
//	var (
//	  pool *pool.Pool
//	  redisServer = flag.String("redisServer", ":6379", "")
//	)
//
//	func main() {
//	  flag.Parse()
//	  pool = newPool(*redisServer)
//	  ...
//	}
//
// A request handler gets a connection from the pool and closes the connection
// when the handler is done:
//
//	func serveHome(w http.ResponseWriter, r *http.Request) {
//	    conn := pool.Get()
//	    defer pool.Put(conn)
//	    ...
//	}
//
// Use the Dial function to authenticate connections with the AUTH command or
// select a database with the SELECT command:
//
//	pool := &pool.Pool{
//	  // Other pool configuration not shown in this example.
//	  Dial: func () (io.Closer, error) {
//	    c, err := redis.Dial("tcp", server)
//	    if err != nil {
//	      return nil, err
//	    }
//	    if _, err := c.Do("AUTH", password); err != nil {
//	      c.Close()
//	      return nil, err
//	    }
//	    if _, err := c.Do("SELECT", db); err != nil {
//	      c.Close()
//	      return nil, err
//	    }
//	    return c, nil
//	  },
//	}
//
// Use the TestOnBorrow function to check the health of an idle connection
// before the connection is returned to the application. This example PINGs
// connections that have been idle more than a minute:
//
//	pool := &pool.Pool{
//	  // Other pool configuration not shown in this example.
//	  TestOnBorrow: func(c io.Closer, t time.Time) error {
//	    if time.Since(t) < time.Minute {
//	      return nil
//	    }
//	    _, err := c.Do("PING")
//	    return err
//	  },
//	}
type Pool struct {
	// Dial is an application supplied function for creating and configuring a
	// connection.
	//
	// The connection returned from Dial must not be in a special state
	// (subscribed to pubsub channel, transaction started, ...).
	Dial func() (io.Closer, error)

	// TestOnBorrow is an optional application supplied function for checking
	// the health of an idle connection before the connection is used again by
	// the application. Argument t is the time that the connection was returned
	// to the pool. If the function returns an error, then the connection is
	// closed.
	TestOnBorrow func(c io.Closer, t time.Time) error

	// Maximum number of idle connections in the pool.
	MaxIdle int

	// Maximum number of connections allocated by the pool at a given time.
	// When zero, there is no limit on the number of connections in the pool.
	MaxActive int

	// Close connections after remaining idle for this duration. If the value
	// is zero, then idle connections are not closed. Applications should set
	// the timeout to a value less than the server's timeout.
	IdleTimeout time.Duration

	// If Wait is true and the pool is at the MaxActive limit, then Get() waits
	// for a connection to be returned to the pool before returning.
	Wait bool

	// Close connections older than this duration. If the value is zero, then
	// the pool does not close connections based on age.
	MaxConnLifetime time.Duration

	chInitialized uint32 // set to 1 when field ch is initialized

	mu          sync.Mutex    // mu protects the following fields
	closed      bool          // set to true when the pool is closed.
	active      int           // the number of open connections in the pool
	ch          chan struct{} // limits open connections when p.Wait is true
	idle        idleList      // idle connections
	poolConnMap map[io.Closer]*poolConn
}

// NewPool creates a new pool.
//
// Deprecated: Initialize the Pool directory as shown in the example.
//func NewPool(newFn func() (io.Closer, error), maxIdle int) *Pool {
//	return &Pool{Dial: newFn, MaxIdle: maxIdle}
//}

// Get gets a connection. The application must Put the returned connection to Pool.
// Note: This method may return nil.
func (p *Pool) Get() io.Closer {
	pc, err := p.get(nil)
	if err != nil {
		return nil
	}
	p.mu.Lock()
	if p.poolConnMap == nil {
		p.poolConnMap = make(map[io.Closer]*poolConn)
	}
	p.poolConnMap[pc.c] = pc
	p.mu.Unlock()
	return pc.c
}

// Put back the connection to Pool
func (p *Pool) Put(c io.Closer) error {
	p.mu.Lock()
	pc := p.poolConnMap[c]
	delete(p.poolConnMap, c)
	p.mu.Unlock()
	return p.put(pc, false)
}

// Stats contains pool statistics.
type Stats struct {
	// ActiveCount is the number of connections in the pool. The count includes
	// idle connections and connections in use.
	ActiveCount int
	// IdleCount is the number of idle connections in the pool.
	IdleCount int
}

// Stats returns pool's statistics.
func (p *Pool) Stats() Stats {
	p.mu.Lock()
	stats := Stats{
		ActiveCount: p.active,
		IdleCount:   p.idle.count,
	}
	p.mu.Unlock()

	return stats
}

// ActiveCount returns the number of connections in the pool. The count
// includes idle connections and connections in use.
func (p *Pool) ActiveCount() int {
	p.mu.Lock()
	active := p.active
	p.mu.Unlock()
	return active
}

// IdleCount returns the number of idle connections in the pool.
func (p *Pool) IdleCount() int {
	p.mu.Lock()
	idle := p.idle.count
	p.mu.Unlock()
	return idle
}

// Close releases the resources used by the pool.
func (p *Pool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.active -= p.idle.count
	pc := p.idle.front
	p.idle.count = 0
	p.idle.front, p.idle.back = nil, nil
	if p.ch != nil {
		close(p.ch)
	}
	p.mu.Unlock()
	for ; pc != nil; pc = pc.next {
		pc.c.Close()
	}
	return nil
}

func (p *Pool) lazyInit() {
	// Fast path.
	if atomic.LoadUint32(&p.chInitialized) == 1 {
		return
	}
	// Slow path.
	p.mu.Lock()
	if p.chInitialized == 0 {
		p.ch = make(chan struct{}, p.MaxActive)
		if p.closed {
			close(p.ch)
		} else {
			for i := 0; i < p.MaxActive; i++ {
				p.ch <- struct{}{}
			}
		}
		atomic.StoreUint32(&p.chInitialized, 1)
	}
	p.mu.Unlock()
}

// get prunes stale connections and returns a connection from the idle list or
// creates a new connection.
func (p *Pool) get(ctx interface {
	Done() <-chan struct{}
	Err() error
}) (*poolConn, error) {

	// Handle limit for p.Wait == true.
	if p.Wait && p.MaxActive > 0 {
		p.lazyInit()
		if ctx == nil {
			<-p.ch
		} else {
			select {
			case <-p.ch:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	p.mu.Lock()

	// Prune stale connections at the back of the idle list.
	if p.IdleTimeout > 0 {
		n := p.idle.count
		for i := 0; i < n && p.idle.back != nil && p.idle.back.t.Add(p.IdleTimeout).Before(nowFunc()); i++ {
			pc := p.idle.back
			p.idle.popBack()
			p.mu.Unlock()
			pc.c.Close()
			p.mu.Lock()
			p.active--
		}
	}

	// Get idle connection from the front of idle list.
	for p.idle.front != nil {
		pc := p.idle.front
		p.idle.popFront()
		p.mu.Unlock()
		if (p.TestOnBorrow == nil || p.TestOnBorrow(pc.c, pc.t) == nil) &&
			(p.MaxConnLifetime == 0 || nowFunc().Sub(pc.created) < p.MaxConnLifetime) {
			return pc, nil
		}
		pc.c.Close()
		p.mu.Lock()
		p.active--
	}

	// Check for pool closed before dialing a new connection.
	if p.closed {
		p.mu.Unlock()
		return nil, errors.New("redigo: get on closed pool")
	}

	// Handle limit for p.Wait == false.
	if !p.Wait && p.MaxActive > 0 && p.active >= p.MaxActive {
		p.mu.Unlock()
		return nil, ErrPoolExhausted
	}

	p.active++
	p.mu.Unlock()
	c, err := p.Dial()
	if err != nil {
		c = nil
		p.mu.Lock()
		p.active--
		if p.ch != nil && !p.closed {
			p.ch <- struct{}{}
		}
		p.mu.Unlock()
	}
	return &poolConn{c: c, created: nowFunc()}, err
}

func (p *Pool) put(pc *poolConn, forceClose bool) error {
	p.mu.Lock()
	if !p.closed && !forceClose {
		pc.t = nowFunc()
		p.idle.pushFront(pc)
		if p.idle.count > p.MaxIdle {
			pc = p.idle.back
			p.idle.popBack()
		} else {
			pc = nil
		}
	}

	if pc != nil {
		p.mu.Unlock()
		pc.c.Close()
		p.mu.Lock()
		p.active--
	}

	if p.ch != nil && !p.closed {
		p.ch <- struct{}{}
	}
	p.mu.Unlock()
	return nil
}

type idleList struct {
	count       int
	front, back *poolConn
}

type poolConn struct {
	c          io.Closer
	t          time.Time
	created    time.Time
	next, prev *poolConn
}

func (l *idleList) pushFront(pc *poolConn) {
	pc.next = l.front
	pc.prev = nil
	if l.count == 0 {
		l.back = pc
	} else {
		l.front.prev = pc
	}
	l.front = pc
	l.count++
	return
}

func (l *idleList) popFront() {
	pc := l.front
	l.count--
	if l.count == 0 {
		l.front, l.back = nil, nil
	} else {
		pc.next.prev = nil
		l.front = pc.next
	}
	pc.next, pc.prev = nil, nil
}

func (l *idleList) popBack() {
	pc := l.back
	l.count--
	if l.count == 0 {
		l.front, l.back = nil, nil
	} else {
		pc.prev.next = nil
		l.back = pc.prev
	}
	pc.next, pc.prev = nil, nil
}
