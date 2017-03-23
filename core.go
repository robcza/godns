package main

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"io"

	"github.com/golang/protobuf/proto"
)

var (
	timer     *time.Ticker
	cacheFile = "/tmp/whitelist.bin"
)

func init() {

}

// StartCoreClient starts periodic download of cache files from CORE
func StartCoreClient(whitelistCache SinklistCache) {
	timer = time.NewTicker(time.Minute * 1)
	for {
		updateFromCore(whitelistCache)
		<-timer.C
	}
}

func updateFromCore(cache SinklistCache) {
	coreCache := downloadCache()
	updateCache(cache, coreCache)
}

func downloadCache() *CoreCache {
	// FIXME: fetch cache protobuf from Core
	return readFile()
}

func updateCache(cache SinklistCache, coreCache *CoreCache) {
	logDebugMemory("Before cache update")
	newCache := make(map[string]bool)
	for _, r := range coreCache.Record {
		newCache[*r.Key] = *r.Value
	}
	logDebugMemory("After cache prepared")
	cache.Replace(newCache)
	logDebugMemory("After cache update")
}

func readFile() *CoreCache {
	logDebugMemory("Before loading cache file")
	in, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		log.Fatalln("Error reading file:", err)
	}

	logDebugMemory("Before reading proto")
	cacheData := &CoreCache{}
	if err := proto.Unmarshal(in, cacheData); err != nil {
		log.Fatalln("Failed to parse cache data:", err)
	}
	logDebugMemory("After reading proto")

	return cacheData
}

func logDebugMemory(label string) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	logger.Debug(label+"- Alloc:%d, TotalA:%d, HeapA:%d, HeapSys:%d", mem.Alloc, mem.TotalAlloc, mem.HeapAlloc, mem.HeapSys)

}

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
	if err := ioutil.WriteFile(cacheFile, out, 0644); err != nil {
		log.Fatalln("Failed to write cache:", err)
	}
}
