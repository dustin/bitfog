package main

import "log"

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
				log.Printf("Data mismatch:  %s", srckey)
				toadd = append(toadd, srckey)
			}
		} else {
			log.Printf("Missing key:  %s", srckey)
			toadd = append(toadd, srckey)
		}
	}

	return toadd, toremove
}
