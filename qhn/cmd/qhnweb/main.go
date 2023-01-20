package main

import (
	"flag"
	"fmt"
	httpHn "gophercises.com/qhn/http"
	"html/template"
	"log"
	"net/http"
	"runtime"
)

const (
	defaultPort       = 3000
	defaultNumStories = 30
)

func main() {
	// parse flags
	var (
		port       int
		numStories uint
	)
	flag.IntVar(&port, "port", defaultPort, "the port to start the web server on")
	flag.UintVar(&numStories, "num_stories", defaultNumStories, "the number of top stories to display")
	flag.Parse()

	tpl := template.Must(template.ParseFiles("web/templates/index.gohtml"))
	handler := httpHn.NewHandler(numStories, tpl, nil)

	http.Handle("/", handler)
	runtime.GOMAXPROCS(2)

	// Start the server
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

//func handler(numStories int, tpl *template.Template) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		readChannel := make(chan int)
//		ctx, cancel := context.WithCancel(context.Background())
//		defer cancel()
//
//		start := time.Now()
//		var client hn.Client
//		ids, err := client.TopItems()
//
//		if err != nil {
//			http.Error(w, "Failed to load top stories", http.StatusInternalServerError)
//			return
//		}
//		var stories []item
//		itemChans := make([]chan *hn.Item, parallelFetch)
//		for chIdx := 0; chIdx < parallelFetch; chIdx++ {
//			itemChans[chIdx] = make(chan *hn.Item)
//		}
//
//		go func() {
//			countFetched := 0
//			defer func() {
//				for _, ch := range itemChans {
//					close(ch)
//				}
//			}()
//			for _, id := range ids {
//				select {
//				case <-ctx.Done():
//					return
//				default:
//					go fetchItemChan(itemChans[countFetched], &client, id)
//					countFetched++
//					if countFetched >= len(itemChans) {
//						readChannel <- countFetched
//						countFetched = <-readChannel
//					}
//				}
//			}
//		}()
//
//		go func() {
//			for {
//				select {
//				case <-ctx.Done():
//					return
//				case countFetched := <-readChannel:
//					for chIdx := 0; chIdx < countFetched; chIdx++ {
//						rawItem := <-itemChans[chIdx]
//						if rawItem != nil {
//							item := parseHNItem(*rawItem)
//							if isStoryLink(item) {
//								stories = append(stories, item)
//							}
//						}
//						if len(stories) >= numStories {
//							for chIdx++; chIdx < countFetched; chIdx++ {
//								//read the rest of the fetched items to prevent panic when channel closed
//								<-itemChans[chIdx]
//							}
//							cancel()
//						}
//					}
//					readChannel <- 0
//				}
//			}
//		}()
//
//		<-ctx.Done()
//
//		data := templateData{
//			Stories: stories[:numStories],
//			Time:    time.Now().Sub(start),
//		}
//		err = tpl.Execute(w, data)
//		if err != nil {
//			http.Error(w, "Failed to process the template", http.StatusInternalServerError)
//			return
//		}
//	}
//}
