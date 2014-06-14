package main

import (
	"encoding/gob"
	"os"

	"github.com/dustin/bitfog"
)

type db struct {
	path    string
	changed bool

	files map[string]bitfog.FileData
}

func (d *db) AddFile(name string, fd bitfog.FileData) error {
	d.files[name] = fd
	d.changed = true
	return nil
}

func (d *db) RmFile(name string) error {
	delete(d.files, name)
	d.changed = true
	return nil
}

func (d *db) Close() error {
	if !d.changed {
		return nil
	}

	f, err := os.Create(d.path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(d.files)
}

func newDb(path string) (db, error) {
	return db{path: path,
		changed: true,
		files:   make(map[string]bitfog.FileData),
	}, nil
}

func openDb(path string) (db, error) {
	rv := db{path: path, files: make(map[string]bitfog.FileData)}
	f, err := os.Open(path)
	if err != nil {
		return rv, nil
	}
	defer f.Close()
	return rv, gob.NewDecoder(f).Decode(&rv.files)
}
