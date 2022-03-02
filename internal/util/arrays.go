package util

import (
	"fmt"
	"strconv"
)

// Filter an array or arrays to an array of unique arrays
func FilterArrayToSetDouble(array [][]uint) (result [][]uint) {
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
func FilterArrayToSet(array []uint) (result []uint) {
	hld := make(map[uint]bool)
	for _, element := range array {
		if _, present := hld[element]; !present {
			hld[element] = true
			result = append(result, element)
		}
	}
	return
}

// Flatten an array of arrays of uints to an array of uints
func FlattenArray(array [][]uint) (return_array []uint) {
	//return_array = make([]uint, 0)
	for _, sub_array := range array {
		return_array = append(return_array, sub_array...)
	}
	return
}

// Check if an array of uints contains a given uing
func ArrayContains(array []uint, search uint) (found bool) {
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

// Check if a uint slice contains a value
func UintSliceHas(arr []uint, value uint) (found bool) {
	return ArrayContains(arr, value)
}

// Check if an array includes a value
func ArrayIncludes(array []uint, search uint) bool {
	return ArrayContains(array, search)
}
