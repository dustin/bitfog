package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/dustin/bitfog"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, `
  builddb url dbname     # build a database from the container URL
  emptydb dbname         # build an empty database (representing blank dest)
  fetch destdb src path  # fetch the missing items into a temp dir
  store srcdb dest path  # store fetched things into the dest

`)
		flag.PrintDefaults()
	}
}

var client = newBitfogClient()

func dbFromURL(u, path string) error {
	data, err := client.decodeURL(u)
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

func fetchTmp(path, src string, paths []string, fd map[string]bitfog.FileData) error {
	log.Printf("Fetching %d files", len(paths))

	for _, fn := range paths {
		if fd[fn].Dest == "" {
			log.Printf("  + %s", fn)
			dest := filepath.Join(path, fn)
			if err := client.downloadFile(src+fn, dest); err != nil {
				return err
			}
		}
	}
	return nil
}

func fetch() {
	if flag.NArg() < 4 {
		flag.Usage()
		os.Exit(1)
	}

	destdb, srcurl, tmpPath := flag.Arg(1), flag.Arg(2), flag.Arg(3)

	destData, err := openDb(destdb)
	if err != nil {
		log.Fatalf("Error reading DB:  %v", err)
	}
	defer destData.Close()

	log.Printf("Read %d files", len(destData.files))

	srcData, err := client.decodeURL(srcurl)
	if err != nil {
		log.Fatalf("Error reading from src: %s: %v", srcurl, err)
	}

	toadd, toremove := computeChanged(srcData, destData.files)

	err = os.RemoveAll(tmpPath)
	if err != nil {
		log.Fatalf("Error cleaning up tmp dir: %v", err)
	}
	err = os.Mkdir(tmpPath, 0777)
	if err != nil {
		log.Fatalf("Error recreating tmp dir: %v", err)
	}

	log.Printf("Need to add %d files, and remove %d", len(toadd), len(toremove))
	err = fetchTmp(tmpPath, srcurl, toadd, srcData)
	if err != nil {
		log.Fatalf("Error downloading file: %v", err)
	}
	for _, fn := range toremove {
		log.Printf("  - %s", fn)
	}
}

func store() {
	if flag.NArg() < 4 {
		flag.Usage()
		os.Exit(1)
	}

	srcdb, desturl, tmpPath := flag.Arg(1), flag.Arg(2), flag.Arg(3)

	srcData, err := openDb(srcdb)
	if err != nil {
		log.Fatalf("Error reading DB:  %v", err)
	}
	defer srcData.Close()

	log.Printf("Read %d files", len(srcData.files))

	destData, err := client.decodeURL(desturl)
	if err != nil {
		log.Fatalf("Error reading from dest: %s: %v", desturl, err)
	}

	toadd, toremove := computeChanged(srcData.files, destData)

	log.Printf("Need to add %d files, and remove %d around %s",
		len(toadd), len(toremove), tmpPath)

	for _, fn := range toremove {
		log.Printf(" - %s", fn)
		err = client.deleteFile(desturl + fn)
		if err != nil {
			log.Fatalf("Error deleting %s: %v", fn, err)
		}
	}

	for _, fn := range toadd {
		log.Printf(" + %s", fn)
		src := filepath.Join(tmpPath, fn)
		if srcData.files[fn].Dest == "" {
			err = client.uploadFile(src, desturl+fn)
		} else {
			err = client.createSymlink(srcData.files[fn].Dest, desturl+fn)
		}
		if err != nil {
			if !os.IsNotExist(err) {
				log.Fatalf("Error uploading %s: %#v", fn, err)
			}
		}
	}
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
	case "fetch":
		fetch()
	case "store":
		store()
	}
}
