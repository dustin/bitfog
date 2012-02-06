package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type itemConf struct {
	Path     string
	Writable bool
}

var paths = map[string]itemConf{"tmp": {"/tmp/", false}}

func doIndex(w http.ResponseWriter, req *http.Request) {
	log.Printf("Listing areas.")
	keys := make([]string, len(paths)-1)
	for k, _ := range paths {
		keys = append(keys, k)
	}
	log.Printf("Stuff:  %#v, %#v", keys, paths)
	json.NewEncoder(w).Encode(keys)
}

func notFoundPath(p string, w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(404)
	fmt.Fprintf(w, "Path not found: %s\n", p)
}

func handler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	parts := strings.SplitN(req.URL.Path[1:], "/", 2)
	subpath := ""
	if len(parts) > 1 {
		subpath = parts[1]
	}
	path, foundPath := paths[parts[0]]

	switch {
	default:
		notFoundPath(parts[0], w, req)
	case parts[0] == "":
		doIndex(w, req)
	case foundPath:
		handlePath(path, subpath, w, req)
	}
}

func main() {
	addr := ":8675"
	s := &http.Server{
		Addr:         addr,
		Handler:      http.HandlerFunc(handler),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Printf("Listening to web requests on %s", addr)
	log.Fatal(s.ListenAndServe())
}
