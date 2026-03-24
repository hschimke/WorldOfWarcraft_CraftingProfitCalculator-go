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
				s1: Set[uint]{13: {}, 389374: {}, 894798749736: {}},
				s2: Set[uint]{13: {}, 389374: {}, 894798749736: {}},
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
				s1: Set[uint]{13: {}, 389374: {}, 8: {}},
				s2: Set[uint]{8: {}, 13: {}, 389374: {}},
			},
			want: true,
		},
		{
			name: "not equal same length",
			args: args{
				s1: Set[uint]{13: {}, 389374: {}, 100: {}},
				s2: Set[uint]{13: {}, 389374: {}, 101: {}},
			},
			want: false,
		},
		{
			name: "not equal different length",
			args: args{
				s1: Set[uint]{13: {}, 389374: {}, 100: {}},
				s2: Set[uint]{13: {}, 389374: {}},
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

func TestUintSet_Has(t *testing.T) {
	tests := []struct {
		name string
		set  Set[uint]
		args uint
		want bool
	}{
		{
			name: "multi has",
			set:  Set[uint]{13: {}, 389374: {}, 894798749736: {}, 8: {}},
			args: 894798749736,
			want: true,
		},
		{
			name: "multi does not have",
			set:  Set[uint]{13: {}, 389374: {}, 894798749736: {}, 8: {}},
			args: 6465,
			want: false,
		},
		{
			name: "empty",
			set:  Set[uint]{},
			args: 6465,
			want: false,
		},
		{
			name: "nil",
			set:  nil,
			args: 6465,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.Has(tt.args); got != tt.want {
				t.Errorf("Set.Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUintSet_Add(t *testing.T) {
	tests := []struct {
		name  string
		set   Set[uint]
		value uint
	}{
		{
			name:  "add new",
			set:   Set[uint]{13: {}, 389374: {}},
			value: 478,
		},
		{
			name:  "add existing",
			set:   Set[uint]{13: {}, 389374: {}},
			value: 13,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set == nil {
				tt.set = NewSet[uint]()
			}
			tt.set.Add(tt.value)
			if !tt.set.Has(tt.value) {
				t.Errorf("Set.Has() = false, want true after Add")
			}
		})
	}
}
