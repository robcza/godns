package main

import (
	"io/ioutil"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"crypto/md5"

	"fmt"

	"github.com/golang/protobuf/proto"
)

type CacheFileNotFound struct {
	file string
}

func (e CacheFileNotFound) Error() string {
	return e.file + " " + "not found"
}

const (
	whitelistCacheFile  = "/data/whitelist.bin"
	iocCacheFile        = "/data/ioc.bin"
	customListCacheFile = "/data/custlist.bin"

	whitelistURI  = "/whitelist"
	iocURI        = "/ioclist"
	customListURI = "/customlist"

	md5HeaderKey = "X-file-md5"
)

// StartCoreClient prepares caches and schedules periodic updates
func StartCoreClient(listCache *ListCache) {
	if settings.LOCAL_RESOLVER {
		ensureCachePrepared(listCache.Customlist, prepareRequest(customListURI), customListCacheFile)
		ensureCachePrepared(listCache.Ioclist, prepareRequest(iocURI), iocCacheFile)
	} else {
		tryLoadCacheFile(listCache.Whitelist, whitelistCacheFile)
	}

	// separate goroutine to keep caches updated
	go waitUpdateCaches(listCache)
}

func waitUpdateCaches(listCache *ListCache) {
	var (
		iocReq        *http.Request
		customListReq *http.Request
		whitelistReq  *http.Request
	)

	if settings.LOCAL_RESOLVER {
		iocReq = prepareRequest(iocURI)
		customListReq = prepareRequest(customListURI)
	} else {
		whitelistReq = prepareRequest(whitelistURI)
	}

	whitelistTimer := time.NewTicker(time.Minute * time.Duration(settings.CACHE_REFRESH_WHITELIST))
	defer whitelistTimer.Stop()
	iocTimer := time.NewTicker(time.Minute * time.Duration(settings.CACHE_REFRESH_IOC))
	defer iocTimer.Stop()
	customlistTimer := time.NewTicker(time.Minute * time.Duration(settings.CACHE_REFRESH_CUSTOMLIST))
	defer customlistTimer.Stop()

	for {
		select {
		case <-whitelistTimer.C:
			if !settings.LOCAL_RESOLVER {
				updateCoreCache(listCache.Whitelist, whitelistReq, whitelistCacheFile)
			}
		case <-iocTimer.C:
			if settings.LOCAL_RESOLVER {
				updateCoreCache(listCache.Ioclist, iocReq, iocCacheFile)
			}
		case <-customlistTimer.C:
			if settings.LOCAL_RESOLVER {
				updateCoreCache(listCache.Customlist, customListReq, customListCacheFile)
			}
		}
	}
}

func ensureCachePrepared(cache SinklistCache, req *http.Request, cacheFile string) {
	logger.Debug("Checking file " + cacheFile)
	if tryLoadCacheFile(cache, cacheFile) == nil {
		logger.Info("Cache loaded from file " + cacheFile)
		return
	}
	for updateCoreCache(cache, req, cacheFile) != nil {
		logger.Error("Could not download cache " + cacheFile + ", retrying")
		time.Sleep(time.Second)
	}
	logger.Info("Cache " + cacheFile + " downloaded and parsed")
}

func updateCoreCache(cache SinklistCache, req *http.Request, cacheFile string) error {
	coreCache, err := downloadCache(req, cacheFile)
	if err != nil {
		return err
	}
	updateCache(cache, coreCache)
	return nil
}

func downloadCache(req *http.Request, cacheFile string) (*CoreCache, error) {
	var data []byte
	var err error
	err = retry(settings.CACHE_RETRY_COUNT, time.Duration(settings.CACHE_RETRY_INTERVAL)*time.Second, func() (err error) {
		logger.Debug("Fetching " + cacheFile + " from core")
		resp, err := CoreCacheClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("Core status code %d", resp.StatusCode)
		}
		data, err = ioutil.ReadAll(resp.Body)
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
	newCache := make(map[string]tAction)
	for _, r := range coreCache.Record {
		newCache[r.GetKey()] = tAction(r.GetValue())
	}
	// logDebugMemory("After cache prepared")
	cache.Replace(newCache)
	// logDebugMemory("After cache update")
}

func tryLoadCacheFile(cache SinklistCache, file string) error {
	cacheData, err := readCacheFile(file)
	if err != nil {
		switch err.(type) {
		case CacheFileNotFound:
			logger.Info("Cache file " + file + " not found")
		default:
			logger.Warn("Encountered error processing file "+file+" :", err)
		}
		return err
	}

	updateCache(cache, cacheData)
	return nil
}

func readCacheFile(file string) (*CoreCache, error) {
	// logDebugMemory("Before loading cache file")
	in, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, CacheFileNotFound{file: file}
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

func prepareRequest(uri string) *http.Request {
	req, _ := http.NewRequest("GET", settings.CACHE_URL+uri, nil)
	req.Header.Set(settings.ORACULUM_ACCESS_TOKEN_KEY, settings.ORACULUM_ACCESS_TOKEN_VALUE)
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
