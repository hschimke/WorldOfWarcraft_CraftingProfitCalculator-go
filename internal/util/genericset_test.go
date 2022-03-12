package util

import "testing"

func TestSetEqualUint(t *testing.T) {
	type args struct {
		s1 Set[uint]
		s2 Set[uint]
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "identical",
			args: args{
				s1: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: false},
				},
				s2: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: false},
				},
			},
			want: true,
		},
		{
			name: "empty",
			args: args{
				s1: Set[uint]{},
				s2: Set[uint]{},
			},
			want: true,
		},
		{
			name: "equal different order",
			args: args{
				s1: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 8: true, 100: false, 894798749736: true},
				},
				s2: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: false},
				},
			},
			want: true,
		},
		{
			name: "equal different length",
			args: args{
				s1: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 8: true, 100: false, 894798749736: true},
				},
				s2: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 8: true, 894798749736: true},
				},
			},
			want: true,
		},
		{
			name: "not equal same length",
			args: args{
				s1: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: true},
				},
				s2: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: false},
				},
			},
			want: false,
		},
		{
			name: "not equal different length",
			args: args{
				s1: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: true},
				},
				s2: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true},
				},
			},
			want: false,
		},
		{
			name: "not equal different length flip",
			args: args{
				s1: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true},
				},
				s2: Set[uint]{
					set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: true},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetEqual(tt.args.s1, tt.args.s2); got != tt.want {
				t.Errorf("SetEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
