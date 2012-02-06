package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

type FileData struct {
	Size  int64  `json:"size"`
	Mode  int32  `json:"mode"`
	Mtime int64  `json:"mtime"`
	Hash  uint64 `json:"hash,omitempty"`
	Dest  string `json:"linkdest,omitempty"`
}

func decodeURL(u string) (map[string]FileData, error) {
	rv := map[string]FileData{}

	resp, err := http.Get(u)
	if err != nil {
		return rv, err
	}
	if resp.StatusCode != 200 {
		return rv, errors.New(fmt.Sprintf("Error httping: %v", resp.Status))
	}
	defer resp.Body.Close()
	r := bufio.NewReader(resp.Body)

	d := json.NewDecoder(r)

	done := false
	for !done {
		type fdata struct {
			Name  string `json:"name"`
			Size  int64  `json:"size"`
			Mode  int32  `json:"mode"`
			Mtime int64  `json:"mtime"`
			Hash  uint64 `json:"hash,omitempty"`
			Dest  string `json:"linkdest,omitempty"`
		}
		fd := fdata{}
		err = d.Decode(&fd)
		switch err {
		default:
			return rv, errors.New(fmt.Sprintf("Error decoding: %v", err))
		case nil:
			rv[fd.Name] = FileData{
				Size:  fd.Size,
				Mode:  fd.Mode,
				Mtime: fd.Mtime,
				Hash:  fd.Hash,
				Dest:  fd.Dest,
			}
		case io.EOF:
			done = true
		}
	}
	return rv, nil
}

func dbFromURL(u, path string) error {
	data, err := decodeURL(u)
	if err != nil {
		return err
	}

	storage, err := newDb(path)
	if err != nil {
		return err
	}
	defer storage.Close()

	for fn, fd := range data {
		storage.AddFile(fn, fd)
	}

	return nil
}

func main() {
	flag.Parse()

	err := dbFromURL("http://localhost:8675/src/", "test.db")
	if err != nil {
		log.Fatalf("Error making list: %v", err)
	}
}
