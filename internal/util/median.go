package util

import (
	"fmt"
	"maps"
	"slices"
)

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Median calculates the median of a slice of any numeric type
func Median[T Number](array []T) (float64, error) {
	if len(array) == 0 {
		return 0, fmt.Errorf("array cannot be empty")
	}

	sortedArray := slices.Clone(array)
	slices.Sort(sortedArray)
	length := len(sortedArray)
	var median float64
	if length%2 == 0 {
		middleNumbers := sortedArray[length/2-1 : length/2+1]
		median = (float64(middleNumbers[0]) + float64(middleNumbers[1])) / 2
	} else {
		median = float64(sortedArray[length/2])
	}
	return median, nil
}

// MedianFromMap calculates the median from a frequency map of values
func MedianFromMap[T Number](source map[T]uint64) (float64, error) {
	if len(source) == 0 {
		return 0, fmt.Errorf("array cannot be empty")
	}
	sum := uint64(0)
	for _, value := range source {
		sum += value
	}
	useMiddle := sum%2 == 0

	var returnValue float64

	keys := slices.Collect(maps.Keys(source))
	slices.Sort(keys)

	if useMiddle {
		target1 := sum / 2
		target2 := sum/2 + 1

		var pickup1, pickup2 float64
		found1, found2 := false, false

		runningTotal := uint64(0)
		for _, key := range keys {
			value := source[key]
			runningTotal += value

			if !found1 && runningTotal >= target1 {
				pickup1 = float64(key)
				found1 = true
			}
			if !found2 && runningTotal >= target2 {
				pickup2 = float64(key)
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
				returnValue = float64(key)
				break
			}
		}
	}

	return returnValue, nil
}
