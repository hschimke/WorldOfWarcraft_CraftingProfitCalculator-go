function sqlToString(sql: string, values?: Array<string | number | boolean | null>): string {
    const value_str = values !== undefined ? values.map((val) => {
        return `[value: ${val} type: ${typeof (val)}] `;
    }) : '';
    return `${sql} : ${value_str}`;
}

export {sqlToString}