package util

import (
	"iter"
	"strings"
)

// FilterStringArray filters an array of strings to return only those values containing a given partial
func FilterStringArray(array []string, partial string) iter.Seq[string] {
	comparePartial := strings.ToLower(partial)
	return func(yield func(string) bool) {
		for _, name := range array {
			if strings.Contains(strings.ToLower(name), comparePartial) {
				if !yield(name) {
					return
				}
			}
		}
	}
}
