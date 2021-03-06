package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"sethwklein.net/go/errutil"

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

func (c *bitfogClient) decodeURL(ctx context.Context, u string) (map[string]bitfog.FileData, error) {
	rv := map[string]bitfog.FileData{}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	resp, err := c.client.Do(req)
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

func (c *bitfogClient) downloadFile(ctx context.Context, src, dest string) (err error) {
	req, err := http.NewRequest("GET", src, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	resp, err := c.client.Do(req)
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
	defer errutil.AppendCall(&err, f.Close)

	_, err = io.Copy(f, resp.Body)
	return err
}

func (c *bitfogClient) deleteFile(ctx context.Context, dest string) error {
	req, err := http.NewRequest("DELETE", dest, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
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

func (c *bitfogClient) uploadFile(ctx context.Context, src, dest string) error {
	srcfile, err := c.fs.Open(src)
	if err != nil {
		return err
	}
	defer srcfile.Close()

	req, err := http.NewRequest("PUT", dest, srcfile)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
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

func (c *bitfogClient) createSymlink(ctx context.Context, target, dest string) error {
	req, err := http.NewRequest("PUT", dest, strings.NewReader(target))
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
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
