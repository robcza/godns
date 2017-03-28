package main

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"io"

	"crypto/md5"

	"fmt"

	"github.com/golang/protobuf/proto"
)

const (
	whitelistCacheFile  = "/tmp/whitelist.bin"
	iocCacheFile        = "/tmp/ioc.bin"
	customListCacheFile = "/tmp/custlist.bin"

	whitelistURI  = "/whitelist"
	iocURI        = "/ioc"
	customListURI = "/customlist"

	md5HeaderKey = "X-file-md5"
)

// StartCoreClient starts periodic download of cache files from CORE
func StartCoreClient(whitelist SinklistCache, ioc SinklistCache, customlist SinklistCache) {
	whitelistReq := prepareRequest(whitelistURI)
	iocReq := prepareRequest(iocURI)
	customListReq := prepareRequest(customListURI)

	whitelistTimer := time.NewTicker(time.Minute * time.Duration(settings.CACHE_REFRESH_WHITELIST))
	defer whitelistTimer.Stop()
	iocTimer := time.NewTicker(time.Minute * time.Duration(settings.CACHE_REFRESH_IOC))
	defer iocTimer.Stop()
	customlistTimer := time.NewTicker(time.Minute * time.Duration(settings.CACHE_REFRESH_CUSTOMLIST))
	defer customlistTimer.Stop()

	for {
		select {
		case <-whitelistTimer.C:
			updateCoreCache(whitelist, whitelistReq, whitelistCacheFile)
		case <-iocTimer.C:
			updateCoreCache(ioc, iocReq, iocCacheFile)
		case <-customlistTimer.C:
			updateCoreCache(customlist, customListReq, customListCacheFile)
		}
	}
}

func updateCoreCache(cache SinklistCache, req *http.Request, cacheFile string) {
	coreCache, err := downloadCache(req, cacheFile)
	if err != nil {
		logger.Error("Error updating core cache:", err)
	}
	updateCache(cache, coreCache)
}

func downloadCache(req *http.Request, cacheFile string) (*CoreCache, error) {
	var data []byte
	var err error
	err = retry(settings.CACHE_RETRY_COUNT, time.Duration(settings.CACHE_RETRY_INTERVAL)*time.Second, func() (err error) {
		resp, err := CoreClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if resp.Header.Get(md5HeaderKey) != fmt.Sprintf("%x", md5.Sum(data)) {
			logger.Error("Downloaded file md5 header: "+resp.Header.Get(md5HeaderKey)+", calculated md5: %x", md5.Sum(data))
			return err
		}
		return nil
	})

	if err != nil {
		logger.Error("Failed to download valid cache data:", err)
		return nil, err
	}

	cacheData := &CoreCache{}
	if err = proto.Unmarshal(data, cacheData); err != nil {
		logger.Error("Failed to parse cache data:", err)
		return nil, err
	}

	if err = ioutil.WriteFile(cacheFile, data, 0644); err != nil {
		logger.Error("Error writing cache file: ", err)
	}

	return cacheData, nil
}

func updateCache(cache SinklistCache, coreCache *CoreCache) {
	// logDebugMemory("Before cache update")
	newCache := make(map[string]bool)
	for _, r := range coreCache.Record {
		newCache[r.GetKey()] = r.GetValue()
	}
	// logDebugMemory("After cache prepared")
	cache.Replace(newCache)
	// logDebugMemory("After cache update")
}

func readCacheFile(file string) (*CoreCache, error) {
	// logDebugMemory("Before loading cache file")
	in, err := ioutil.ReadFile(file)
	if err != nil {
		logger.Error("Error reading file:", err)
		return nil, err
	}

	// logDebugMemory("Before reading proto")
	cacheData := &CoreCache{}
	if err := proto.Unmarshal(in, cacheData); err != nil {
		logger.Error("Failed to parse cache data:", err)
		return nil, err
	}
	// logDebugMemory("After reading proto")

	return cacheData, nil
}

func logDebugMemory(label string) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	logger.Debug(label+"- Alloc:%d, TotalA:%d, HeapA:%d, HeapSys:%d", mem.Alloc, mem.TotalAlloc, mem.HeapAlloc, mem.HeapSys)
}

// FillTestData prepares test whitelist cache from hosts.csv file
func FillTestData() {
	f, err := os.Open("/hosts.csv")
	if err != nil {
		log.Fatalln("Error opening file:", err)
	}
	defer f.Close()

	csvr := csv.NewReader(f)
	cacheData := &CoreCache{}

	for {
		row, err := csvr.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalln("Unexpected csv error:", err)
		}

		result := true
		hash := RequestHash(strings.TrimSuffix(row[0], "."), strings.TrimSuffix(row[0], "."), "")
		pair := &Pair{
			Key:   &hash,
			Value: &result,
		}

		cacheData.Record = append(cacheData.Record, pair)
	}
	log.Println("Writing buffer")
	out, err := proto.Marshal(cacheData)
	if err != nil {
		log.Fatalln("Failed to encode cache:", err)
	}
	if err := ioutil.WriteFile(whitelistCacheFile, out, 0644); err != nil {
		log.Fatalln("Failed to write cache:", err)
	}
}

func prepareRequest(uri string) *http.Request {
	req, _ := http.NewRequest("GET", settings.CACHE_URL+uri, nil)
	req.Header.Set(settings.ORACULUM_ACCESS_TOKEN_KEY, settings.ORACULUM_ACCESS_TOKEN_VALUE)
	req.Header.Set("Content-Type", "application/json")
	if settings.CLIENT_ID > 0 {
		req.Header.Set(settings.CLIENT_ID_HEADER, strconv.Itoa(settings.CLIENT_ID))
	}
	return req
}

func retry(attempts int, sleep time.Duration, callback func() error) (err error) {
	for i := 0; ; i++ {
		err = callback()
		if err == nil {
			return
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)

		logger.Debug("retrying after error:", err)
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
