package main

import (
	"flag"
	"log"
)

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
