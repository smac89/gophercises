package urlshort

import (
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"log"
	"net/http"
	"time"
)

type pathMap struct {
	pathsToUrl []struct {
		Path string `yaml:"path"`
		Url  string `yaml:"url"`
	}
}

// MapHandler will return an http.HandlerFunc (which also
// implements http.Handler) that will attempt to map any
// paths (keys in the map) to their corresponding URL (values
// that each key in the map points to, in string format).
// If the Path is not provided in the map, then the fallback
// http.Handler will be called instead.
func MapHandler(pathsToUrls map[string]string, fallback http.Handler) http.HandlerFunc {
	if len(pathsToUrls) == 0 {
		return fallback.ServeHTTP
	}
	log.Printf("Creating mappings: %v\n", pathsToUrls)
	return func(w http.ResponseWriter, req *http.Request) {
		if url, ok := pathsToUrls[req.URL.Path]; ok {
			log.Printf("Redirecting %s to %s", req.URL.Path, url)
			http.Redirect(w, req, url, http.StatusMovedPermanently)
		} else {
			fallback.ServeHTTP(w, req)
		}
	}
}

// YAMLHandler will parse the provided YAML and then return
// an http.HandlerFunc (which also implements http.Handler)
// that will attempt to map any paths to their corresponding
// URL. If the Path is not provided in the YAML, then the
// fallback http.Handler will be called instead.
//
// YAML is expected to be in the format:
//
//   - Path: /some-Path
//     Url: https://www.some-url.com/demo
//
// The only errors that can be returned all related to having
// invalid YAML data.
//
// See MapHandler to create a similar http.HandlerFunc via
// a mapping of paths to urls.
func YAMLHandler(yml []byte, fallback http.Handler) (http.HandlerFunc, error) {
	pm := pathMap{}
	if err := yaml.Unmarshal(yml, &pm); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %v", err)
	}
	pathsToUrl := make(map[string]string)
	for _, v := range pm.pathsToUrl {
		pathsToUrl[v.Path] = v.Url
	}
	return MapHandler(pathsToUrl, fallback), nil
}

func BoltDbHandler(dbPath, dbBucket string, fallback http.Handler) (http.HandlerFunc, error) {
	db, err := bolt.Open(dbPath, 0666, &bolt.Options{ReadOnly: true, Timeout: 1 * time.Second})
	if errors.Is(err, fs.ErrNotExist) {
		log.Printf("dbfile not found, using fallback\n%v\n", err)
		return fallback.ServeHTTP, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open db file\n%v", err)
	}
	defer func(db *bolt.DB) {
		err := db.Close()
		if err != nil {
			log.Printf("failed to close the db\n%v\n", err)
		}
	}(db)
	tx, err := db.Begin(false)
	if err != nil {
		return nil, fmt.Errorf("unable to start transaction\n%v", err)
	}

	defer func(tx *bolt.Tx) {
		err := tx.Rollback()
		if err != nil {
			log.Printf("failed to rollback transaction\n%v\n", err)
		}
	}(tx)
	c := tx.Bucket([]byte(dbBucket)).Cursor()
	pathsToUrl := make(map[string]string)
	for k, v := c.First(); k != nil; k, v = c.Next() {
		pathsToUrl[string(k)] = string(v)
	}
	return MapHandler(pathsToUrl, fallback), nil
}

func (p *pathMap) UnmarshalYAML(unmarshal func(any) error) error {
	return unmarshal(&p.pathsToUrl)
}
