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
	handler := httpHn.NewHandler(numStories, tpl, &httpHn.HandlerOpts{ParallelFetch: uint(float64(numStories) * 1.25)})

	http.Handle("/", handler)
	runtime.GOMAXPROCS(2)

	// Start the server
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
