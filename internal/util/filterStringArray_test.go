package util

import (
	"reflect"
	"testing"
)

func TestFilterStringArray(t *testing.T) {
	type args struct {
		array   []string
		partial string
		logName string
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
				logName: "",
			},
			want: []string{"lkdjf", "jlkje8", "iu4087dujO*", "jdflkj38h xz", "jdlk:KDJ"},
		},
		{
			name: "inexact filter many",
			args: args{
				array:   []string{"lkdjf", "jlkje8", "iu4087dujO*", "jdflkj38h xz", "jdlk:KDJ"},
				partial: "jd",
				logName: "",
			},
			want: []string{"jdflkj38h xz", "jdlk:KDJ"},
		},
		{
			name: "exact filter many",
			args: args{
				array:   []string{"lkdjf", "jlkje8", "iu4087dujO*", "jdflkj38h xz", "jdlk:KDJ"},
				partial: "jdlk:KDJ",
				logName: "",
			},
			want: []string{"jdlk:KDJ"},
		},
		{
			name: "no filter one",
			args: args{
				array:   []string{"lkdjf"},
				partial: "",
				logName: "",
			},
			want: []string{"lkdjf"},
		},
		{
			name: "exact filter one",
			args: args{
				array:   []string{"jdlk:KDJ"},
				partial: "jdlk:KDJ",
				logName: "",
			},
			want: []string{"jdlk:KDJ"},
		},
		{
			name: "no filter none",
			args: args{
				array:   []string{},
				partial: "",
				logName: "",
			},
			want: []string{},
		},
		{
			name: "filter none",
			args: args{
				array:   []string{},
				partial: "jd",
				logName: "",
			},
			want: []string{},
		},
		{
			name: "no filter nil",
			args: args{
				array:   nil,
				partial: "",
				logName: "",
			},
			want: []string{},
		},
		{
			name: "filter nil",
			args: args{
				array:   nil,
				partial: "jd",
				logName: "",
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterStringArray(tt.args.array, tt.args.partial, tt.args.logName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterStringArray() = %v, want %v", got, tt.want)
			}
		})
	}
}
