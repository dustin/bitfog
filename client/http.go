package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/bitfog"
	"github.com/dustin/httputil"
)

type fsOps struct {
	Create   func(string) (io.WriteCloser, error)
	Open     func(string) (io.ReadCloser, error)
	MkdirAll func(string, os.FileMode) error
}

type bitfogClient struct {
	client *http.Client
	fs     fsOps
}

var posixFsOps = fsOps{
	func(s string) (io.WriteCloser, error) {
		return os.Create(s)
	},
	func(s string) (io.ReadCloser, error) {
		return os.Open(s)
	},
	os.MkdirAll,
}

func newBitfogClient() *bitfogClient {
	return &bitfogClient{&http.Client{}, posixFsOps}
}

func (c *bitfogClient) decodeURL(u string) (map[string]bitfog.FileData, error) {
	rv := map[string]bitfog.FileData{}

	resp, err := c.client.Get(u)
	if err != nil {
		return rv, err
	}
	if resp.StatusCode != 200 {
		return rv, httputil.HTTPErrorf(resp, "Error fetching %v - %S\n%B", u)
	}
	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)

	for {
		fd := bitfog.FileData{}
		err = d.Decode(&fd)
		switch err {
		default:
			return rv, fmt.Errorf("error decoding %v: %v", u, err)
		case nil:
			rv[fd.Name] = fd
		case io.EOF:
			return rv, nil
		}
	}
}

func (c *bitfogClient) downloadFile(src, dest string) error {
	resp, err := c.client.Get(src)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return httputil.HTTPErrorf(resp, "error getting %v - %S\n%B", src)
	}

	f, err := c.fs.Create(dest)
	if err != nil {
		c.fs.MkdirAll(filepath.Dir(dest), 0777)
		f, err = c.fs.Create(dest)
		if err != nil {
			return err
		}
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func (c *bitfogClient) deleteFile(dest string) error {
	req, err := http.NewRequest("DELETE", dest, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return httputil.HTTPError(resp)
	}
	return nil
}

func (c *bitfogClient) uploadFile(src, dest string) error {
	srcfile, err := c.fs.Open(src)
	if err != nil {
		return err
	}
	defer srcfile.Close()

	req, err := http.NewRequest("PUT", dest, srcfile)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return httputil.HTTPError(resp)
	}
	return nil
}

func (c *bitfogClient) createSymlink(target, dest string) error {
	req, err := http.NewRequest("PUT", dest, strings.NewReader(target))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/symlink")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return httputil.HTTPError(resp)
	}
	return nil
}
