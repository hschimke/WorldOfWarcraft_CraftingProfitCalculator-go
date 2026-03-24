package auction_history

import (
	"testing"
)

func TestCheckBonus(t *testing.T) {
	tests := []struct {
		name      string
		bonusList []uint
		target    []uint
		expected  bool
	}{
		{
			name:      "Empty requested bonuses matches anything",
			bonusList: []uint{},
			target:    []uint{1, 2, 3},
			expected:  true,
		},
		{
			name:      "Empty requested bonuses matches empty target",
			bonusList: []uint{},
			target:    []uint{},
			expected:  true,
		},
		{
			name:      "Requested bonuses with empty target fails",
			bonusList: []uint{1},
			target:    []uint{},
			expected:  false,
		},
		{
			name:      "All requested bonuses present",
			bonusList: []uint{1, 2},
			target:    []uint{1, 2, 3, 4},
			expected:  true,
		},
		{
			name:      "Some requested bonuses missing",
			bonusList: []uint{1, 5},
			target:    []uint{1, 2, 3, 4},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkBonus(tt.bonusList, tt.target)
			if result != tt.expected {
				t.Errorf("checkBonus(%v, %v) = %v, expected %v", tt.bonusList, tt.target, result, tt.expected)
			}
		})
	}
}

func TestBuildSQLWithAddins(t *testing.T) {
	baseSQL := "SELECT * FROM auctions"
	
	tests := []struct {
		name     string
		addins   []string
		expected string
	}{
		{
			name:     "No addins",
			addins:   []string{},
			expected: "SELECT * FROM auctions",
		},
		{
			name:     "One addin",
			addins:   []string{"item_id = $1"},
			expected: "SELECT * FROM auctions WHERE item_id = $1 ",
		},
		{
			name:     "Multiple addins",
			addins:   []string{"item_id = $1", "region = $2"},
			expected: "SELECT * FROM auctions WHERE item_id = $1 AND region = $2 ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSQLWithAddins(baseSQL, tt.addins)
			if result != tt.expected {
				t.Errorf("buildSQLWithAddins(%q, %v) = %q, expected %q", baseSQL, tt.addins, result, tt.expected)
			}
		})
	}
}