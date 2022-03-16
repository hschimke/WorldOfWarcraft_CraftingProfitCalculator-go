package util

import (
	"fmt"
	"strconv"
)

// Filter an array or arrays to an array of unique arrays
func FilterArrayToSetDouble[T comparable](array [][]T) (result [][]T) {
	hld := make(map[string]bool)
	for _, element := range array {
		srch := fmt.Sprint(element)
		if _, present := hld[srch]; !present {
			hld[srch] = true
			result = append(result, element)
		}
	}
	return
}

// Filter an array to a set
func FilterArrayToSet[T comparable](array []T) (result []T) {
	return SetFromSlice(array).ToSlice()
}

// Flatten an array of arrays of uints to an array of uints
func FlattenArray[T comparable](array [][]T) (return_array []T) {
	for _, sub_array := range array {
		return_array = append(return_array, sub_array...)
	}
	return
}

// Parse an array of strinsg to an array of uints
func ParseStringArrayToUint(array []string) []uint {
	var r []uint
	for _, s := range array {
		if hld, hldErr := strconv.ParseUint(s, 10, 64); hldErr == nil {
			r = append(r, uint(hld))
		}
	}
	return r
}
