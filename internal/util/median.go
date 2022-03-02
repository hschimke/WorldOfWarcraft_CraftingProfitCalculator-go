package util

import (
	"fmt"
	"math"
	"sort"
)

func Median(array []float64) (float64, error) {
	sortedArray := sort.Float64Slice(array)
	length := len(sortedArray)
	var median float64
	if length == 0 {
		return math.NaN(), fmt.Errorf("array cannot be empty")
	} else if length%2 == 0 {
		middleNumbers := sortedArray[length/2-1 : length/2+1]
		median = (middleNumbers[0] + middleNumbers[1]) / 2
	} else {
		median = sortedArray[length/2]
	}
	return median, nil
}
