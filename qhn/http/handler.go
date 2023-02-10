package http

import (
	"context"
	"encoding/gob"
	"fmt"
	"github.com/patrickmn/go-cache"
	"gophercises.com/qhn/hn"
	"gophercises.com/qhn/internal/util"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	cacheKeyTopStories        = "topStories"             //A key used to represent the cached topStories
	cacheKeyTopStoriesUpdated = "_" + cacheKeyTopStories //A key used to store freshly fetched topStories
	cacheStoriesExpireFreq    = time.Minute * 30
	cachePersistFreq          = cacheStoriesExpireFreq + time.Minute //The frequency with cache items are persisted
)

type HandlerOpts struct {
	ParallelFetch uint
}

type handler struct {
	*HandlerOpts
	tpl        *template.Template
	numStories uint
	cache      *cache.Cache
	cacheInit  *sync.Once
}

func (opts *HandlerOpts) fillDefaults() *HandlerOpts {
	if opts.ParallelFetch == 0 {
		opts.ParallelFetch = parallelFetch
	}
	return opts
}

var defaultHandlerOpts = (&HandlerOpts{}).fillDefaults()

func NewHandler(numStories uint, tpl *template.Template, opts *HandlerOpts) http.Handler {
	if opts == nil {
		opts = defaultHandlerOpts
	} else {
		opts = opts.fillDefaults()
	}
	return &handler{
		tpl:         tpl,
		numStories:  numStories,
		cacheInit:   &sync.Once{},
		HandlerOpts: opts,
	}
}

func (hnd *handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	start := time.Now()

	stories := hnd.getStoriesFromCache()
	if stories == nil {
		var err error
		stories, err = getStories(hnd.numStories, hnd.ParallelFetch)

		if err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
			return
		}
		hnd.cacheTopStories(stories)
	}

	data := util.TemplateData{
		Stories: stories,
		Time:    time.Now().Sub(start),
	}
	err := hnd.tpl.Execute(w, &data)
	if err != nil {
		http.Error(w, "Failed to process the template", http.StatusInternalServerError)
		return
	}
}

func getStories(numStories, concurrency uint) ([]util.Item, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storyChannel, err := streamTopStories(ctx, concurrency)

	if err != nil {
		return nil, err
	}
	var (
		stories      []util.Item
		storiesCount uint = 0
	)

	for {
		if storiesCount >= numStories {
			cancel()
			break
		}

		if story, storyChannelOpen := <-storyChannel; storyChannelOpen {
			stories = append(stories, *story)
			storiesCount++
		} else {
			break
		}
	}
	return stories, nil
}

func streamTopStories(ctx context.Context, concurrency uint) (<-chan *util.Item, error) {
	var client hn.Client
	ids, err := client.TopItems()
	if err != nil {
		return nil, fmt.Errorf("failed to load top stories: %v", err)
	}

	fetchChannels, readChannel := fetchItems(ctx, &client, ids, concurrency)
	storyChannel := make(chan *util.Item)

	go func() {
		defer close(storyChannel)

		for {
			select {
			case countFetched, readChannelOpen := <-readChannel:
				for chIdx := uint(0); chIdx < countFetched; chIdx++ {
					select {
					case <-ctx.Done():
						readChannel <- chIdx
						return
					case rawItem := <-fetchChannels[chIdx]:
						if rawItem != nil {
							item := util.ParseHNItem(*rawItem)
							if util.IsStoryLink(item) {
								storyChannel <- &item
							}
						}
					}
				}
				if readChannelOpen {
					readChannel <- countFetched
				} else {
					return
				}
			}
		}
	}()
	return storyChannel, nil
}

func fetchItems(ctx context.Context, client *hn.Client, ids []int, chunkSize uint) ([]chan *hn.Item, chan uint) {
	fetchChannels, closeFetches := createRwChannels[*hn.Item](chunkSize)
	readChannel := make(chan uint)

	go func() {
		defer close(readChannel)
		for idChunk := range util.ChunkSlice(ids, chunkSize) {
			actualChunkSize := uint(len(idChunk))
			for idx, id := range idChunk {
				select {
				case <-ctx.Done():
					closeFetches <- util.Range{
						Start: 0,
						End:   uint(idx),
					}
					return
				default:
					go util.FetchItemChan(fetchChannels[idx], client, id)
				}
			}

			readChannel <- actualChunkSize
			lastReadCount := <-readChannel
			if lastReadCount < actualChunkSize {
				closeFetches <- util.Range{
					Start: lastReadCount,
					End:   actualChunkSize,
				}
				break
			}
		}
	}()
	return fetchChannels, readChannel
}

func createRwChannels[T any](size uint) ([]chan T, chan<- util.Range) {
	fetchChannels := createChannelSlice[T](size)
	doneChannel := make(chan util.Range)

	go func() {
		defer func() {
			for _, ch := range fetchChannels {
				close(ch)
			}
			close(doneChannel)
		}()

		unreadRange := <-doneChannel

		for unreadIndex := unreadRange.Start; unreadIndex < unreadRange.End; unreadIndex++ {
			<-fetchChannels[unreadIndex]
		}
	}()

	return fetchChannels, doneChannel
}

func createChannelSlice[T any](size uint) []chan T {
	channels := make([]chan T, size)
	for chIdx := uint(0); chIdx < size; chIdx++ {
		channels[chIdx] = make(chan T)
	}
	return channels
}

func (hnd *handler) cacheTopStories(stories []util.Item) {
	hnd.cache.Set(cacheKeyTopStories, stories, cacheStoriesExpireFreq)
}

func (hnd *handler) checkStaleCache(lastRefreshTime time.Time) {
	if time.Now().Sub(lastRefreshTime) > cacheStoriesExpireFreq {
		hnd.cache.Flush()
	}
}

func (hnd *handler) loadCache() {
	gob.RegisterName("TopStories", make([]util.Item, 1))
	c, lastUpdateTime := util.LoadCacheFromDisk(cachePersistFreq)

	c.OnEvicted(func(key string, _ any) {
		if key == cacheKeyTopStories {
			if stories, found := c.Get(cacheKeyTopStoriesUpdated); found {
				hnd.cacheTopStories(stories.([]util.Item))
				c.Delete(cacheKeyTopStoriesUpdated)
				log.Println("refreshed top stories")
			}
		}
	})

	go func() {
		if cacheStoriesExpireFreq > time.Minute {
			//A timer to fetch new items into the cache before it expires
			cacheUpdateFreq := cacheStoriesExpireFreq - time.Minute
			ticker := time.NewTicker(cacheUpdateFreq)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					time.AfterFunc(time.Minute, func() {
						//reset the timer after a minute so that we always
						//remain a minute behind the main cached items
						ticker.Reset(cacheUpdateFreq)
					})
					if _, found := c.Get(cacheKeyTopStoriesUpdated); !found {
						log.Println("pre-fetching top stories")
						if stories, err := getStories(hnd.numStories, hnd.ParallelFetch); err != nil {
							log.Printf("error pre-fetching stories: %v", err)
						} else {
							c.SetDefault(cacheKeyTopStoriesUpdated, stories)
							log.Println("pre-fetched top stories")
						}
					}
				}
			}
		}
	}()

	hnd.cache = c
	hnd.checkStaleCache(lastUpdateTime)
}

func (hnd *handler) getStoriesFromCache() []util.Item {
	if hnd.cache == nil {
		hnd.cacheInit.Do(hnd.loadCache)
	}
	c := hnd.cache
	if stories, found := c.Get(cacheKeyTopStories); found {
		return stories.([]util.Item)
	}

	if stories, found := c.Get(cacheKeyTopStoriesUpdated); found {
		hnd.cacheTopStories(stories.([]util.Item))
		return stories.([]util.Item)
	}
	return nil
}
