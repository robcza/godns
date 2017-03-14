package main

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

type Sinkhole struct {
	Sinkhole string `json:"sinkhole"`
}

type CoreError struct {
	When time.Time
	What string
}

func (e CoreError) Error() string {
	return fmt.Sprintf("%v: %v", e.When, e.What)
}

var (
	transportHTTP11 *http.Transport
	// transportHTTP2  http2.Transport
	transportHTTP2 *http.Transport
	client         *http.Client

	coreDisabled             uint32 = 0
	disabledSecondsTimestamp int64  = 0
)

func init() {
	if settings.LOCAL_RESOLVER {
		transportHTTP2 = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: settings.INSECURE_SKIP_VERIFY,
				NextProtos:         []string{"h2"},
				MinVersion:         tls.VersionTLS12,
				Certificates:       []tls.Certificate{credentials.clientKeyPair},
				ClientCAs:          credentials.caCertPool,
			},
		}
		client = &http.Client{
			Transport: transportHTTP2,
			Timeout:   time.Duration(settings.ORACULUM_API_TIMEOUT) * time.Millisecond,
		}
	} else {
		transportHTTP11 = &http.Transport{
			MaxIdleConnsPerHost: 20,
		}
		client = &http.Client{
			Transport: transportHTTP11,
			Timeout:   time.Duration(settings.ORACULUM_API_TIMEOUT) * time.Millisecond,
		}
	}
}

func dryAPICall(query string, clientAddress string, qname string) {
	var trimmedQname = strings.TrimSuffix(qname, ".")
	if atomic.LoadInt64(&disabledSecondsTimestamp) == 0 {
		logger.Debug("disabledSecondsTimestamp was 0, setting it to the current time")
		atomic.StoreInt64(&disabledSecondsTimestamp, int64(time.Now().Unix()))
		return
	}
	currentTime := int64(time.Now().Unix())
	lastStamp := atomic.LoadInt64(&disabledSecondsTimestamp)
	if (currentTime-lastStamp)*1000 > settings.ORACULUM_SLEEP_WHEN_DISABLED {
		logger.Debug("Doing dry API call...")
		start := time.Now()
		//Doesn't hurt IP
		_, err := doAPICall(trimmedQname, clientAddress, trimmedQname)
		elapsed := time.Since(start)
		if err != nil {
			logger.Error("Core remains DISABLED. Gonna wait. Error: %s", err)
			atomic.StoreInt64(&disabledSecondsTimestamp, int64(time.Now().Unix()))
			return
		}
		if elapsed > time.Duration(settings.ORACULUM_API_FIT_TIMEOUT)*time.Millisecond {
			logger.Error("Core remains DISABLED. Gonna wait. Elapsed time: %s, FitResponseTime: %s", elapsed, time.Duration(settings.ORACULUM_API_FIT_TIMEOUT)*time.Millisecond)
			atomic.StoreInt64(&disabledSecondsTimestamp, int64(time.Now().Unix()))
			return
		}
		logger.Error("Core is now ENABLED")
		atomic.StoreUint32(&coreDisabled, 0)
	} else {
		logger.Debug("Not enough time passed, waiting for another call. Elapsed: %s ms, Limit: %s ms", (currentTime-lastStamp)*1000, settings.ORACULUM_SLEEP_WHEN_DISABLED)
	}
	return
}

func doAPICall(query string, clientAddress string, trimmedQname string) (value bool, err error) {
	var bufferQuery bytes.Buffer
	bufferQuery.WriteString(settings.ORACULUM_URL)
	bufferQuery.WriteString("/")
	bufferQuery.WriteString(clientAddress)
	bufferQuery.WriteString("/")
	bufferQuery.WriteString(query)
	bufferQuery.WriteString("/")
	bufferQuery.WriteString(trimmedQname)
	url := bufferQuery.String()
	logger.Debug("URL:>", url)

	//var jsonStr = []byte(`{"Key":"Something Else"}`)
	//req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set(settings.ORACULUM_ACCESS_TOKEN_KEY, settings.ORACULUM_ACCESS_TOKEN_VALUE)
	req.Header.Set("Content-Type", "application/json")
	if settings.CLIENT_ID > 0 {
		req.Header.Set(settings.CLIENT_ID_HEADER, strconv.Itoa(settings.CLIENT_ID))
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Debug("There has been an error with backend.")
		return false, err
	}
	defer resp.Body.Close()

	logger.Debug("Response Status:", resp.Status)
	logger.Debug("Response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		logger.Debug("Response Body:", string(body))
		return false, CoreError{time.Now(), "Non HTTP 200."}
	}
	// i.e. "null" or possible stray byte, not a sinkhole IP
	if len(body) < 6 {
		logger.Debug("Response short.")
		return false, nil
	}

	var sinkhole Sinkhole
	//TODO Use Sinkole instead of env property
	err = json.Unmarshal(body, &sinkhole)
	if err != nil {
		logger.Debug("There has been an error with unmarshalling the response: %s", body)
		return false, err
	}
	logger.Debug("\nSINKHOLE RETURNED from Core[%s]\n", sinkhole.Sinkhole)

	return true, nil
}

func sinkitBackendCall(query string, clientAddress string, trimmedQname string, oraculumCache Cache, cacheOnly bool) bool {
	//TODO This is just a provisional check. We need to think it over...
	if len(query) > 250 || len(query) < 3 {
		logger.Warn("Query is too long or too short: %d\n", len(query))
		return false
	}
	if strings.ContainsAny(trimmedQname, " ,*") || strings.ContainsAny(query, " ,*") {
		logger.Warn("trimmedQname `%s' or query `%s' contained a space, comma or an asterisk.\n", trimmedQname, query)
		return false
	}
	if len(clientAddress) < 3 || len(clientAddress) > 41 {
		logger.Warn("Client address is too short or too long %s\n", clientAddress)
		return false
	}
	if len(trimmedQname) < 3 || len(trimmedQname) > 250 {
		logger.Warn("Query FQDN is likely invalid: %s\n", trimmedQname)
		return false
	}

	key := RequestHash(query, trimmedQname, clientAddress)

	answer, err := oraculumCache.Get(key)
	if err == nil {
		return *answer
	}

	if cacheOnly {
		return false
	}

	start := time.Now()
	goToSinkhole, err := doAPICall(query, clientAddress, trimmedQname)
	elapsed := time.Since(start)
	if err != nil {
		atomic.StoreUint32(&coreDisabled, 1)
		atomic.StoreInt64(&disabledSecondsTimestamp, int64(time.Now().Unix()))
		logger.Error("Core was DISABLED. Error: %s", err)
		return false
	}
	if elapsed > time.Duration(settings.ORACULUM_API_FIT_TIMEOUT)*time.Millisecond {
		atomic.StoreUint32(&coreDisabled, 1)
		atomic.StoreInt64(&disabledSecondsTimestamp, int64(time.Now().Unix()))
		logger.Error("Core was DISABLED. Elapsed time: %s, FitResponseTime: %s", elapsed, time.Duration(settings.ORACULUM_API_FIT_TIMEOUT)*time.Millisecond)
		return false
	}

	oraculumCache.Set(key, &goToSinkhole)

	return goToSinkhole
}

func sinkByHostname(qname string, clientAddress string, oraculumCache Cache, cacheOnly bool) bool {
	var trimmedQname = strings.TrimSuffix(qname, ".")
	// Yes, twice trimmedQname
	return sinkitBackendCall(trimmedQname, clientAddress, trimmedQname, oraculumCache, cacheOnly)
}

// We do not sinkhole here, the side effect is that CNAMEs slip through.
func sinkByIPAddress(msg *dns.Msg, clientAddress string, qname string, oraculumCache Cache, cacheOnly bool) {
	var trimmedQname = strings.TrimSuffix(qname, ".")
	for _, element := range msg.Answer {
		logger.Debug("\nKARMTAG: RR Element: %s\n", element)
		vals := strings.Split(element.String(), "	")
		// We loop through the elements, TTL, IN, Class...
		for i := range vals {
			logger.Debug("KARMTAG: value: %s\n", vals[i])
			if strings.EqualFold(vals[i], "A") || strings.EqualFold(vals[i], "CNAME") || strings.EqualFold(vals[i], "AAAA") {
				logger.Debug("KARMTAG: value matches: %s\n", vals[i])
				// Length in bytes, not runes. Shorter doesn't make sense.
				// We ditch .root-servers.net.
				if len(vals) > i+1 && len(vals[i+1]) > 3 && !strings.HasSuffix(vals[i+1], ".root-servers.net.") {
					logger.Debug("KARMTAG: to send to Core: %s\n", vals[i+1])
					go sinkitBackendCall(strings.TrimSuffix(vals[i+1], "."), clientAddress, trimmedQname, oraculumCache, cacheOnly)
				}
				break
			}
		}
	}
}

func processCoreCom(msg *dns.Msg, qname string, clientAddress string, oraculumCache Cache) {
	// Don't bother contacting Infinispan Sinkit Core
	if settings.ORACULUM_DISABLED {
		logger.Debug("SINKIT_RESOLVER_DISABLE_INFINISPAN TRUE\n")
		return
	} else {
		logger.Debug("SINKIT_RESOLVER_DISABLE_INFINISPAN FALSE or N/A\n")
	}
	logger.Debug("\n KARMTAG: Resolved to: %s\n", msg.Answer)

	if atomic.LoadUint32(&coreDisabled) == 1 {
		logger.Debug("Core is DISABLED. Gonna call dryAPICall.")
		//TODO qname or r for the dry run???
		go dryAPICall(qname, clientAddress, qname)
		if settings.ORACULUM_IP_ADDRESSES_ENABLED {
			sinkByIPAddress(msg, clientAddress, qname, oraculumCache, true)
		}
		// We do not sinkhole based on IP address.
		if sinkByHostname(qname, clientAddress, oraculumCache, true) {
			logger.Debug("\n KARMTAG: %s GOES TO SINKHOLE!\n", msg.Answer)
			sendToSinkhole(msg, qname)
		}
	} else {
		if settings.ORACULUM_IP_ADDRESSES_ENABLED {
			sinkByIPAddress(msg, clientAddress, qname, oraculumCache, false)
		}
		// We do not sinkhole based on IP address.
		if sinkByHostname(qname, clientAddress, oraculumCache, false) {
			logger.Debug("\n KARMTAG: %s GOES TO SINKHOLE!\n", msg.Answer)
			sendToSinkhole(msg, qname)
		}
	}
}

func sendToSinkhole(msg *dns.Msg, qname string) {
	var buffer bytes.Buffer
	buffer.WriteString(qname)
	buffer.WriteString("	")
	buffer.WriteString(strconv.Itoa(settings.SINKHOLE_TTL))
	buffer.WriteString("	")
	buffer.WriteString("IN	")
	buffer.WriteString("A	")
	buffer.WriteString(settings.SINKHOLE_ADDRESS)
	sinkRecord, _ := dns.NewRR(buffer.String())
	msg.Answer = []dns.RR{sinkRecord}
	return
}

// RequestHash computes hash of dns query
func RequestHash(query string, trimmedQname string, clientAddress string) string {
	keygen := md5.New()
	var buffer bytes.Buffer
	buffer.WriteString(query)
	if !settings.LOCAL_RESOLVER {
		buffer.WriteString(clientAddress)
	}
	buffer.WriteString(trimmedQname)
	keygen.Write(buffer.Bytes())
	return hex.EncodeToString(keygen.Sum(nil))
}
