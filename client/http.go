package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/bitfog"
)

func decodeURL(u string) (map[string]bitfog.FileData, error) {
	rv := map[string]bitfog.FileData{}

	resp, err := http.Get(u)
	if err != nil {
		return rv, err
	}
	if resp.StatusCode != 200 {
		return rv, fmt.Errorf("error httping: %v", resp.Status)
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
			return rv, fmt.Errorf("error decoding: %v", err)
		case nil:
			if verbose {
				fmt.Printf(" got %#v\n", fd)
			}
			rv[fd.Name] = bitfog.FileData{
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

func downloadFile(src, dest string) error {
	resp, err := http.Get(src)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("error getting %s: %v", src, resp.Status)
	}

	f, err := os.Create(dest)
	if err != nil {
		os.MkdirAll(filepath.Dir(dest), 0777)
		f, err = os.Create(dest)
		if err != nil {
			return err
		}
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func deleteFile(dest string) error {
	req, err := http.NewRequest("DELETE", dest, nil)
	if err != nil {
		return err
	}
	c := http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return errors.New(resp.Status)
	}
	return nil
}

func uploadFile(src, dest string) error {
	srcfile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcfile.Close()

	req, err := http.NewRequest("PUT", dest, srcfile)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	c := http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return errors.New(resp.Status)
	}
	return nil
}

func createSymlink(target, dest string) error {
	req, err := http.NewRequest("PUT", dest, strings.NewReader(target))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/symlink")

	c := http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return errors.New(resp.Status)
	}
	return nil
}
