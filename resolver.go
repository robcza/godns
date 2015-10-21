package main

import (
// Standard library packages
	"fmt"
	"strings"
	"sync"
	"time"
	"bytes"
//	"sort"
	"net"
	"strconv"
// Third party packages
	"github.com/miekg/dns"
	"os"
	"io/ioutil"
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type Listed struct {
	Year uint16 `json:"year"`
	Month uint8 `json:"month"`
	DayOfMonth uint8 `json:"dayOfMonth"`
	HourOfDay  uint8 `json:"hourOfDay"`
	Minute uint8 `json:"minute"`
	Second  uint8 `json:"second"`
}

type BlackListedRecord struct {
	BlackListedDomainOrIP string `json:"blackListedDomainOrIP"`
	Listed Listed `json:"listed"`
	Sources map[string]string `json:"sources"`
}

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

type CoreError struct {
	When time.Time
	What string
}

func (e CoreError) Error() string {
	return fmt.Sprintf("%v: %v", e.When, e.What)
}


func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

const (
	hardRequestTimeout = 2 * time.Second
	fitResponseTime = 10  * time.Millisecond
	sleepWhenDisabled int64 = 5 //Seconds
)

var coreApiServer = "http://"+os.Getenv("SINKIT_CORE_SERVER")+":"+os.Getenv("SINKIT_CORE_SERVER_PORT")+"/sinkit/rest/blacklist/dns"
var timeout = time.Duration(hardRequestTimeout)
var transport = http.Transport{
	Dial: dialTimeout,
}
var coreDisabled uint32 = 1
var disabledSecondsTimestamp int64 = 0

func dryAPICall(query string, clientAddress string) {
	if (atomic.LoadInt64(&disabledSecondsTimestamp) == 0) {
		logger.Debug("disabledSecondsTimestamp was 0, setting it to the current time")
		atomic.StoreInt64(&disabledSecondsTimestamp, int64(time.Now().Unix()))
		return
	}
	if (int64(time.Now().Unix()) - atomic.LoadInt64(&disabledSecondsTimestamp) > sleepWhenDisabled) {
		logger.Debug("Doing dry API call...")
		start := time.Now()
		_, err := doAPICall(query, clientAddress)
		elapsed := time.Since(start)
		if (err == nil && elapsed < fitResponseTime) {
			logger.Debug("Core is now ENABLED")
			atomic.StoreUint32(&coreDisabled, 0)
			return
		} else {
			logger.Debug("Core remains DISABLED")
		}
	} else {
		logger.Debug("Not enough time passed, waiting for another call...")
	}
	return
}

func doAPICall(query string, clientAddress string) (value bool, err error) {
	var bufferQuery bytes.Buffer
	bufferQuery.WriteString(coreApiServer)
	bufferQuery.WriteString("/")
	bufferQuery.WriteString(clientAddress)
	bufferQuery.WriteString("/")
	bufferQuery.WriteString(query)
	url := bufferQuery.String()
	logger.Debug("URL:>", url)

	//var jsonStr = []byte(`{"title":"Buy cheese and bread for breakfast."}`)
	//req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("X-sinkit-token", os.Getenv("SINKIT_ACCESS_TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Transport: &transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Debug("There has been an error with backend.")
		return false, err
	}
	defer resp.Body.Close()

	logger.Debug("response Status:", resp.Status)
	logger.Debug("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	if (resp.StatusCode != 200) {
		logger.Debug("response Body:", string(body))
		return false, CoreError{time.Now(), "Non HTTP 200."}
	}
	if (len(body) < 10) {
		return false, nil
	}

	var blacklistedRecord BlackListedRecord
	err = json.Unmarshal(body, &blacklistedRecord)
	if err != nil {
		logger.Debug("There has been an error with unmarshalling the response: %s", body)
		return false, err
	}
	fmt.Printf("\nblacklistedRecord.sources[%s]\n", blacklistedRecord.Sources)

	return true, nil
}


func sinkitBackendCall(query string, clientAddress string) (bool) {
	// Don't bother contacting Infinispan Sinkit Core
	// Move this to configuration, this is evaluating each time...
	infDisabled, err := strconv.ParseBool(os.Getenv("SINKIT_RESOLVER_DISABLE_INFINISPAN"))
	if (err == nil && infDisabled) {
		return false
	}

	//TODO This is just a provisional check. We need to think it over...
	if (len(query) > 250) {
		fmt.Printf("Query is too long: %d\n", len(query))
		return false
	}

	start := time.Now()
	goToSinkhole, err := doAPICall(query, clientAddress)
	elapsed := time.Since(start)
	if (err != nil || elapsed > fitResponseTime) {
		atomic.StoreUint32(&coreDisabled, 1)
		return false
	}

	return goToSinkhole
}

// Dummy playground
func sinkByHostname(qname string, clientAddress string) (bool) {
	return sinkitBackendCall(strings.TrimSuffix(qname, "."), clientAddress)
}

// Dummy playground
func sinkByIPAddress(msg *dns.Msg, clientAddress string) (bool) {
	/*if !aRecord.Equal(ip) {
			t.Fatalf("IP %q does not match registered IP %q", aRecord, ip)
		}*/
	//dummyTestIPAddresses := []string{"81.19.0.120"}
	//sort.Strings(dummyTestIPAddresses)
	for _, element := range msg.Answer {
		logger.Debug("\nKARMTAG: RR Element: %s\n", element)
		//if (sort.SearchStrings(dummyTestIPAddresses, element.String()) !=  len(dummyTestIPAddresses)) {
		//		return true
		//	}
		var respElemSegments = strings.Split(element.String(), "	")
		if (sinkitBackendCall((respElemSegments[len(respElemSegments)-1:])[0], clientAddress)) {
			return true
		}
	}
	return false
}

// Dummy playground
func sendToSinkhole(msg *dns.Msg, qname string) {
	var buffer bytes.Buffer
	buffer.WriteString(qname)
	buffer.WriteString("	")
	buffer.WriteString("10	")
	buffer.WriteString("IN	")
	buffer.WriteString("A	")
	buffer.WriteString(os.Getenv("SINKIT_SINKHOLE_IP"))
	//Sink only the first record
	//msg.Answer[0], _ = dns.NewRR(buffer.String())
	//Sink all records:
	sinkRecord, _ := dns.NewRR(buffer.String())
	msg.Answer = []dns.RR{sinkRecord}
	//Debug("\n KARMTAG: A record: %s", msg.Answer[0].(*dns.A).String())
	//Debug("\n KARMTAG: CNAME record: %s", msg.Answer[1].(*dns.CNAME).String())
	return
}

// Lookup will ask each nameserver in top-to-bottom fashion, starting a new request
// in every second, and return as early as possible (have an answer).
// It returns an error if no request has succeeded.
func (r *Resolver) Lookup(net string, req *dns.Msg, remoteAddress net.Addr) (message *dns.Msg, err error) {
	c := &dns.Client{
		Net:          net,
		ReadTimeout:  r.Timeout(),
		WriteTimeout: r.Timeout(),
	}

	qname := req.Question[0].Name
	clientAddress := strings.Split(remoteAddress.String(), ":")[0]

	res := make(chan *dns.Msg, 1)
	var wg sync.WaitGroup
	L := func(nameserver string) {
		defer wg.Done()
		r, rtt, err := c.Exchange(req, nameserver)
		if err != nil {
			logger.Warn("%s socket error on %s", qname, nameserver)
			logger.Warn("error:%s", err.Error())
			return
		}
		// If SERVFAIL happen, should return immediately and try another upstream resolver.
		// However, other Error code like NXDOMAIN is an clear response stating
		// that it has been verified no such domain existas and ask other resolvers
		// would make no sense. See more about #20
		if r != nil && r.Rcode != dns.RcodeSuccess {
			logger.Warn("%s failed to get an valid answer on %s", qname, nameserver)
			if r.Rcode == dns.RcodeServerFailure {
				return
			}
		} else {
			logger.Debug("%s resolv on %s (%s) ttl: %d", UnFqdn(qname), nameserver, net, rtt)
		}
		logger.Debug("\n KARMTAG: %s resolv on %s (%s) ttl: %d\n", UnFqdn(qname), nameserver, net, rtt)
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
			logger.Debug("\n KARMTAG: Resolved to: %s\n", r.Answer)
			if (atomic.LoadUint32(&coreDisabled) == 1) {
				logger.Debug("Core is DISABLED. Gonna call dryAPICall")
				//TODO WARNING qname or r ???
				go dryAPICall(qname, clientAddress)
				logger.Debug("...returning.")
			} else {
				if (sinkByHostname(qname, clientAddress) || sinkByIPAddress(r, clientAddress)) {
					logger.Debug("\n KARMTAG: %s GOES TO SINKHOLE! XXX\n", r.Answer)
					sendToSinkhole(r, qname)
				}
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
		logger.Debug("\n Resolved to: %s", r.Answer)
		if (atomic.LoadUint32(&coreDisabled) == 1) {
			logger.Debug("Core is DISABLED. Gonna call dryAPICall")
			//TODO WARNING qname or r ???
			go dryAPICall(qname, clientAddress)
			logger.Debug("...returning.")
		} else {
			if (sinkByHostname(qname, clientAddress) || sinkByIPAddress(r, clientAddress)) {
				logger.Debug("\n KARMTAG: %s GOES TO SINKHOLE! QQQQ\n", r.Answer)
				sendToSinkhole(r, qname)
			}
		}
		return r, nil
	default:
		return nil, ResolvError{qname, net, r.Nameservers()}
	}

}

// Namservers return the array of nameservers, with port number appended.
// '#' in the name is treated as port separator, as with dnsmasq.
func (r *Resolver) Nameservers() (ns []string) {
	if (settings.Backend.UseExclusively) {
		logger.Debug("Using exclusively these backend servers:\n")
		for _, server := range settings.Backend.BackendResolvers {
			logger.Debug(" Appending backend server: %s \n", server)
			ns = append(ns, server)
		}
	} else {
		for _, server := range r.config.Servers {
			if i := strings.IndexByte(server, '#'); i > 0 {
				server = server[:i] + ":" + server[i+1:]
			} else {
				server = server + ":" + r.config.Port
			}
			ns = append(ns, server)
		}
	}
	return
}

func (r *Resolver) Timeout() time.Duration {
	return time.Duration(r.config.Timeout) * time.Second
}
