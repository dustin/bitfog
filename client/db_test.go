package main

import (
	"os"
	"testing"

	"github.com/dustin/bitfog"
)

const testDbName = ",test.db"

func TestDBSimple(t *testing.T) {
	defer os.Remove(testDbName)
	db, err := newDb(testDbName)
	if err != nil {
		t.Fatalf("Error getting test db: %v", err)
	}
	if !db.changed {
		t.Errorf("Expected new DB to be changed. Wasn't")
	}
	err = db.AddFile("/path",
		bitfog.FileData{Name: "path", Size: 1732, Mode: 0644, Hash: 54857})
	if err != nil {
		t.Errorf("Error adding: %v", err)
	}

	err = db.AddFile("/path2",
		bitfog.FileData{Name: "path2", Size: 40550, Mode: 0644, Hash: 60957})
	if err != nil {
		t.Errorf("Error adding: %v", err)
	}

	err = db.RmFile("/path2")
	if err != nil {
		t.Errorf("Error removing: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Errorf("Error closing db: %v", err)
	}

	db, err = openDb(testDbName + ".nonexistent")
	if err == nil {
		t.Errorf("Expected to fail to open nonexistent DB, got %v", db)
	}

	db, err = openDb(testDbName)
	if err != nil {
		t.Fatalf("error reopening db: %v", err)
	}
	if db.changed {
		t.Errorf("Newly open DB is marked as changed")
	}
	if len(db.files) != 1 {
		t.Errorf("Expected state to have one file, has %v", db.files)
	}

}
