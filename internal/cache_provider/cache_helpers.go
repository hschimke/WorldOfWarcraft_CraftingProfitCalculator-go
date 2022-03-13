package cache_provider

import (
	"math/rand"
	"time"
)

const singleDAY = time.Hour * 24
const singleWEEK = singleDAY * 7
const DYNAMIC_TIME_BASE = singleWEEK                 //604800  //seconds 1 week
const STATIC_TIME_BASE = DYNAMIC_TIME_BASE * 4       //2.419e+6            //seconds 4 weeks
const DYNAMIC_WINDOW = singleDAY * 3                 //259200                  //seconds 3 days
const STATIC_WINDOW = (singleWEEK) + (singleDAY + 3) //786240                   //seconds 1.3 weeks

const COMPUTED_TIME_BASE = singleDAY * 3 //259200 //seconds 3 days
const COMPUTED_WINDOW = time.Hour * 13   //46800     //seconds 13 hours

// Get a random expiration window
func GetRandomWithWindow(base time.Duration, window time.Duration) time.Duration {
	var (
		high = base + window
		low  = base - window
	)
	hms := high.Microseconds()
	lms := low.Microseconds()
	microsec_window := rand.Int63n(hms-lms) + lms
	return time.Duration(time.Microsecond * time.Duration(microsec_window))
}

// Get a random expiration for a dynamic API result
func GetDynamicTimeWithShift() time.Duration {
	return (GetRandomWithWindow(DYNAMIC_TIME_BASE, DYNAMIC_WINDOW))
}

// Get a random expiration for a static API result
func GetStaticTimeWithShift() time.Duration {
	return (GetRandomWithWindow(STATIC_TIME_BASE, STATIC_WINDOW))
}

// Get a random expiration for a computed result
func GetComputedTimeWithShift() time.Duration {
	return (GetRandomWithWindow(COMPUTED_TIME_BASE, COMPUTED_WINDOW))
}
