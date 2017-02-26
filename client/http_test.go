package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/dustin/bitfog"
)

type constantTransport struct {
	status int
	body   []byte
}

var errNotInitialized = errors.New("no transport")

func (c *constantTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if c == nil {
		return nil, errNotInitialized
	}
	return &http.Response{
		StatusCode: c.status,
		Status:     http.StatusText(c.status),
		Body:       ioutil.NopCloser(bytes.NewReader(c.body)),
	}, nil
}

func fakeClient(status int, body string) *bitfogClient {
	return &bitfogClient{&http.Client{Transport: &constantTransport{status, []byte(body)}}, posixFsOps}
}

func brokenClient() *bitfogClient {
	return &bitfogClient{&http.Client{Transport: (*constantTransport)(nil)}, posixFsOps}
}

func TestDecodeFail(t *testing.T) {
	c := brokenClient()
	rv, err := c.decodeURL("http://whatever/")
	if err == nil {
		t.Errorf("Expected failure, got %v", rv)
	}

	c = fakeClient(500, "Broken")
	rv, err = c.decodeURL("http://whatever/")
	if err == nil {
		t.Errorf("Expected failure, got %v", rv)
	}
}

func TestDecodeSuccess(t *testing.T) {
	res := `{"name": "a", "size": 37665, "mode": 644, "mtime": 1402853551, "hash": 90018}
{"name": "b", "size": 21866, "mode": 644, "mtime": 1402853551, "hash": 62130}
{"name": "c", "size": 75648, "mode": 644, "mtime": 1402853551, "hash": 51301, "linkdest": "a"}`

	c := fakeClient(200, res)
	rv, err := c.decodeURL("http://whatever/")
	if err != nil {
		t.Fatalf("Error decoding:  %v", err)
	}
	if len(rv) != 3 {
		t.Fatalf("Expected three results, got %v", rv)
	}
	exp := map[string]bitfog.FileData{
		"a": bitfog.FileData{Name: "a", Size: 37665, Mode: 644, Mtime: 1402853551, Hash: 90018},
		"b": bitfog.FileData{Name: "b", Size: 21866, Mode: 644, Mtime: 1402853551, Hash: 62130},
		"c": bitfog.FileData{Name: "c", Size: 75648, Mode: 644, Mtime: 1402853551, Hash: 51301, Dest: "a"},
	}

	for k, v := range exp {
		got := rv[k]
		if v != got {
			t.Errorf("Wrong at %q: wanted %#v, got %#v", k, v, got)
		}
	}
}

func TestDecodePartialSuccess(t *testing.T) {
	res := `{"name": "a", "size": 37665, "mode": 644, "mtime": 1402853551, "hash": 90018} blah`

	c := fakeClient(200, res)
	rv, err := c.decodeURL("http://whatever/")
	if err == nil {
		t.Fatalf("Expected an error, but didn't get one:  %v", err)
	}
	if len(rv) != 1 {
		t.Fatalf("Expected one result, got %v", rv)
	}
	exp := map[string]bitfog.FileData{
		"a": bitfog.FileData{Name: "a", Size: 37665, Mode: 644, Mtime: 1402853551, Hash: 90018},
	}

	for k, v := range exp {
		got := rv[k]
		if v != got {
			t.Errorf("Wrong at %q: wanted %#v, got %#v", k, v, got)
		}
	}
}

func TestDeleteFile(t *testing.T) {
	ctx := context.Background()
	c := fakeClient(204, "")
	err := c.deleteFile(ctx, "http://whatever/x")
	if err != nil {
		t.Errorf("Error trying delete: %v", err)
	}

	c = fakeClient(500, "")
	err = c.deleteFile(ctx, "http://whatever/x")
	if err == nil {
		t.Errorf("expected 500 error, but succeeded")
	}

	c = brokenClient()
	err = c.deleteFile(ctx, "http://whatever/x")
	if err == nil {
		t.Errorf("Expected error deleting, but succeeded")
	}

	c = brokenClient()
	err = c.deleteFile(ctx, "://whatever/x")
	if err == nil {
		t.Errorf("Expected error deleting, but succeeded")
	}
}

func TestCreateSymlink(t *testing.T) {
	ctx := context.Background()
	c := fakeClient(204, "")
	err := c.createSymlink(ctx, "y", "http://whatever/x")
	if err != nil {
		t.Errorf("Error trying to create symlink: %v", err)
	}

	c = brokenClient()
	err = c.createSymlink(ctx, "y", "http://whatever/x")
	if err == nil {
		t.Errorf("Expected error creating symlink, but succeeded")
	}

	c = brokenClient()
	err = c.createSymlink(ctx, "y", "://whatever/x")
	if err == nil {
		t.Errorf("Expected error creating symlink, but succeeded")
	}

	c = fakeClient(500, "")
	err = c.createSymlink(ctx, "y", "http://whatever/x")
	if err == nil {
		t.Errorf("expected 500 error, but succeeded")
	}
}

var errNoMore = errors.New("exhausted mocks are exhausted")

func mkFakeOps(creates []struct {
	wc  io.WriteCloser
	err error
},
	opens []struct {
		rc  io.ReadCloser
		err error
	},
	mkdirs []error) fsOps {

	return fsOps{
		func(string) (io.WriteCloser, error) {
			if len(creates) == 0 {
				return nil, errNoMore
			}
			el := creates[0]
			creates = creates[1:]
			return el.wc, el.err
		},
		func(string) (io.ReadCloser, error) {
			if len(opens) == 0 {
				return nil, errNoMore
			}
			el := opens[0]
			opens = opens[1:]
			return el.rc, el.err
		},
		func(string, os.FileMode) error {
			if len(mkdirs) == 0 {
				return errNoMore
			}
			el := mkdirs[0]
			mkdirs = mkdirs[1:]
			return el
		},
	}
}

type nopWriteCloser struct {
	io.Writer
}

func (n nopWriteCloser) Close() error {
	return nil
}

func TestDownload(t *testing.T) {
	ctx := context.Background()

	c := fakeClient(500, "")
	err := c.downloadFile(ctx, "http://whatever/x", "/tmp/some/path")
	if err == nil {
		t.Errorf("Expected error downloading")
	}

	c = brokenClient()
	err = c.downloadFile(ctx, "http://whatever/x", "/tmp/some/path")
	if err == nil {
		t.Errorf("Expected error downloading")
	}

	c = fakeClient(200, "content")
	c.fs = mkFakeOps(
		[]struct {
			wc  io.WriteCloser
			err error
		}{
			{nil, errors.New("nope")},
			{nil, errors.New("nope")},
		},
		nil,
		[]error{nil})
	err = c.downloadFile(ctx, "http://whatever/x", "/tmp/some/path")
	if err == nil {
		t.Errorf("Expected error trying to make dirs %v", err)
	}

	c = fakeClient(200, "content")
	c.fs = mkFakeOps(
		[]struct {
			wc  io.WriteCloser
			err error
		}{
			{nil, errors.New("nope")},
			{nopWriteCloser{ioutil.Discard}, nil},
		},
		nil,
		[]error{nil})
	err = c.downloadFile(ctx, "http://whatever/x", "/tmp/some/path")
	if err != nil {
		t.Errorf("Expected no error downloading, got %v", err)
	}
}

func TestUpload(t *testing.T) {
	ctx := context.Background()
	c := fakeClient(500, "")
	c.fs = mkFakeOps(nil,
		[]struct {
			rc  io.ReadCloser
			err error
		}{
			{nil, errors.New("nope")},
		},
		nil)
	err := c.uploadFile(ctx, "/tmp/some/path", "http://whatever/x")
	if err == nil {
		t.Errorf("Expected error uploading")
	}

	c = fakeClient(500, "")
	c.fs = mkFakeOps(nil,
		[]struct {
			rc  io.ReadCloser
			err error
		}{
			{ioutil.NopCloser(strings.NewReader("x")), nil},
		},
		nil)
	err = c.uploadFile(ctx, "/tmp/some/path", "http://whatever/x")
	if err == nil {
		t.Errorf("Expected error uploading")
	}

	c = brokenClient()
	c.fs = mkFakeOps(nil,
		[]struct {
			rc  io.ReadCloser
			err error
		}{
			{ioutil.NopCloser(strings.NewReader("x")), nil},
		},
		nil)
	err = c.uploadFile(ctx, "/tmp/some/path", "http://whatever/x")
	if err == nil {
		t.Errorf("Expected error uploading")
	}

	c = fakeClient(500, "")
	c.fs = mkFakeOps(nil,
		[]struct {
			rc  io.ReadCloser
			err error
		}{
			{ioutil.NopCloser(strings.NewReader("x")), nil},
		},
		nil)
	err = c.uploadFile(ctx, "/tmp/some/path", "://whatever/x")
	if err == nil {
		t.Errorf("Expected error uploading")
	}

	c = fakeClient(204, "")
	c.fs = mkFakeOps(nil,
		[]struct {
			rc  io.ReadCloser
			err error
		}{
			{ioutil.NopCloser(strings.NewReader("x")), nil},
		},
		nil)
	err = c.uploadFile(ctx, "/tmp/some/path", "http://whatever/x")
	if err != nil {
		t.Errorf("Unexpected error uploading: %v", err)
	}
}
