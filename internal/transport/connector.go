package transport

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// DialConfig holds connection options.
type DialConfig struct {
	Endpoint   string        // host:port
	TLS        bool          // enable TLS
	Timeout    time.Duration // dial timeout
	KeepAlive  time.Duration // TCP keepalive
	ReadIdle   time.Duration // optional read deadline extension per read
	RetryBacko time.Duration // base backoff between reconnect attempts
}

// Connector manages one persistent TCP connection.
type Connector struct {
	cfg    DialConfig
	mu     sync.RWMutex
	conn   net.Conn
	closed atomic.Bool

	onMsg  func([]byte) // callback on full ISO message (including MLI)
	onUp   func()
	onDown func(error)
}

func NewConnector(cfg DialConfig) *Connector { return &Connector{cfg: cfg} }

func (c *Connector) SetCallbacks(onMsg func([]byte), onUp func(), onDown func(error)) {
	c.onMsg, c.onUp, c.onDown = onMsg, onUp, onDown
}

// Start runs the connect/reconnect loop in a goroutine.
func (c *Connector) Start() { go c.loop() }

func (c *Connector) loop() {
	backoff := c.cfg.RetryBacko
	if backoff <= 0 {
		backoff = 2 * time.Second
	}

	for !c.closed.Load() {
		if err := c.dial(); err != nil {
			if c.onDown != nil {
				c.onDown(err)
			}
			time.Sleep(backoff)
			// Exponential-ish backoff with cap
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = c.cfg.RetryBacko
		if backoff <= 0 {
			backoff = 2 * time.Second
		}
		if c.onUp != nil {
			c.onUp()
		}
		c.readLoop()
		// readLoop exits on error or close
		if c.onDown != nil {
			c.onDown(errors.New("disconnected"))
		}
	}
}

func (c *Connector) dial() error {
	d := &net.Dialer{Timeout: c.cfg.Timeout, KeepAlive: c.cfg.KeepAlive}
	var (
		conn net.Conn
		err  error
	)
	if c.cfg.TLS {
		conn, err = tls.DialWithDialer(d, "tcp", c.cfg.Endpoint, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = d.Dial("tcp", c.cfg.Endpoint)
	}
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	return nil
}

func (c *Connector) readLoop() {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return
	}

	reader := bufio.NewReader(conn)
	for !c.closed.Load() {
		_ = conn.SetReadDeadline(time.Now().Add(c.cfg.ReadIdle))
		// Read MLI 2 bytes
		mliBytes := make([]byte, 2)
		if _, err := io.ReadFull(reader, mliBytes); err != nil {
			c.closeConn()
			return
		}
		mli := int(binary.BigEndian.Uint16(mliBytes))
		if mli <= 0 || mli > (64*1024) { // sanity
			c.closeConn()
			return
		}
		payload := make([]byte, mli)
		if _, err := io.ReadFull(reader, payload); err != nil {
			c.closeConn()
			return
		}
		full := append(mliBytes, payload...)
		if c.onMsg != nil {
			c.onMsg(full)
		}
	}
}

// Send writes a full wire message (already has MLI prefix).
func (c *Connector) Send(b []byte) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("not connected")
	}
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := conn.Write(b)
	return err
}

func (c *Connector) closeConn() {
	c.mu.Lock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
}

func (c *Connector) Close() {
	c.closed.Store(true)
	c.closeConn()
}
