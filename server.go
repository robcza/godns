package main

import (
	"strconv"
	"time"

	"github.com/miekg/dns"
)

type DNSServer struct {
	host     string
	port     int
	rTimeout time.Duration
	wTimeout time.Duration
}

func (s *DNSServer) Run() {
	addr := s.host + ":" + strconv.Itoa(s.port)
	Handler := NewHandler()

	tcpHandler := dns.NewServeMux()
	tcpHandler.HandleFunc(".", Handler.DoTCP)

	udpHandler := dns.NewServeMux()
	udpHandler.HandleFunc(".", Handler.DoUDP)

	tcpServer := &dns.Server{Addr: addr,
		Net:          "tcp",
		Handler:      tcpHandler,
		ReadTimeout:  s.rTimeout,
		WriteTimeout: s.wTimeout}

	udpServer := &dns.Server{Addr: addr,
		Net:          "udp",
		Handler:      udpHandler,
		UDPSize:      settings.GODNS_UDP_PACKET_SIZE,
		ReadTimeout:  s.rTimeout,
		WriteTimeout: s.wTimeout}

	go s.start(udpServer)
	go s.start(tcpServer)

}

func (s *DNSServer) start(ds *dns.Server) {
	addr := s.host + ":" + strconv.Itoa(s.port)
	logger.Info("Start %s listener on %s", ds.Net, addr)
	err := ds.ListenAndServe()
	if err != nil {
		logger.Error("Start %s listener on %s failed:%s", ds.Net, addr, err.Error())
	}
}
