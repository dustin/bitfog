package main

import (
	"encoding/json"
	"errors"
	"hash/crc64"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/bitfog"
)

var crcTable = crc64.MakeTable(crc64.ISO)

var SkipFile = errors.New("Skip this file.")

var flushInterval = (time.Duration(10) * time.Second)

func computeHash(path string) uint64 {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("Error in crc: %v", err)
		return 0
	}
	defer f.Close()
	h := crc64.New(crcTable)
	io.Copy(h, f)
	return h.Sum64()
}

func isa(mode os.FileMode, seeking os.FileMode) bool {
	return mode&seeking == seeking
}

func describe(p, fileName string, info os.FileInfo, checksum bool) (fd bitfog.FileData, err error) {
	fd.Name = fileName
	fd.Size = info.Size()
	fd.Mode = int32(info.Mode())
	fd.Mtime = info.ModTime().Unix()

	switch {
	default:
		if checksum {
			fd.Hash = computeHash(p)
		}
	case isa(info.Mode(), os.ModeSymlink):
		fd.Dest, err = os.Readlink(p)
		if err != nil {
			return
		}
	case isa(info.Mode(), os.ModeNamedPipe):
		log.Printf("Ignoring named pipe:  %v", p)
		return fd, SkipFile
	case isa(info.Mode(), os.ModeSocket):
		log.Printf("Ignoring socket:  %v", p)
		return fd, SkipFile
	}
	return
}

func listPath(conf itemConf, w http.ResponseWriter, req *http.Request) {
	e := json.NewEncoder(w)

	walking := conf.Path
	flusher, isFlusher := w.(http.Flusher)
	nextFlush := time.Now().Add(flushInterval)

	f := func(p string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Traversal error: %v", err)
		}
		if !info.IsDir() {
			if !strings.HasPrefix(p, walking) {
				log.Fatalf("Dir doesn't have prefix: %s %s", p, walking)
			}
			fileName := p[len(walking):]

			fd, err := describe(p, fileName, info, conf.Checksum)
			switch err {
			default:
				log.Printf("Error describing file: %v", err)
			case nil:
				e.Encode(fd)
				if isFlusher && time.Now().After(nextFlush) {
					flusher.Flush()
					nextFlush = time.Now().Add(flushInterval)
				}
			case SkipFile:
				// Just skipping htis
			}
		}
		return nil
	}

	filepath.Walk(walking, f)
}
