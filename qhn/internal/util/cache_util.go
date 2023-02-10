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

func createCacheFromSource(source map[string]cache.Item, cachePersistFreq time.Duration) *cache.Cache {
	var (
		c              *cache.Cache
		cachePurgeFreq = cachePersistFreq + time.Minute
	)
	if source == nil {
		c = cache.New(cache.NoExpiration, cachePurgeFreq)
	} else {
		c = cache.NewFrom(cache.NoExpiration, cachePurgeFreq, source)
	}

	go func() {
		if cachePersistFreq > 0 {
			//cache is saved to disk a minute before it is purged
			ticker := time.NewTicker(cachePersistFreq)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					time.AfterFunc(time.Minute, func() {
						ticker.Reset(cachePersistFreq)
					})
					persistCacheToDisk(c)
				}
			}
		} else {
			log.Printf("cache will not be persisted! frequency duration is not positive")
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
	if err := encoder.Encode(time.Now()); err != nil {
		log.Printf("failed to write to write last updated time to disk: %v\n", err)
	}
}

func LoadCacheFromDisk(cachePersistFreq time.Duration) (*cache.Cache, time.Time) {
	encodeFile, err := os.OpenFile(cacheFilePath, os.O_RDONLY|os.O_CREATE, 0644)
	lastCacheUpdate := time.UnixMilli(0)
	if err != nil {
		log.Printf("failed to load cache from disk: %v\n", err)
		return createCacheFromSource(nil, cachePersistFreq), lastCacheUpdate
	}

	defer encodeFile.Close()

	decoder := gob.NewDecoder(encodeFile)
	cacheItems := make(map[string]cache.Item)
	if err := decoder.Decode(&cacheItems); err != nil {
		log.Printf("failed to decode cache: %v\n", err)
	} else {
		log.Println("cache loaded from disk")
	}
	if err := decoder.Decode(&lastCacheUpdate); err != nil {
		log.Printf("failed to decode last update time: %v\n", err)
	}
	return createCacheFromSource(cacheItems, cachePersistFreq), lastCacheUpdate
}
