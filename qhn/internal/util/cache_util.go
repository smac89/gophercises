package util

import (
	"encoding/gob"
	"github.com/patrickmn/go-cache"
	"log"
	"os"
	"time"
)

const (
	cacheFilePath = "qhn.gob" //The name of the cache file
)

func createCacheFromSource(source map[string]cache.Item, cacheCleanupFreq time.Duration) *cache.Cache {
	var c *cache.Cache
	if source == nil {
		c = cache.New(cache.NoExpiration, cacheCleanupFreq)
	} else {
		c = cache.NewFrom(cache.NoExpiration, cacheCleanupFreq, source)
	}

	go func() {
		if cacheCleanupFreq >= time.Minute*2 {
			//cache is saved to disk a minute before it is purged
			persistFreq := cacheCleanupFreq - time.Minute
			ticker := time.NewTicker(persistFreq)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					time.AfterFunc(time.Minute, func() {
						ticker.Reset(persistFreq)
					})
					persistCacheToDisk(c)
				}
			}
		}
	}()

	return c
}

func persistCacheToDisk(c *cache.Cache) {
	encodeFile, err := os.OpenFile(cacheFilePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("failed to open/create cache file: %v\n", err)
		return
	}

	defer encodeFile.Close()
	encoder := gob.NewEncoder(encodeFile)
	if err := encoder.Encode(c.Items()); err != nil {
		log.Printf("failed to write cache to disk: %v\n", err)
	} else {
		log.Println("cache persisted to disk")
	}
}

func LoadCacheFromDisk(cacheCleanupFreq time.Duration) *cache.Cache {
	encodeFile, err := os.OpenFile(cacheFilePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("failed to load cache from disk: %v\n", err)
		return createCacheFromSource(nil, cacheCleanupFreq)
	}

	defer encodeFile.Close()

	decoder := gob.NewDecoder(encodeFile)
	cacheItems := make(map[string]cache.Item)
	if err := decoder.Decode(&cacheItems); err != nil {
		log.Printf("failed to decode cache: %v\n", err)
	} else {
		log.Println("cache loaded from disk")
	}
	return createCacheFromSource(cacheItems, cacheCleanupFreq)
}
