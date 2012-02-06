package main

import (
	"fmt"
	"io"
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
	if !strings.HasPrefix(abs, path) {
		return "", &fileError{http.StatusBadRequest, "No"}
	}

	fi, err := os.Stat(abs)
	if err == nil && fi.IsDir() {
		return "", &fileError{http.StatusBadRequest, "That's not a file."}
	}
	return abs, nil
}

func doPut(abs string, w http.ResponseWriter, req *http.Request) {
	f, err := os.Create(abs)
	if err != nil {
		os.MkdirAll(filepath.Dir(abs), 0777)
		f, err = os.Create(abs)
		if err != nil {
			log.Printf("Problem opening %s: %v", abs, err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Error deleting file.\n")
		}
	}
	defer f.Close()
	defer func() {
		log.Printf("Created %s", abs)
	}()
	io.Copy(f, req.Body)
}

func doDelete(abs string, w http.ResponseWriter, req *http.Request) {
	err := os.Remove(abs)
	if err != nil {
		log.Printf("Error deleting:  %v", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error deleting file.\n")
	}
	log.Printf("Deleted %s", abs)

}

func handlePath(conf itemConf, subpath string, w http.ResponseWriter, req *http.Request) {
	if subpath == "" {
		log.Printf("Listing %s", conf.Path)
		listPath(conf.Path, w, req)
	} else {
		abs, err := absolutize(conf.Path, subpath)
		if err != nil {
			w.WriteHeader(err.status)
			fmt.Fprintf(w, "%s\n", err.msg)
			return
		}
		switch req.Method {
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Can't %s here.\n", req.Method)
		case "GET":
			http.ServeFile(w, req, abs)
		case "PUT":
			if conf.Writable {
				doPut(abs, w, req)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprintf(w, "Can't %s here.\n", req.Method)
			}
		case "DELETE":
			if conf.Writable {
				doDelete(abs, w, req)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprintf(w, "Can't %s here.\n", req.Method)
			}
		}
	}
}
