package main

import (
	"flag"
	"fmt"
	"gophercises.com/cyoa"
	cyoahttp "gophercises.com/cyoa/http"
	"log"
	"net/http"
	"os"
)

func main() {
	var (
		filename string
		port     int
	)
	flag.StringVar(&filename, "file", "gopher.json", "the JSON file with the CYOA story")
	flag.IntVar(&port, "port", 3000, "the port to start the CYOA web application on")
	flag.Parse()

	log.Printf("Using the story in %s\n", filename)

	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	var story cyoa.Story
	if err := story.DecodeJson(f); err != nil {
		panic(err)
	}

	h := cyoahttp.NewStoryHandler(story, nil)
	log.Printf("Starting the server on port %d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), h))
}
