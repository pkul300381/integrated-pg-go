package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"go-payment-gateway/internal/admin"
	"go-payment-gateway/internal/iso8583"
	"go-payment-gateway/internal/transport"
)

func main() {
	var (
		endpoint     = flag.String("endpoint", "127.0.0.1:5001", "upstream host:port")
		tlsEnable    = flag.Bool("tls", false, "enable TLS to upstream")
		adminAddr    = flag.String("admin", ":8080", "admin http listen addr")
		echoInterval = flag.Duration("echo-interval", 15*time.Second, "period between 0800 echo tests")
	)
	flag.Parse()

	st := &admin.State{Started: time.Now()}
	st.Conn.Endpoint = *endpoint

	conn := transport.NewConnector(transport.DialConfig{
		Endpoint:  *endpoint,
		TLS:       *tlsEnable,
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
		ReadIdle:  60 * time.Second,
		RetryBacko: 2 * time.Second,
	})

	var stan int64 = time.Now().Unix() % 1000000 // seed

	conn.SetCallbacks(
		func(msg []byte) {
			atomic.AddUint64(&st.Conn.RxMsgs, 1)
			m, err := iso8583.Unpack(msg)
			if err != nil {
				log.Printf("RX unpack error: %v", err)
				atomic.AddUint64(&st.Conn.Errs, 1)
				return
			}
			if iso8583.IsEchoResponse(m) {
				log.Printf("RX 0810 echo response, STAN=%06d", iso8583.MustParseSTAN(m))
			} else {
				log.Printf("RX %s (not handled in skeleton)", m.MTI)
			}
		},
		func() {
			st.Conn.Up = true
			st.Conn.LastChangeTs = time.Now()
			log.Printf("connected to %s (tls=%v)", *endpoint, *tlsEnable)
		},
		func(err error) {
			st.Conn.Up = false
			st.Conn.LastChangeTs = time.Now()
			log.Printf("disconnected from %s: %v", *endpoint, err)
		},
	)

	conn.Start()
	adm := admin.Serve(*adminAddr, st)

	// periodic echo sender
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(*echoInterval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if !st.Conn.Up { continue }
				s := int(atomic.AddInt64(&stan, 1))
				m := iso8583.NewEchoRequest(s)
				b, err := m.Pack()
				if err != nil {
					log.Printf("pack error: %v", err)
					atomic.AddUint64(&st.Conn.Errs, 1)
					continue
				}
				if err := conn.Send(b); err != nil {
					log.Printf("TX error: %v", err)
					atomic.AddUint64(&st.Conn.Errs, 1)
					continue
				}
				st.Conn.LastEchoSTAN = s
				st.Conn.LastEchoAt = time.Now()
				atomic.AddUint64(&st.Conn.TxMsgs, 1)
				log.Printf("TX 0800 echo request, STAN=%06d", s)
			case <-stop:
				return
			}
		}
	}()

	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	close(stop)
	conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = adm.Shutdown(ctx)
	log.Println("gateway stopped")
}
