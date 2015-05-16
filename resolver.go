package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"bytes"

	"github.com/miekg/dns"
	"sort"
)

type ResolvError struct {
	qname, net  string
	nameservers []string
}

func (e ResolvError) Error() string {
	errmsg := fmt.Sprintf("%s resolv failed on %s (%s)", e.qname, strings.Join(e.nameservers, "; "), e.net)
	return errmsg
}

type Resolver struct {
	config *dns.ClientConfig
}

// Dummy playground
func sinkByHostname(qname string) (bool){
	dummyTestHostnames := []string{"google.com", "root.cz", "seznam.cz", "youtube.com"}
	//OMG...
	sort.Strings(dummyTestHostnames)
	if(sort.SearchStrings(dummyTestHostnames, qname) == len(dummyTestHostnames)){
		return false
	}
	return true
}

// Dummy playground
func sinkByIPAddress(msg *dns.Msg) (bool) {

	/*if !aRecord.Equal(ip) {
			t.Fatalf("IP %q does not match registered IP %q", aRecord, ip)
		}*/

	dummyTestIPAddresses := []string{"81.19.0.120"}
	sort.Strings(dummyTestIPAddresses)
	//TODO: This dummy snippet is just for testing; We will search redis/use REST API presently.
	for _, element := range msg.Answer {
		Info("\nKARMTAG: RR Element: %s\n", element)
		if(sort.SearchStrings(dummyTestIPAddresses, element.String()) !=  len(dummyTestIPAddresses)){
			return true
		}
	}
	return false
}

// Dummy playground
func sendToSinkhole(msg *dns.Msg, qname string) {
	//TODO: Isn't it a clumsy concatenation?
	var buffer bytes.Buffer
	buffer.WriteString(qname)
	buffer.WriteString("	")
	buffer.WriteString("300	")
	buffer.WriteString("IN	")
	buffer.WriteString("A	")
	buffer.WriteString("127.0.0.1")
	//Sink only the first record
	//msg.Answer[0], _ = dns.NewRR(buffer.String())
	//Sink all records:
	sinkRecord, _ := dns.NewRR(buffer.String())
	msg.Answer = []dns.RR{sinkRecord}

	Info("\n KARMTAG: A record: %s", msg.Answer[0].(*dns.A).A)
	return
}

// Lookup will ask each nameserver in top-to-bottom fashion, starting a new request
// in every second, and return as early as possible (have an answer).
// It returns an error if no request has succeeded.
func (r *Resolver) Lookup(net string, req *dns.Msg) (message *dns.Msg, err error) {
	c := &dns.Client{
		Net:          net,
		ReadTimeout:  r.Timeout(),
		WriteTimeout: r.Timeout(),
	}

	qname := req.Question[0].Name

	res := make(chan *dns.Msg, 1)
	var wg sync.WaitGroup
	L := func(nameserver string) {
		defer wg.Done()
		r, rtt, err := c.Exchange(req, nameserver)
		if err != nil {
			Debug("%s socket error on %s", qname, nameserver)
			Debug("error:%s", err.Error())
			return
		}
		if r != nil && r.Rcode != dns.RcodeSuccess {
			Debug("%s failed to get an valid answer on %s", qname, nameserver)
			return
		}
		Info("\n KARMTAG: %s resolv on %s (%s) ttl: %d\n", UnFqdn(qname), nameserver, net, rtt)
		select {
		case res <- r:
		default:
		}
	}

	ticker := time.NewTicker(time.Duration(settings.ResolvConfig.Interval) * time.Millisecond)
	defer ticker.Stop()
	// Start lookup on each nameserver top-down, in every second
	for _, nameserver := range r.Nameservers() {
		wg.Add(1)
		go L(nameserver)
		// but exit early, if we have an answer
		select {
		case r := <-res:
			Info("\n KARMTAG: Resolved to: %s\n", r.Answer)
			if(sinkByIPAddress(r) || sinkByHostname(qname)) {
				Info("\n KARMTAG: %s GOES TO SINKHOLE! XXX\n", r.Answer)
				sendToSinkhole(r, qname)
			}
			return r, nil
		case <-ticker.C:
			continue
		}
	}
	// wait for all the namservers to finish
	wg.Wait()
	select {
	case r := <-res:
		// TODO: Remove the following block, it is covered in the aforementioned loop
		Info("\n Resolved to: %s", r.Answer)
		if(sinkByIPAddress(r) || sinkByHostname(qname)) {
			Info("\n KARMTAG: %s GOES TO SINKHOLE! QQQQ\n", r.Answer)
			sendToSinkhole(r, qname)
		}
		return r, nil
	default:
		return nil, ResolvError{qname, net, r.Nameservers()}
	}

}

// Namservers return the array of nameservers, with port number appended.
// '#' in the name is treated as port separator, as with dnsmasq.
func (r *Resolver) Nameservers() (ns []string) {
	for _, server := range r.config.Servers {
		if i := strings.IndexByte(server, '#'); i > 0 {
			server = server[:i] + ":" + server[i+1:]
		} else {
			server = server + ":" + r.config.Port
		}
		ns = append(ns, server)
	}
	return
}

func (r *Resolver) Timeout() time.Duration {
	return time.Duration(r.config.Timeout) * time.Second
}
