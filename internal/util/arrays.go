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
	hld := make(map[T]bool)
	for _, element := range array {
		if _, present := hld[element]; !present {
			hld[element] = true
			result = append(result, element)
		}
	}
	return result
}

// Flatten an array of arrays of uints to an array of uints
func FlattenArray[T comparable](array [][]T) (return_array []T) {
	//return_array = make([]uint, 0)
	for _, sub_array := range array {
		return_array = append(return_array, sub_array...)
	}
	return
}

// Check if an array of uints contains a given uing
func ArrayContains[T comparable](array []T, search T) (found bool) {
	found = false
	for _, item := range array {
		if item == search {
			found = true
		}
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

// SliceEqual checks that two slices are exactly equal, including order
func SlicesEqual[T comparable](slice1 []T, slice2 []T) bool {
	found := true
	if len(slice1) != len(slice2) {
		return false
	}
	for index, element := range slice1 {
		found = found && element == slice2[index]
	}
	return found
}

// Check if a uint slice contains a value
func UintSliceHas[T comparable](arr []T, value T) (found bool) {
	return ArrayContains(arr, value)
}

// Check if an array includes a value
func ArrayIncludes[T comparable](array []T, search T) bool {
	return ArrayContains(array, search)
}
