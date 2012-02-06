package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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
