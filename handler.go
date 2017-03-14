package main

import (
	"net"
	"time"

	"github.com/miekg/dns"
)

const (
	notIPQuery = 0
	_IP4Query  = 4
	_IP6Query  = 6
)

type Question struct {
	qname  string
	qtype  string
	qclass string
}

func (q *Question) String() string {
	return q.qname + " " + q.qclass + " " + q.qtype
}

type GODNSHandler struct {
	resolver      *Resolver
	oraculumCache Cache
}

func NewHandler() *GODNSHandler {

	var (
		clientConfig  *dns.ClientConfig
		resolver      *Resolver
		oraculumCache Cache
	)

	clientConfig, err := dns.ClientConfigFromFile(settings.RESOLV_CONF_FILE)
	if err != nil {
		logger.Error(":%s is not a valid resolv.conf file\n", settings.RESOLV_CONF_FILE)
		logger.Error(err.Error())
		panic(err)
	}
	clientConfig.Timeout = settings.BACKEND_RESOLVER_RW_TIMEOUT
	resolver = &Resolver{clientConfig}

	switch settings.ORACULUM_CACHE_BACKEND {
	// TODO might have other implementations...
	case "memory":
		oraculumCache = &MemoryCache{
			Backend:  make(map[string]Data),
			Expire:   time.Duration(settings.ORACULUM_CACHE_EXPIRE) * time.Millisecond,
			Maxcount: settings.ORACULUM_CACHE_MAXCOUNT,
		}
	default:
		logger.Error("Invalid cache backend %s", settings.ORACULUM_CACHE_BACKEND)
		panic("Invalid cache backend")
	}

	resolver.init()

	FillTestData()
	go StartCoreClient(oraculumCache)

	return &GODNSHandler{resolver, oraculumCache}
}

func (h *GODNSHandler) do(Net string, w dns.ResponseWriter, req *dns.Msg) {
	q := req.Question[0]
	Q := Question{UnFqdn(q.Name), dns.TypeToString[q.Qtype], dns.ClassToString[q.Qclass]}

	var remote net.IP
	if Net == "tcp" {
		remote = w.RemoteAddr().(*net.TCPAddr).IP
	} else {
		remote = w.RemoteAddr().(*net.UDPAddr).IP
	}
	logger.Debug("%s lookup %s", remote, Q.String())
	logger.Debug("Question: %s", Q.String())
	logger.Debug("w.RemoteAddr().String(): %s", w.RemoteAddr().String())

	mesg, err := h.resolver.Lookup(Net, req, w.RemoteAddr(), h.oraculumCache)

	if err != nil {
		logger.Warn("Resolve query error %s", err)
		dns.HandleFailed(w, req)
		return
	}

	w.WriteMsg(mesg)
}

func (h *GODNSHandler) DoTCP(w dns.ResponseWriter, req *dns.Msg) {
	h.do("tcp", w, req)
}

func (h *GODNSHandler) DoUDP(w dns.ResponseWriter, req *dns.Msg) {
	h.do("udp", w, req)
}

func (h *GODNSHandler) isIPQuery(q dns.Question) int {
	if q.Qclass != dns.ClassINET {
		return notIPQuery
	}

	switch q.Qtype {
	case dns.TypeA:
		return _IP4Query
	case dns.TypeAAAA:
		return _IP6Query
	default:
		return notIPQuery
	}
}

func UnFqdn(s string) string {
	if dns.IsFqdn(s) {
		return s[:len(s)-1]
	}
	return s
}
