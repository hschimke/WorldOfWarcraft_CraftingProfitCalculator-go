const DYNAMIC_TIME_BASE = 604800; //seconds 1 week
const STATIC_TIME_BASE = 2.419e+6; //seconds 4 weeks
const DYNAMIC_WINDOW = 259200; //seconds 3 days
const STATIC_WINDOW = 786240; //seconds 1.3 weeks

const COMPUTED_TIME_BASE = 259200; //seconds 3 days
const COMPUTED_WINDOW = 46800; //seconds 13 hours

function getRandomWithWindow(base: number, window: number): number {
    const high = base + window;
    const low = base - window;
    return Math.floor(
        Math.random() * (high - low) + low
    )
}

function getDynamicTimeWithShift() {
    return (getRandomWithWindow(DYNAMIC_TIME_BASE, DYNAMIC_WINDOW));
}

function getStaticTimeWithShift() {
    return (getRandomWithWindow(STATIC_TIME_BASE, STATIC_WINDOW));
}

function getComputedTimeWithShift() {
    return (getRandomWithWindow(COMPUTED_TIME_BASE, COMPUTED_WINDOW));
}

function convertToMS(seconds: number) {
    return seconds * 1000;
}

export { DYNAMIC_TIME_BASE, STATIC_TIME_BASE, getDynamicTimeWithShift, getStaticTimeWithShift, convertToMS, getComputedTimeWithShift }