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

func TestUintSet_Has(t *testing.T) {
	type fields struct {
		set map[uint]bool
	}
	type args struct {
		check uint
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "multi has",
			fields: fields{set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: false}},
			args:   args{check: 894798749736},
			want:   true,
		},
		{
			name:   "multi does not have",
			fields: fields{set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: false}},
			args:   args{check: 6465},
			want:   false,
		},
		{
			name:   "multi does not have (with remove)",
			fields: fields{set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: false}},
			args:   args{check: 100},
			want:   false,
		},
		{
			name:   "empty",
			fields: fields{set: map[uint]bool{}},
			args:   args{check: 6465},
			want:   false,
		},
		{
			name:   "nil",
			fields: fields{set: nil},
			args:   args{check: 6465},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Set[uint]{
				set: tt.fields.set,
			}
			if got := s.Has(tt.args.check); got != tt.want {
				t.Errorf("UintSet.Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUintSet_Add(t *testing.T) {
	type fields struct {
		set map[uint]bool
	}
	type args struct {
		value uint
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		expects bool
	}{
		{
			name:    "add new",
			fields:  fields{set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: false}},
			args:    args{value: 478},
			expects: true,
		},
		{
			name:    "add existing",
			fields:  fields{set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: false}},
			args:    args{value: 13},
			expects: true,
		},
		{
			name:    "re-add existing false",
			fields:  fields{set: map[uint]bool{13: true, 389374: true, 894798749736: true, 8: true, 100: false}},
			args:    args{value: 100},
			expects: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Set[uint]{
				set: tt.fields.set,
			}
			s.Add(tt.args.value)
			if got := s.Has(tt.args.value); got != tt.expects {
				t.Errorf("UintSet.Has() = %v, want %v", got, tt.expects)
			}
		})
	}
}

func TestStringSet_Has(t *testing.T) {
	type fields struct {
		set map[string]bool
	}
	type args struct {
		check string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "multi has",
			fields: fields{set: map[string]bool{"13": true, "389374": true, "894798749736": true, "8": true, "100": false}},
			args:   args{check: "894798749736"},
			want:   true,
		},
		{
			name:   "multi does not have",
			fields: fields{set: map[string]bool{"13": true, "389374": true, "894798749736": true, "8": true, "100": false}},
			args:   args{check: "6465"},
			want:   false,
		},
		{
			name:   "multi does not have (with remove)",
			fields: fields{set: map[string]bool{"13": true, "389374": true, "894798749736": true, "8": true, "100": false}},
			args:   args{check: "100"},
			want:   false,
		},
		{
			name:   "empty",
			fields: fields{set: map[string]bool{}},
			args:   args{check: "6465"},
			want:   false,
		},
		{
			name:   "nil",
			fields: fields{set: nil},
			args:   args{check: "6465"},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Set[string]{
				set: tt.fields.set,
			}
			if got := s.Has(tt.args.check); got != tt.want {
				t.Errorf("UintSet.Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringSet_Add(t *testing.T) {
	type fields struct {
		set map[string]bool
	}
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		expects bool
	}{
		{
			name:    "add new",
			fields:  fields{set: map[string]bool{"13": true, "389374": true, "894798749736": true, "8": true, "100": false}},
			args:    args{value: "478"},
			expects: true,
		},
		{
			name:    "add existing",
			fields:  fields{set: map[string]bool{"13": true, "389374": true, "894798749736": true, "8": true, "100": false}},
			args:    args{value: "13"},
			expects: true,
		},
		{
			name:    "re-add existing false",
			fields:  fields{set: map[string]bool{"13": true, "389374": true, "894798749736": true, "8": true, "100": false}},
			args:    args{value: "100"},
			expects: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Set[string]{
				set: tt.fields.set,
			}
			s.Add(tt.args.value)
			if got := s.Has(tt.args.value); got != tt.expects {
				t.Errorf("UintSet.Has() = %v, want %v", got, tt.expects)
			}
		})
	}
}
