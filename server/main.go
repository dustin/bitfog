package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var paths = map[string]string{"tmp": "/tmp/"}

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

func showPath(path, subpath string, w http.ResponseWriter, req *http.Request) {
	if subpath == "" {
		log.Printf("Listing %s", path)
		listPath(path, w, req)
	} else {
		abs, err := filepath.Abs(filepath.Join(path, filepath.Clean(subpath)))
		if err != nil {
			log.Printf("Error canonicalizing path:  %v", err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Something went wrong.  I think it was you.\n")
			return
		}
		log.Printf("Showing %s under %s:  %s", subpath, path, abs)
		if !strings.HasPrefix(abs, path) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "No\n")
			return
		}

		fi, err := os.Stat(abs)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Error retrieving file.\n")
			return
		}

		if fi.IsDir() {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "That's not a file.\n")
			return
		}

		http.ServeFile(w, req, filepath.Join(path, subpath))
	}
}

func handler(w http.ResponseWriter, req *http.Request) {
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
		showPath(path, subpath, w, req)
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
