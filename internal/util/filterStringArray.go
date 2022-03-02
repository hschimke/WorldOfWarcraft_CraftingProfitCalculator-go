package util

import (
	"strings"
)

// Filter an array of strings to return only those values containing a given partial
func FilterStringArray(array []string, partial string, logName string) []string {
	var filteredNames []string
	if len(partial) > 0 {
		//cpclog.Debugf(`Partial search for all %s with "%s"`, logName, partial)
		comparePartial := strings.ToLower(partial)
		for _, name := range array {
			if strings.Contains(strings.ToLower(name), comparePartial) {
				filteredNames = append(filteredNames, name)
			}
		}

		if len(filteredNames) == 0 {
			filteredNames = make([]string, 0)
		}
	} else {
		//cpclog.Debug("Returning all unfiltered ", logName)
		filteredNames = array

		if len(filteredNames) == 0 {
			filteredNames = make([]string, 0)
		}
	}
	return filteredNames
}
