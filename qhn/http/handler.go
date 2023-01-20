package http

import (
	"context"
	"fmt"
	"gophercises.com/qhn/hn"
	"gophercises.com/qhn/internal/util"
	"html/template"
	"net/http"
	"time"
)

type HandlerOpts struct {
	ParallelFetch uint
}

type handler struct {
	*HandlerOpts
	tpl        *template.Template
	numStories uint
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
		HandlerOpts: opts,
	}
}

func (hnd handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	storyChannel, err := fetchTopStories(ctx, hnd.ParallelFetch)

	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		return
	}
	var (
		stories      []util.Item
		storiesCount uint = 0
	)

	for {
		if storiesCount >= hnd.numStories {
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

	data := util.TemplateData{
		Stories: stories,
		Time:    time.Now().Sub(start),
	}
	err = hnd.tpl.Execute(w, &data)
	if err != nil {
		http.Error(w, "Failed to process the template", http.StatusInternalServerError)
		return
	}
}

func fetchTopStories(ctx context.Context, concurrency uint) (<-chan *util.Item, error) {
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
