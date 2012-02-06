package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type itemConf struct {
	Path     string `json:"path"`
	Writable bool   `json:"writable"`
	Checksum bool   `json:"checksum"`
}

var paths = make(map[string]itemConf)

func doIndex(w http.ResponseWriter, req *http.Request) {
	log.Printf("Listing areas.")
	keys := []string{}
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

func loadConf(c string) {
	f, err := os.Open(c)
	if err != nil {
		log.Fatalf("Error opening conf file: %v", err)
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&paths)
	if err != nil {
		log.Fatalf("Error reading conf file:  %v", err)
	}
}

func main() {
	addr := flag.String("addr", ":8675", "Address to bind to")
	confFile := flag.String("conf", "bitfog.json", "Configuration file")
	flag.Parse()

	loadConf(*confFile)

	s := &http.Server{
		Addr:    *addr,
		Handler: http.HandlerFunc(handler),
	}
	log.Printf("Listening to web requests on %s", *addr)
	log.Fatal(s.ListenAndServe())
}
