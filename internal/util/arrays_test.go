package util

import (
	"reflect"
	"testing"
)

func TestParseStringArrayToUint(t *testing.T) {
	type args struct {
		array []string
	}
	tests := []struct {
		name string
		args args
		want []uint
	}{
		{
			name: "all uints",
			args: args{array: []string{"1", "3", "38749", "89749784", "8372", "38", "0"}},
			want: []uint{1, 3, 38749, 89749784, 8372, 38, 0},
		},
		{
			name: "some uints",
			args: args{array: []string{"-1", "3", "387.49", "89749784", "hi", "38", "0"}},
			want: []uint{3, 89749784, 38, 0},
		},
		{
			name: "one uints",
			args: args{array: []string{"1"}},
			want: []uint{1},
		},
		{
			name: "no uints",
			args: args{array: []string{"-1"}},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseStringArrayToUint(tt.args.array); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseStringArrayToUint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlattenArray(t *testing.T) {
	type args struct {
		array [][]uint
	}
	tests := []struct {
		name             string
		args             args
		wantReturn_array []uint
	}{
		{
			name: "multiple elements",
			args: args{array: [][]uint{
				{345, 343, 23},
				{38, 342, 54},
				{985950, 3243243, 3},
				{3879438, 3, 3432},
			}},
			wantReturn_array: []uint{345, 343, 23, 38, 342, 54, 985950, 3243243, 3, 3879438, 3, 3432},
		},
		{
			name: "multiple elements different sizes",
			args: args{array: [][]uint{
				{345, 343, 23, 938},
				{38, 342, 54},
				{985950, 3},
				{3879438, 3, 3432},
			}},
			wantReturn_array: []uint{345, 343, 23, 938, 38, 342, 54, 985950, 3, 3879438, 3, 3432},
		},
		{
			name:             "single element",
			args:             args{array: [][]uint{{345, 343, 23}}},
			wantReturn_array: []uint{345, 343, 23},
		},
		{
			name:             "nil array",
			args:             args{array: nil},
			wantReturn_array: nil,
		},
		{
			name:             "empty array",
			args:             args{array: [][]uint{}},
			wantReturn_array: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotReturn_array := FlattenArray(tt.args.array); !reflect.DeepEqual(gotReturn_array, tt.wantReturn_array) {
				t.Errorf("FlattenArray() = %v, want %v", gotReturn_array, tt.wantReturn_array)
			}
		})
	}
}

func TestFilterArrayToSet(t *testing.T) {
	type args struct {
		array []uint
	}
	tests := []struct {
		name       string
		args       args
		wantResult []uint
	}{
		{
			name:       "no overlap",
			args:       args{array: []uint{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}},
			wantResult: []uint{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
		},
		{
			name:       "one overlap",
			args:       args{array: []uint{1, 2, 3, 4, 5, 5, 6, 7, 8, 9, 0}},
			wantResult: []uint{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
		},
		{
			name:       "many overlap",
			args:       args{array: []uint{1, 2, 3, 4, 5, 5, 5, 5, 5, 5, 6, 7, 8, 8, 9, 0}},
			wantResult: []uint{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
		},
		{
			name:       "empty",
			args:       args{array: []uint{}},
			wantResult: nil,
		}, {
			name:       "nil",
			args:       args{array: nil},
			wantResult: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//if gotResult := FilterArrayToSet(tt.args.array); !reflect.DeepEqual(gotResult, tt.wantResult) {
			if gotResult := FilterArrayToSet(tt.args.array); !SetEqual(*(SetFromSlice(gotResult)), *(SetFromSlice(tt.wantResult))) {
				t.Errorf("FilterArrayToSet() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func TestFilterArrayToSetDouble(t *testing.T) {
	type args struct {
		array [][]uint
	}
	tests := []struct {
		name       string
		args       args
		wantResult [][]uint
	}{
		{
			name:       "no overlap",
			args:       args{array: [][]uint{{1, 2, 3}, {4, 5, 6, 7}, {8, 9, 0}}},
			wantResult: [][]uint{{1, 2, 3}, {4, 5, 6, 7}, {8, 9, 0}},
		},
		{
			name:       "one overlap",
			args:       args{array: [][]uint{{1, 2, 3}, {4, 5, 6, 7}, {4, 5, 6, 7}, {8, 9, 0}}},
			wantResult: [][]uint{{1, 2, 3}, {4, 5, 6, 7}, {8, 9, 0}},
		},
		{
			name:       "many overlap",
			args:       args{array: [][]uint{{1, 2, 3}, {4, 5, 6, 7}, {4, 5, 6, 7}, {4, 5, 6, 7}, {4, 5, 6, 7}, {4, 5, 6, 7}, {4, 5, 6, 7}, {4, 5, 6, 7}, {4, 5, 6, 7}, {4, 5, 6, 7}, {8, 9, 0}, {8, 9, 0}}},
			wantResult: [][]uint{{1, 2, 3}, {4, 5, 6, 7}, {8, 9, 0}},
		},
		{
			name:       "empty",
			args:       args{array: [][]uint{}},
			wantResult: nil,
		}, {
			name:       "nil",
			args:       args{array: nil},
			wantResult: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotResult := FilterArrayToSetDouble(tt.args.array); !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("FilterArrayToSetDouble() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}
