/**
 * Format a raw value into a string for Gold, Silver, and Copper
 * @param {!number} price_in The blizzard provided cost number.
 * @returns {string} The formatted Gold,Silver,Copper value as seen in game.
 */
function goldFormatter(price_in: number): string {
    const price = Math.trunc(price_in);
    const copper = price % 100;
    const silver = (((price % 10000) - copper)) / 100;
    const gold = (price - (price % 10000)) / 10000;
    return `${gold.toLocaleString()}g ${silver.toLocaleString()}s ${copper.toLocaleString()}c`;
}

/**
 * Provide a string to indent a preformatted text.
 * @param level The number of indents to include.
 */
function indentAdder(level: number): string {
    let str = '';
    for (let i = 0; i < level; i++) {
        str += '  ';
    }
    return str;
}

/**
 * Generate a preformatted text item price analysis and shopping list.
 * @param {!object} output_data The object created by generateOutputFormat.
 * @param {!number} indent The number of spaces the current level should be indented.
 */
function textFriendlyOutputFormat(output_data: OutputFormatObject, indent: number): string {
    /*
     * Output format:
     * Item
     *   Price Data (hih/low/average)
     *   Recipe Options
     *     Recipe
     *       Component Price
     *   Best Component Crafting Cost
     *   Worst Componenet Crafting Cost
     *   Average Component Crafting Cost
    */

    let return_string = '';

    //logger.debug('Building Formatted Price List');

    return_string += indentAdder(indent) + `${output_data.name} (${output_data.id}) Requires ${output_data.required}\n`;
    if ((output_data.ah !== undefined) && (output_data.ah.sales > 0)) {
        return_string += indentAdder(indent + 1) + `AH ${output_data.ah.sales}: ${goldFormatter(output_data.ah.high)}/${goldFormatter(output_data.ah.low)}/${goldFormatter(output_data.ah.average)}\n`;
    }
    if (output_data.vendor > 0) {
        return_string += indentAdder(indent + 1) + `Vendor ${goldFormatter(output_data.vendor)}\n`;
    }
    if (output_data.recipes !== undefined) {
        for (let recipe_option of output_data.recipes) {
            return_string += indentAdder(indent + 1) + `${recipe_option.name} - ${recipe_option.rank} - (${recipe_option.id}) : ${goldFormatter(recipe_option.high)}/${goldFormatter(recipe_option.low)}/${goldFormatter(recipe_option.average)}\n`;
            if ((recipe_option.ah !== undefined) && (recipe_option.ah.sales > 0)) {
                return_string += indentAdder(indent + 2) + `AH ${recipe_option.ah.sales}: ${goldFormatter(recipe_option.ah.high)}/${goldFormatter(recipe_option.ah.low)}/${goldFormatter(recipe_option.ah.average)}\n`;
            }
            return_string += '\n';
            if (recipe_option.parts !== undefined) {
                for (let opt of recipe_option.parts) {
                    return_string += textFriendlyOutputFormat(opt, indent + 2)
                    return_string += '\n'
                }
            }
        }
    }

    if (output_data.bonus_prices !== undefined) {
        for (const bonus_price of output_data.bonus_prices) {
            return_string += indentAdder(indent + 2) + `${output_data.name} (${output_data.id}) iLvl ${bonus_price.level}\n`;
            return_string += indentAdder(indent + 3) + `AH ${bonus_price.ah.sales}: ${goldFormatter(bonus_price.ah.high)}/${goldFormatter(bonus_price.ah.low)}/${goldFormatter(bonus_price.ah.average)}\n`;
        }
    }

    //logger.debug('Building formatted shopping list');
    // Add lists if it's appropriate
    if ('shopping_lists' in output_data && Object.keys(output_data.shopping_lists).length > 0) {
        return_string += indentAdder(indent) + `Shopping List For: ${output_data.name}\n`;
        for (let list of Object.keys(output_data.shopping_lists)) {
            return_string += indentAdder(indent + 1) + `List for rank ${list}\n`;
            for (let li of output_data.shopping_lists[list]) {
                return_string += indentAdder(indent + 2) + `[${li.quantity.toLocaleString().padStart(8, ' ')}] -- ${li.name} (${li.id})\n`;
                if (li.cost.vendor !== undefined) {
                    return_string += indentAdder(indent + 10);
                    return_string += `vendor: ${goldFormatter(li.cost.vendor)}\n`;
                }
                if (li.cost.ah !== undefined) {
                    return_string += indentAdder(indent + 10);
                    return_string += `ah: ${goldFormatter(li.cost.ah.high)}/${goldFormatter(li.cost.ah.low)}/${goldFormatter(li.cost.ah.average)}\n`;
                }
            }
        }
    }

    return return_string;
}

export { textFriendlyOutputFormat, goldFormatter };