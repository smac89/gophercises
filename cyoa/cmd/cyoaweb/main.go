package main

import (
	"context"
	"flag"
	"fmt"
	"gophercises.com/cyoa"
	cyoahttp "gophercises.com/cyoa/http"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	srv := startStoryServer(story, port, nil)
	exit := make(chan os.Signal, 1)

	signal.Notify(exit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-exit

	log.Println("Received shutdown signal")
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func startStoryServer(story cyoa.Story, port int, handlerOpts *cyoahttp.HandlerOption) *http.Server {
	h := cyoahttp.NewStoryHandler(story, handlerOpts)
	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: h}
	go func() {
		log.Printf("Starting the server on port %d\n", port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
		log.Println("Shutting down server")
	}()
	return srv
}
