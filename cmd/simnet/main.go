package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"net"
	"time"

	"go-payment-gateway/internal/iso8583"
)

func main() {
	listen := flag.String("listen", ":5001", "listen addr")
	flag.Parse()

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("simnet listening on %s", *listen)
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handle(c)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	log.Printf("client %s connected", conn.RemoteAddr())
	reader := bufio.NewReader(conn)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		mliBytes := make([]byte, 2)
		if _, err := io.ReadFull(reader, mliBytes); err != nil {
			log.Printf("read mli: %v", err)
			return
		}
		mli := int(binary.BigEndian.Uint16(mliBytes))
		payload := make([]byte, mli)
		if _, err := io.ReadFull(reader, payload); err != nil {
			log.Printf("read payload: %v", err)
			return
		}

		full := append(mliBytes, payload...)
		msg, err := iso8583.Unpack(full)
		if err != nil {
			log.Printf("unpack: %v", err)
			continue
		}
		log.Printf("RX %s fields=%v", msg.MTI, msg.Fields)

		if msg.MTI == "0800" {
			// Build 0810 response echoing STAN, DE70
			r := iso8583.New("0810")
			if v, ok := msg.Get(11); ok {
				r.Set(11, v)
			}
			r.Set(7, time.Now().UTC().Format("0102150405"))
			if v, ok := msg.Get(70); ok {
				r.Set(70, v)
			}
			b, err := r.Pack()
			if err != nil {
				log.Printf("pack resp: %v", err)
				continue
			}
			if _, err := conn.Write(b); err != nil {
				log.Printf("write resp: %v", err)
				return
			}
			log.Printf("TX 0810 echo resp STAN=%v", msg.Fields[11])
		}
	}
}
