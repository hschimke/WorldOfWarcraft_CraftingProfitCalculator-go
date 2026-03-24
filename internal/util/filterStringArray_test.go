package util

import (
	"reflect"
	"slices"
	"testing"
)

func TestFilterStringArray(t *testing.T) {
	type args struct {
		array   []string
		partial string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "no filter many",
			args: args{
				array:   []string{"lkdjf", "jlkje8", "iu4087dujO*", "jdflkj38h xz", "jdlk:KDJ"},
				partial: "",
			},
			want: []string{"lkdjf", "jlkje8", "iu4087dujO*", "jdflkj38h xz", "jdlk:KDJ"},
		},
		{
			name: "inexact filter many",
			args: args{
				array:   []string{"lkdjf", "jlkje8", "iu4087dujO*", "jdflkj38h xz", "jdlk:KDJ"},
				partial: "jd",
			},
			want: []string{"jdflkj38h xz", "jdlk:KDJ"},
		},
		{
			name: "exact filter many",
			args: args{
				array:   []string{"lkdjf", "jlkje8", "iu4087dujO*", "jdflkj38h xz", "jdlk:KDJ"},
				partial: "jdlk:KDJ",
			},
			want: []string{"jdlk:KDJ"},
		},
		{
			name: "no filter one",
			args: args{
				array:   []string{"lkdjf"},
				partial: "",
			},
			want: []string{"lkdjf"},
		},
		{
			name: "exact filter one",
			args: args{
				array:   []string{"jdlk:KDJ"},
				partial: "jdlk:KDJ",
			},
			want: []string{"jdlk:KDJ"},
		},
		{
			name: "no filter none",
			args: args{
				array:   []string{},
				partial: "",
			},
			want: []string{},
		},
		{
			name: "filter none",
			args: args{
				array:   []string{},
				partial: "jd",
			},
			want: []string{},
		},
		{
			name: "no filter nil",
			args: args{
				array:   nil,
				partial: "",
			},
			want: []string{},
		},
		{
			name: "filter nil",
			args: args{
				array:   nil,
				partial: "jd",
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slices.Collect(FilterStringArray(tt.args.array, tt.args.partial))
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterStringArray() = %v, want %v", got, tt.want)
			}
		})
	}
}
