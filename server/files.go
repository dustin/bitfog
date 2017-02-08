package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
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
	log.Printf("Writing %v", abs)
	ctype := req.Header.Get("Content-Type")
	switch ctype {
	default:
		http.Error(w, "invalid content type: "+ctype, 400)
		return
	case "application/octet-stream":
		os.RemoveAll(abs)
		f, err := os.Create(abs)
		if err != nil {
			os.MkdirAll(filepath.Dir(abs), 0777)
			f, err = os.Create(abs)
			if err != nil {
				log.Printf("Problem opening %s: %v", abs, err)
				http.Error(w, "error deleting file: "+err.Error(), 500)
				return
			}
		}
		defer log.Printf("Created file %s", abs)
		if _, err := io.Copy(f, req.Body); err != nil {
			f.Close()
			http.Error(w, "error writing data: "+err.Error(), 500)
			return
		}
		if err := f.Close(); err != nil {
			http.Error(w, "error closing: "+err.Error(), 500)
			return
		}
	case "application/symlink":
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("Error reading symlink body.")
			http.Error(w, "Error reading symlink body: "+err.Error(), 400)
			return
		}
		dest := string(body)
		err = os.Symlink(dest, abs)
		if err != nil {
			os.MkdirAll(filepath.Dir(abs), 0777)
			os.RemoveAll(abs)
			err = os.Symlink(dest, abs)
			if err != nil {
				log.Printf("Problem symlinking %s: %v", abs, err)
				http.Error(w, "Error creating symlink: "+err.Error(), 500)
				return
			}
		}
		log.Printf("Created symlink: %v -> %v", abs, dest)
	}
	w.WriteHeader(204)
}

func doDelete(abs string, w http.ResponseWriter, req *http.Request) {
	err := os.Remove(abs)
	if err != nil {
		log.Printf("Error deleting:  %v", err)
		http.Error(w, "Error deleting file: "+err.Error(), 500)
		return
	}
	log.Printf("Deleted %s", abs)
	w.WriteHeader(204)
}

func handlePatch(conf itemConf, abs string, w http.ResponseWriter, req *http.Request) {
	mode := req.FormValue("rdiff")
	switch mode {
	default:
		log.Printf("unsupported mode: %v", mode)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid mode: %v", mode)
	case "delta":
		// Compute a delta.
		log.Printf("Computing a patch of %s", abs)
		f, err := ioutil.TempFile(os.TempDir(), "bitfog-"+mode+".")
		if err != nil {
			log.Printf("Error creating tmp file %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error creating tmp file")
			return
		}
		defer f.Close()
		defer os.Remove(f.Name())
		_, err = io.Copy(f, req.Body)
		if err != nil {
			log.Printf("Error writing to tmp file %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error writing to tmp file")
			return
		}
		cmd := exec.Command("rdiff", mode, f.Name(), abs)
		cmd.Stdout = w
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			log.Printf("Error running rdiff on %s: %v", abs, err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error creating result")
			return
		}
		w.WriteHeader(200)
		err = cmd.Wait()
		if err != nil {
			log.Printf("Error completing rdiff: %v", err)
		}
	case "patch":
		if !conf.Writable {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Can't %s here.\n", req.Method)
			return
		}
		// Apply a patch
		log.Printf("Patching %s", abs)
		f, err := ioutil.TempFile(os.TempDir(), "bitfog-diff.")
		if err != nil {
			log.Printf("Error creating tmp file %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error creating tmp file")
			return
		}
		defer f.Close()
		defer os.Remove(f.Name())

		_, err = io.Copy(f, req.Body)
		if err != nil {
			log.Printf("Error writing to tmp file %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error writing to tmp file")
			return
		}

		fout, err := ioutil.TempFile(os.TempDir(), "bitfog-result.")
		if err != nil {
			log.Printf("Error creating tmp file %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error creating tmp file")
			return
		}
		defer fout.Close()
		defer os.Remove(fout.Name())

		cmd := exec.Command("rdiff", mode, abs, f.Name())
		cmd.Stdout = fout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			log.Printf("Error running rdiff on %s: %v", abs, err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error creating result")
			return
		}
		err = cmd.Wait()
		if err != nil {
			log.Printf("Error completing rdiff: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error apply patch")
			return
		}

		err = os.Rename(fout.Name(), abs)
		if err != nil {
			log.Printf("Error completing rdiff: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error moving file into place")
		}

		w.WriteHeader(204)
	}
}

func handleGet(conf itemConf, abs string, w http.ResponseWriter, req *http.Request) {
	fi, err := os.Lstat(abs)
	if err != nil {
		log.Printf("Error getting file info file: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error getting file info.\n")
		return
	}

	if isa(fi.Mode(), os.ModeSymlink) {
		log.Printf("Trying to read a symlink at: %v", abs)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error getting file info.\n")
		return
	}

	switch req.FormValue("rdiff") {
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid rdiff param: %v", req.FormValue("rdiff"))
	case "":
		log.Printf("Getting %s", abs)

		f, err := os.Open(abs)
		if err != nil {
			log.Printf("Error opening file: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Error fetching file.\n")
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		if err != nil {
			log.Printf("Error streaming file: %v", err)
		}
	case "sig":
		// Generating an rdiff signature
		cmd := exec.Command("rdiff", "signature", abs)
		cmd.Stdout = w
		err := cmd.Start()
		if err != nil {
			log.Printf("Error running rdiff on %s: %v", abs, err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error creating result")
			return
		}
		w.WriteHeader(200)
		err = cmd.Wait()
		if err != nil {
			log.Printf("Error completing rdiff: %v", err)
		}
	}
}

func handlePath(conf itemConf, subpath string, w http.ResponseWriter, req *http.Request) {
	if subpath == "" {
		log.Printf("Listing %s", conf.Path)
		w.Header().Set("Content-Type", "application/json")
		listPath(conf, w, req)
	} else {
		abs, err := absolutize(conf.Path, subpath)
		if err != nil {
			w.WriteHeader(err.status)
			fmt.Fprintf(w, "%s\n", err.msg)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		switch req.Method {
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Can't %s here.\n", req.Method)
		case "GET":
			handleGet(conf, abs, w, req)
		case "PATCH":
			handlePatch(conf, abs, w, req)
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
