package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

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
	c := fakeClient(204, "")
	err := c.deleteFile("http://whatever/x")
	if err != nil {
		t.Errorf("Error trying delete: %v", err)
	}

	c = fakeClient(500, "")
	err = c.deleteFile("http://whatever/x")
	if err == nil {
		t.Errorf("expected 500 error, but succeeded")
	}

	c = brokenClient()
	err = c.deleteFile("http://whatever/x")
	if err == nil {
		t.Errorf("Expected error deleting, but succeeded")
	}

	c = brokenClient()
	err = c.deleteFile("://whatever/x")
	if err == nil {
		t.Errorf("Expected error deleting, but succeeded")
	}
}

func TestCreateSymlink(t *testing.T) {
	c := fakeClient(204, "")
	err := c.createSymlink("y", "http://whatever/x")
	if err != nil {
		t.Errorf("Error trying to create symlink: %v", err)
	}

	c = brokenClient()
	err = c.createSymlink("y", "http://whatever/x")
	if err == nil {
		t.Errorf("Expected error creating symlink, but succeeded")
	}

	c = brokenClient()
	err = c.createSymlink("y", "://whatever/x")
	if err == nil {
		t.Errorf("Expected error creating symlink, but succeeded")
	}

	c = fakeClient(500, "")
	err = c.createSymlink("y", "http://whatever/x")
	if err == nil {
		t.Errorf("expected 500 error, but succeeded")
	}
}
