package util

import (
	"iter"
	"slices"
	"strconv"
)

// Filter an array of arrays to an array of unique arrays
func FilterArrayToSetDouble[T comparable](array [][]T) (result [][]T) {
	for _, element := range array {
		found := false
		for _, existing := range result {
			if slices.Equal(existing, element) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, element)
		}
	}
	return
}

// Filter an array to a set
func FilterArrayToSet[T comparable](array []T) (result []T) {
	return SetFromSlice(array).ToSlice()
}

// Flatten an array of arrays of any type to an array of that type
func FlattenArray[T any](array [][]T) (return_array []T) {
	return slices.Concat(array...)
}

// Filter is a generic iterator-based filter
func Filter[T any](seq iter.Seq[T], f func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range seq {
			if f(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// Parse an array of strings to an array of uints
func ParseStringArrayToUint(array []string) []uint {
	var r []uint
	for _, s := range array {
		if hld, hldErr := strconv.ParseUint(s, 10, 64); hldErr == nil {
			r = append(r, uint(hld))
		}
	}
	return r
}
