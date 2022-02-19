package util

import (
	"fmt"
	"strconv"
)

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

func FlattenArray(array [][]uint) (return_array []uint) {
	//return_array = make([]uint, 0)
	for _, sub_array := range array {
		return_array = append(return_array, sub_array...)
	}
	return
}

func ArrayContains(array []uint, search uint) (found bool) {
	found = false
	for _, item := range array {
		if item == search {
			found = true
		}
	}
	return
}

func ParseStringArrayToUint(array []string) []uint {
	var r []uint
	for _, s := range array {
		if hld, hldErr := strconv.ParseUint(s, 10, 64); hldErr == nil {
			r = append(r, uint(hld))
		}
	}
	return r
}

func UintSliceHas(arr []uint, value uint) (found bool) {
	return ArrayContains(arr, value)
}

func ArrayIncludes(array []uint, search uint) bool {
	return ArrayContains(array, search)
}
