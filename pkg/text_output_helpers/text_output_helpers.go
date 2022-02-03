package text_output_helpers

import (
	"fmt"
	"strings"

	"github.com/hschimke/WorldOfWarcraft_CraftingProfitCalculator-go/pkg/globalTypes"
)

/**
 * Format a raw value into a string for Gold, Silver, and Copper
 * @param {!number} price_in The blizzard provided cost number.
 * @returns {string} The formatted Gold,Silver,Copper value as seen in game.
 */
func GoldFormatter(price_in float64) string {
	price := price_in
	copper := uint(price) % 100
	silver := ((uint(price) % 10000) - copper) / 100
	gold := (uint(price) - (uint(price) % 10000)) / 10000
	return fmt.Sprintf("%dg %ds %dc", gold, silver, copper)
}

/**
 * Provide a string to indent a preformatted text.
 * @param level The number of indents to include.
 */
func indentAdder(level uint) string {
	str := ""
	for i := uint(0); i < level; i++ {
		str += "  "
	}
	return str
}

/**
 * Generate a preformatted text item price analysis and shopping list.
 * @param {!object} output_data The object created by generateOutputFormat.
 * @param {!number} indent The number of spaces the current level should be indented.
 */
func TextFriendlyOutputFormat(output_data *globalTypes.OutputFormatObject, indent uint) string {

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

	var ob strings.Builder

	//logger.debug('Building Formatted Price List');

	ob.WriteString(indentAdder(indent))
	ob.WriteString(output_data.Name)
	ob.WriteString(" (")
	ob.WriteString(fmt.Sprint(output_data.Id))
	ob.WriteString(") Requires ")
	ob.WriteString(fmt.Sprint(output_data.Required))
	ob.WriteString("\n")

	if output_data.Ah.Sales > 0 {
		ob.WriteString(indentAdder(indent + 1))
		ob.WriteString("AH ")
		ob.WriteString(fmt.Sprint(output_data.Ah.Sales))
		ob.WriteString(": ")
		ob.WriteString(GoldFormatter(output_data.Ah.High))
		ob.WriteString("/")
		ob.WriteString(GoldFormatter(output_data.Ah.Low))
		ob.WriteString("/")
		ob.WriteString(GoldFormatter(output_data.Ah.Average))
		ob.WriteString("\n")
	}
	if output_data.Vendor > 0 {
		ob.WriteString(indentAdder(indent + 1))
		ob.WriteString("Vendor ")
		ob.WriteString(GoldFormatter(output_data.Vendor))
		ob.WriteString("\n")
	}
	if len(output_data.Recipes) > 0 {
		for _, recipe_option := range output_data.Recipes {
			ob.WriteString(indentAdder(indent + 1))
			ob.WriteString(recipe_option.Name)
			ob.WriteString(" - ")
			ob.WriteString(fmt.Sprint(recipe_option.Rank))
			ob.WriteString(" - (")
			ob.WriteString(fmt.Sprint(recipe_option.Id))
			ob.WriteString(") : ")
			ob.WriteString(GoldFormatter(recipe_option.High))
			ob.WriteString("/")
			ob.WriteString(GoldFormatter(recipe_option.Low))
			ob.WriteString("/")
			ob.WriteString(GoldFormatter(recipe_option.Average))
			ob.WriteString("\n")
			if recipe_option.Ah.Sales > 0 {
				ob.WriteString(indentAdder(indent + 2))
				ob.WriteString("AH ")
				ob.WriteString(fmt.Sprint(recipe_option.Ah.Sales))
				ob.WriteString(": ")
				ob.WriteString(GoldFormatter(recipe_option.Ah.High))
				ob.WriteString("/")
				ob.WriteString(GoldFormatter(recipe_option.Ah.Low))
				ob.WriteString("/")
				ob.WriteString(GoldFormatter(recipe_option.Ah.Average))
				ob.WriteString("\n")
			}
			ob.WriteString("\n")
			if len(recipe_option.Parts) > 0 {
				for _, opt := range recipe_option.Parts {
					ob.WriteString(TextFriendlyOutputFormat(&opt, indent+2))
					ob.WriteString("\n")
				}
			}
		}
	}

	if len(output_data.Bonus_prices) > 0 {
		for _, bonus_price := range output_data.Bonus_prices {
			ob.WriteString(indentAdder(indent + 2))
			ob.WriteString(output_data.Name)
			ob.WriteString("(")
			ob.WriteString(fmt.Sprint(output_data.Id))
			ob.WriteString(") iLvl ")
			ob.WriteString(fmt.Sprint(bonus_price.Level))
			ob.WriteString("\n")

			ob.WriteString(indentAdder(indent + 3))
			ob.WriteString("AH ")
			ob.WriteString(fmt.Sprint(bonus_price.Ah.Sales))
			ob.WriteString(": ")
			ob.WriteString(GoldFormatter(bonus_price.Ah.High))
			ob.WriteString("/")
			ob.WriteString(GoldFormatter(bonus_price.Ah.Low))
			ob.WriteString("/")
			ob.WriteString(GoldFormatter(bonus_price.Ah.Average))
			ob.WriteString("\n")
		}
	}

	//logger.debug('Building formatted shopping list');
	// Add lists if it's appropriate
	if len(output_data.Shopping_lists) > 0 {
		ob.WriteString(indentAdder(indent))
		ob.WriteString("Shopping List For: ")
		ob.WriteString(output_data.Name)
		ob.WriteString("\n")
		for rank, list := range output_data.Shopping_lists {
			ob.WriteString(indentAdder(indent + 1))
			ob.WriteString("List for rank ")
			ob.WriteString(fmt.Sprint(rank))
			ob.WriteString("\n")
			for _, li := range list {
				ob.WriteString(indentAdder(indent + 2))
				ob.WriteString("[")
				ob.WriteString(fmt.Sprintf("%8.0f", li.Quantity))
				ob.WriteString("] -- ")
				ob.WriteString(li.Name)
				ob.WriteString(" (")
				ob.WriteString(fmt.Sprint(li.Id))
				ob.WriteString(")\n")
				if li.Cost.Vendor != 0 {
					ob.WriteString(indentAdder(indent + 10))
					ob.WriteString("vendor: ")
					ob.WriteString(GoldFormatter(li.Cost.Vendor))
					ob.WriteString("\n")
				}
				if li.Cost.Ah.Sales != 0 {
					ob.WriteString(indentAdder(indent + 10))
					ob.WriteString("ah: ")
					ob.WriteString(GoldFormatter(li.Cost.Ah.High))
					ob.WriteString("/")
					ob.WriteString(GoldFormatter(li.Cost.Ah.Low))
					ob.WriteString("/")
					ob.WriteString(GoldFormatter(li.Cost.Ah.Average))
					ob.WriteString("\n")
				}
			}
		}
	}

	return ob.String()
}
