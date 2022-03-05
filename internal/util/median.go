package util

import (
	"fmt"
	"sort"
)

func Median(array []float64) (float64, error) {
	sortedArray := array //sort.Float64Slice(array)
	sort.Float64s(sortedArray)
	length := len(sortedArray)
	var median float64
	if length == 0 {
		return 0, fmt.Errorf("array cannot be empty")
	} else if length%2 == 0 {
		middleNumbers := sortedArray[length/2-1 : length/2+1]
		median = (middleNumbers[0] + middleNumbers[1]) / 2
	} else {
		median = sortedArray[length/2]
	}
	return median, nil
}

func MedianFromMap(source map[float64]uint64) (float64, error) {
	if len(source) == 0 {
		return 0, fmt.Errorf("array cannot be empty")
	}
	sum := uint64(0)
	for _, value := range source {
		sum += value
	}
	useMiddle := sum%2 == 0

	var returnValue float64

	//setupMap
	keys := make([]float64, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Float64s(keys)

	if useMiddle {
		target1 := sum/2 - 1
		target2 := sum / 2

		pickup1 := float64(0)
		pickup2 := float64(0)

		found1 := false
		found2 := false

		runningTotal := uint64(0)
		for _, key := range keys {
			value := source[key]
			runningTotal += value

			if runningTotal >= target1 && !found1 {
				pickup1 = key
				found1 = true
			}
			if runningTotal >= target2 && !found2 {
				pickup2 = key
				found2 = true
			}
			if found1 && found2 {
				returnValue = (pickup1 + pickup2) / 2
				break
			}
		}
	} else {
		target := sum / 2

		runningTotal := uint64(0)
		for _, key := range keys {
			value := source[key]
			runningTotal += value
			if runningTotal >= target {
				returnValue = key
				break
			}
		}
	}

	return returnValue, nil
}
