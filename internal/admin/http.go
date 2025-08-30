package admin

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

type ConnStat struct {
	Endpoint     string        `json:"endpoint"`
	Up           bool          `json:"up"`
	LastChangeTs time.Time     `json:"last_change_ts"`
	LastEchoSTAN int           `json:"last_echo_stan"`
	LastEchoAt   time.Time     `json:"last_echo_at"`
	RxMsgs       uint64        `json:"rx_msgs"`
	TxMsgs       uint64        `json:"tx_msgs"`
	Errs         uint64        `json:"errs"`
}

type State struct {
	Started time.Time `json:"started"`
	// For demo we support a single upstream
	Conn ConnStat `json:"conn"`
}

func Serve(addr string, st *State) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"uptime": time.Since(st.Started).String(),
		})
	})

	mux.HandleFunc("/connections", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(st.Conn)
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "gateway_uptime_seconds %d\n", int(time.Since(st.Started).Seconds()))
		fmt.Fprintf(w, "gateway_tx_messages_total %d\n", atomic.LoadUint64(&st.Conn.TxMsgs))
		fmt.Fprintf(w, "gateway_rx_messages_total %d\n", atomic.LoadUint64(&st.Conn.RxMsgs))
		fmt.Fprintf(w, "gateway_errors_total %d\n", atomic.LoadUint64(&st.Conn.Errs))
		if st.Conn.Up { fmt.Fprintln(w, "gateway_up 1") } else { fmt.Fprintln(w, "gateway_up 0") }
	})

	s := &http.Server{Addr: addr, Handler: mux}
	go func() {
		log.Printf("admin listening on %s", addr)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("admin server error: %v", err)
		}
	}()
	return s
}
