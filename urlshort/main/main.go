package main

import (
	"flag"
	"fmt"
	"gophercises.com/urlshort"
	"log"
	"net/http"
	"os"
)

const (
	yamlPaths = `
- path: /urlshort
  url: https://github.com/gophercises/urlshort
- path: /urlshort-final
  url: https://github.com/gophercises/urlshort/tree/solution
`
	dbBucketName = "PathToUrl"
)

func main() {
	var yamlFile, dbPath string
	flag.StringVar(&yamlFile, "yaml-path", "", "Load path mappings from a yaml file")
	flag.StringVar(&dbPath, "db-name", "bolt.db", "Load mappings from a bolt database")
	flag.Parse()
	mux := defaultMux()

	// Build the MapHandler using the mux as the fallback
	pathsToUrls := map[string]string{
		"/urlshort-godoc": "https://godoc.org/github.com/gophercises/urlshort",
		"/yaml-godoc":     "https://godoc.org/gopkg.in/yaml.v2",
	}
	mapHandler := urlshort.MapHandler(pathsToUrls, mux)

	// Build the YAMLHandler using the mapHandler as the
	// fallback
	var yaml = []byte(yamlPaths)
	if content, err := os.ReadFile(yamlFile); err == nil {
		yaml = content[:]
	} else if yamlFile != "" {
		log.Printf("Unable to read file: %s.\nError: %v", yamlFile, err)
	}
	yamlHandler, err := urlshort.YAMLHandler(yaml, mapHandler)
	if err != nil {
		log.Fatal(err)
	}

	// Get some data from the db if possible
	dbHandler, err := urlshort.BoltDbHandler(dbPath, dbBucketName, yamlHandler)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Starting the server on :8080")
	log.Fatal(http.ListenAndServe(":8080", dbHandler))
}

func defaultMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", hello)
	return mux
}

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, world!")
}
