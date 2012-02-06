package main

import (
	"encoding/gob"
	"os"
)

type db struct {
	path string

	files map[string]FileData
}

func (d *db) AddFile(name string, fd FileData) error {
	d.files[name] = fd
	return nil
}

func (d *db) RmFile(name string) error {
	delete(d.files, name)
	return nil
}

func (d *db) Close() error {
	f, err := os.Create(d.path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(d.files)
}

func newDb(path string) (db, error) {
	return db{path: path,
		files: make(map[string]FileData),
	}, nil
}
