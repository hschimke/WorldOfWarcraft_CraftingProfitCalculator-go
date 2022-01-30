function getRegionCode(region_name: string): RegionCode {
    const check_str = region_name.toLowerCase();
    switch (check_str) {
        case 'us':
        case 'eu':
        case 'kr':
        case 'tw':
            return check_str;
        default:
            throw new Error(`'${region_name} is invalid. Valid regions include 'us', 'eu', 'kr', and 'tw'.`);

    }
}

export { getRegionCode };