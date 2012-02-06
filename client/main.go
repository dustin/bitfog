package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, `
  builddb url dbname  # build a database from the container URL
  emptydb dbname      # build an empty database (representing blank dest)
`)
		flag.PrintDefaults()

	}
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

func builddb() {
	if flag.NArg() < 3 {
		flag.Usage()
		os.Exit(1)
	}
	err := dbFromURL(flag.Arg(1), flag.Arg(2))
	if err != nil {
		log.Fatalf("Error making list: %v", err)
	}
}

func emptydb() {
	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(1)
	}
	storage, err := newDb(flag.Arg(1))
	if err != nil {
		log.Fatalf("Error creating empty DB:  %v", err)
	}
	defer storage.Close()
}

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	switch flag.Arg(0) {
	default:
		flag.Usage()
	case "builddb":
		builddb()
	case "emptydb":
		emptydb()
	}
}
