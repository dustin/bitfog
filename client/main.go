package main

import (
	"context"
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

func dbFromURL(ctx context.Context, u, path string) error {
	data, err := client.decodeURL(ctx, u)
	if err != nil {
		return err
	}

	storage, err := newDb(path)
	if err != nil {
		return err
	}
	defer storage.Close()

	for fn, fd := range data {
		if err := storage.AddFile(fn, fd); err != nil {
			return err
		}
	}

	return nil
}

func builddb(ctx context.Context) {
	if flag.NArg() < 3 {
		flag.Usage()
		os.Exit(1)
	}
	if err := dbFromURL(ctx, flag.Arg(1), flag.Arg(2)); err != nil {
		log.Fatalf("Error making list: %v", err)
	}
}

func emptydb(ctx context.Context) {
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

func fetchTmp(ctx context.Context, path, src string, paths []string, fd map[string]bitfog.FileData) error {
	log.Printf("Fetching %d files", len(paths))

	for _, fn := range paths {
		if fd[fn].Dest == "" {
			log.Printf("  + %s", fn)
			dest := filepath.Join(path, fn)
			if err := client.downloadFile(ctx, src+fn, dest); err != nil {
				return err
			}
		}
	}
	return nil
}

func fetch(ctx context.Context) {
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

	srcData, err := client.decodeURL(ctx, srcurl)
	if err != nil {
		log.Fatalf("Error reading from src: %s: %v", srcurl, err)
	}

	toadd, toremove := computeChanged(srcData, destData.files)

	if err := os.RemoveAll(tmpPath); err != nil {
		log.Fatalf("Error cleaning up tmp dir: %v", err)
	}
	if err := os.Mkdir(tmpPath, 0777); err != nil {
		log.Fatalf("Error recreating tmp dir: %v", err)
	}

	log.Printf("Need to add %d files, and remove %d", len(toadd), len(toremove))
	if err := fetchTmp(ctx, tmpPath, srcurl, toadd, srcData); err != nil {
		log.Fatalf("Error downloading file: %v", err)
	}
	for _, fn := range toremove {
		log.Printf("  - %s", fn)
	}
}

func store(ctx context.Context) {
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

	destData, err := client.decodeURL(ctx, desturl)
	if err != nil {
		log.Fatalf("Error reading from dest: %s: %v", desturl, err)
	}

	toadd, toremove := computeChanged(srcData.files, destData)

	log.Printf("Need to add %d files, and remove %d around %s",
		len(toadd), len(toremove), tmpPath)

	for _, fn := range toremove {
		log.Printf(" - %s", fn)
		if err := client.deleteFile(ctx, desturl+fn); err != nil {
			log.Fatalf("Error deleting %s: %v", fn, err)
		}
	}

	for _, fn := range toadd {
		log.Printf(" + %s", fn)
		src := filepath.Join(tmpPath, fn)
		if srcData.files[fn].Dest == "" {
			err = client.uploadFile(ctx, src, desturl+fn)
		} else {
			err = client.createSymlink(ctx, srcData.files[fn].Dest, desturl+fn)
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

	ctx := context.Background()

	switch flag.Arg(0) {
	default:
		flag.Usage()
	case "builddb":
		builddb(ctx)
	case "emptydb":
		emptydb(ctx)
	case "fetch":
		fetch(ctx)
	case "store":
		store(ctx)
	}
}
