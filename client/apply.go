package main

import (
	"sort"
)

// List of files that need to be added, removed.
func computeChanged(src, dest map[string]FileData) ([]string, []string) {
	toadd, toremove := []string{}, []string{}

	for k := range dest {
		if _, ok := src[k]; !ok {
			toremove = append(toremove, k)
		}
	}

	for srckey, srcval := range src {
		destval, found := dest[srckey]
		if found {
			if !srcval.Equals(destval) {
				toadd = append(toadd, srckey)
			}
		} else {
			toadd = append(toadd, srckey)
		}
	}

	fns := filenames{names: toadd, data: src}
	sort.Sort(&fns)

	return toadd, toremove
}

type filenames struct {
	names []string
	data  map[string]FileData
}

func (f *filenames) Len() int {
	return len(f.names)
}

func (f *filenames) Less(i, j int) bool {
	if _, ok := f.data[f.names[j]]; !ok {
		return false
	}
	if _, ok := f.data[f.names[i]]; !ok {
		return false
	}
	return f.data[f.names[j]].Size < f.data[f.names[i]].Size
}

func (f *filenames) Swap(i, j int) {
	f.names[i], f.names[j] = f.names[j], f.names[i]
}
