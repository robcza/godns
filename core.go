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
	cacheFile = "/tmp/cache.dat"
)

func init() {

}

func StartCoreClient(cache Cache) {
	timer = time.NewTicker(time.Minute * 30)
	for {
		updateFromCore(cache)
		<-timer.C
	}
}

func updateFromCore(cache Cache) {
	coreCache := downloadCache()
	updateCache(cache, coreCache)
}

func downloadCache() *CoreCache {
	// FIXME: fetch cache protobuf from Core
	return readFile()
}

func updateCache(cache Cache, coreCache *CoreCache) {
	// FIXME: replace content of cache, remove missing
	for _, r := range coreCache.Record {
		// FIXME : do in bigger slices not one by one
		cache.Set(*r.Key, r.Value)
	}
	logDebugMemory("After cache update")
}

func readFile() *CoreCache {
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
