package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type fileError struct {
	status int
	msg    string
}

func (fe *fileError) Error() string {
	return fe.msg
}

func absolutize(path, subpath string) (string, *fileError) {
	abs, err := filepath.Abs(filepath.Join(path, filepath.Clean(subpath)))
	if err != nil {
		log.Printf("Error canonicalizing path:  %v", err)
		return "", &fileError{http.StatusBadRequest,
			"Something went wrong, I think it was you"}
	}
	log.Printf("Showing %s under %s:  %s", subpath, path, abs)
	if !strings.HasPrefix(abs, path) {
		return "", &fileError{http.StatusBadRequest, "No"}
	}

	fi, err := os.Stat(abs)
	if err != nil {
		return "", &fileError{http.StatusBadRequest, "Error retrieving file."}
	}

	if fi.IsDir() {
		return "", &fileError{http.StatusBadRequest, "That's not a file."}
	}
	return abs, nil
}

func showPath(path, subpath string, w http.ResponseWriter, req *http.Request) {
	if subpath == "" {
		log.Printf("Listing %s", path)
		listPath(path, w, req)
	} else {
		abs, err := absolutize(path, subpath)
		if err != nil {
			w.WriteHeader(err.status)
			fmt.Fprintf(w, "%s\n", err.msg)
			return
		}
		http.ServeFile(w, req, abs)
	}
}
