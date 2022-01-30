package redis_cache_provider

import (
	"math/rand"
	"time"
)

const DYNAMIC_TIME_BASE = 604800  //seconds 1 week
const STATIC_TIME_BASE = 2.419e+6 //seconds 4 weeks
const DYNAMIC_WINDOW = 259200     //seconds 3 days
const STATIC_WINDOW = 786240      //seconds 1.3 weeks

const COMPUTED_TIME_BASE = 259200 //seconds 3 days
const COMPUTED_WINDOW = 46800     //seconds 13 hours

func GetRandomWithWindow(base uint64, window uint64) time.Duration {
	var (
		high = base + window
		low  = base - window
	)

	return time.Duration(time.Millisecond * time.Duration((rand.Uint64()*(high-low) + low)))
}

func GetDynamicTimeWithShift() time.Duration {
	return (GetRandomWithWindow(DYNAMIC_TIME_BASE, DYNAMIC_WINDOW))
}

func GetStaticTimeWithShift() time.Duration {
	return (GetRandomWithWindow(STATIC_TIME_BASE, STATIC_WINDOW))
}

func GetComputedTimeWithShift() time.Duration {
	return (GetRandomWithWindow(COMPUTED_TIME_BASE, COMPUTED_WINDOW))
}
