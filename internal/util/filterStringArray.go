package util

import (
	"strings"
)

// FilterStringArray filters an array of strings to return only those values containing a given partial
func FilterStringArray(array []string, partial string, logName string) []string {
	if len(partial) == 0 {
		if len(array) == 0 {
			return make([]string, 0)
		}
		return array
	}

	comparePartial := strings.ToLower(partial)
	filteredNames := make([]string, 0)
	for _, name := range array {
		if strings.Contains(strings.ToLower(name), comparePartial) {
			filteredNames = append(filteredNames, name)
		}
	}
	return filteredNames
}
