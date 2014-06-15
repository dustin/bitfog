package main

import (
	"testing"

	"github.com/dustin/bitfog"
)

func diffSet(from, to []string) []string {
	var rv []string
	m := make(map[string]struct{})
	for _, x := range from {
		m[x] = struct{}{}
	}
	for _, x := range to {
		if _, ok := m[x]; !ok {
			rv = append(rv, x)
		}
	}
	return rv
}

func TestFilenamesDiffing(t *testing.T) {
	tests := []struct {
		name          string
		src, dest     map[string]bitfog.FileData
		expAdd, expRm []string
	}{
		{"Dest Empty",
			map[string]bitfog.FileData{
				"a": bitfog.FileData{Size: 717255, Hash: 643476},
				"b": bitfog.FileData{Size: 619280, Hash: 237139},
			},
			map[string]bitfog.FileData{},
			[]string{"a", "b"},
			nil},

		{"Src Empty",
			map[string]bitfog.FileData{},
			map[string]bitfog.FileData{
				"a": bitfog.FileData{Size: 717255, Hash: 643476},
				"b": bitfog.FileData{Size: 619280, Hash: 237139},
			},
			nil,
			[]string{"a", "b"}},

		{"Both Empty",
			map[string]bitfog.FileData{},
			map[string]bitfog.FileData{},
			nil, nil},

		{"Both same",
			map[string]bitfog.FileData{
				"a": bitfog.FileData{Size: 717255, Hash: 643476},
				"b": bitfog.FileData{Size: 619280, Hash: 237139},
			},
			map[string]bitfog.FileData{
				"a": bitfog.FileData{Size: 717255, Hash: 643476},
				"b": bitfog.FileData{Size: 619280, Hash: 237139},
			},
			nil, nil},

		{"One diff",
			map[string]bitfog.FileData{
				"a": bitfog.FileData{Size: 717255, Hash: 643476},
				"b": bitfog.FileData{Size: 619280, Hash: 237139},
			},
			map[string]bitfog.FileData{
				"a": bitfog.FileData{Size: 717255, Hash: 643476},
				"b": bitfog.FileData{Size: 753519, Hash: 237139},
			},
			[]string{"b"},
			nil},

		{"One diff and remove",
			map[string]bitfog.FileData{
				"a": bitfog.FileData{Size: 717255, Hash: 643476},
				"b": bitfog.FileData{Size: 619280, Hash: 237139},
			},
			map[string]bitfog.FileData{
				"a": bitfog.FileData{Size: 717255, Hash: 643476},
				"b": bitfog.FileData{Size: 753519, Hash: 237139},
				"c": bitfog.FileData{Size: 372911, Hash: 634543},
			},
			[]string{"b"},
			[]string{"c"}},
	}

	for _, test := range tests {
		gotAdd, gotRm := computeChanged(test.src, test.dest)
		for _, missing := range diffSet(gotAdd, test.expAdd) {
			t.Errorf("%v: Expected to add %q, but wouldn't", test.name, missing)
		}
		for _, missing := range diffSet(test.expAdd, gotAdd) {
			t.Errorf("%v: Expected not to add %q, but would", test.name, missing)
		}
		for _, missing := range diffSet(gotRm, test.expRm) {
			t.Errorf("%v: Expected to rm %q, but wouldn't", test.name, missing)
		}
		for _, missing := range diffSet(test.expRm, gotRm) {
			t.Errorf("%v: Expected not to rm %q, but would", test.name, missing)
		}
	}
}

func TestFilenamesSorting(t *testing.T) {
	fns := filenames{
		[]string{"a", "b", "c"},
		map[string]bitfog.FileData{
			"a": bitfog.FileData{Size: 75278, Hash: 75589},
			"b": bitfog.FileData{Size: 28398, Hash: 00024},
		},
	}

	tests := []struct {
		i, j int
		exp  bool
	}{
		{0, 1, true},
		{1, 0, false},
		{1, 2, false},
		{2, 1, false},
	}

	if fns.Len() != 3 {
		t.Errorf("Expected three names, got %v", fns.Len())
	}

	for _, test := range tests {
		got := fns.Less(test.i, test.j)
		if got != test.exp {
			t.Errorf("Expected less=%v for %v,%v, got %v",
				test.exp, test.i, test.j, got)
		}
	}

	fns.Swap(0, 1)
	if fns.names[0] != "b" {
		t.Errorf("Expected 0 to be b, got %v", fns.names[0])
	}
	if fns.names[1] != "a" {
		t.Errorf("Expected 0 to be a, got %v", fns.names[1])
	}
}
