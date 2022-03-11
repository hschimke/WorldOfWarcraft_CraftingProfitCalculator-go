package util

import (
	"testing"
)

func TestMedian(t *testing.T) {
	type args struct {
		array []float64
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{name: "odd",
			args: args{
				array: []float64{34, 46, 46, 23, 3, 23, 3, 3, 3356.56, 565, 3, 4, 4, 65, 7, 7, 6, 34, 5, 47, 57, 35, 6, 345, 4, 5, 7, 5, 65, 345, 345, 4, 5, 7, 456, 456, 575, 47, 57, 45, 64, 5, 436, 45, 7, 457, 4, 56, 45, 7, 45, 74, 564, 56, 45, 7, 457, 46, 453, 45, 4},
			},
			want:    45,
			wantErr: false,
		},
		{name: "even",
			args: args{
				array: []float64{34, 46, 9, 46, 23, 3, 23, 3, 3, 3356.56, 565, 3, 4, 4, 65, 7, 7, 6, 34, 5, 47, 57, 35, 6, 345, 4, 5, 7, 5, 65, 345, 345, 4, 5, 7, 456, 456, 575, 47, 57, 45, 64, 5, 436, 45, 7, 457, 4, 56, 45, 7, 45, 74, 564, 56, 45, 7, 457, 46, 453, 45, 4},
			},
			want:    45,
			wantErr: false,
		},
		{
			name: "1",
			args: args{
				array: []float64{5},
			},
			want:    5,
			wantErr: false,
		},
		{
			name: "2",
			args: args{
				array: []float64{5, 5},
			},
			want:    5,
			wantErr: false,
		},
		{
			name: "2 - dif",
			args: args{
				array: []float64{5, 7},
			},
			want:    6,
			wantErr: false,
		},
		{name: "empty",
			args: args{
				array: []float64{},
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Median(tt.args.array)
			if (err != nil) != tt.wantErr {
				t.Errorf("Median() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Median() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedianFromMap(t *testing.T) {
	type args struct {
		source map[float64]uint64
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				source: nil,
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "empty",
			args: args{
				source: make(map[float64]uint64),
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "one odd",
			args: args{
				source: map[float64]uint64{
					5: 5,
				},
			},
			want:    5,
			wantErr: false,
		},
		{
			name: "one even",
			args: args{
				source: map[float64]uint64{
					5: 6,
				},
			},
			want:    5,
			wantErr: false,
		},
		{
			name: "two",
			args: args{
				source: map[float64]uint64{
					5: 5,
					7: 5,
				},
			},
			want:    6,
			wantErr: false,
		},
		{
			name: "odd",
			args: args{
				source: map[float64]uint64{
					3:       4,
					4:       6,
					457:     2,
					564:     1,
					565:     1,
					575:     1,
					3356.56: 1,
					35:      1,
					45:      6,
					46:      3,
					47:      2,
					56:      2,
					57:      2,
					64:      1,
					65:      2,
					5:       5,
					6:       2,
					7:       6,
					9:       1,
					23:      2,
					34:      2,
					74:      1,
					345:     3,
					436:     1,
					453:     1,
					456:     2,
				},
			},
			want:    45,
			wantErr: false,
		},
		{
			name: "even",
			args: args{
				source: map[float64]uint64{
					3:       4,
					4:       6,
					457:     2,
					564:     1,
					565:     1,
					575:     1,
					3356.56: 1,
					35:      1,
					45:      6,
					46:      3,
					47:      2,
					56:      2,
					57:      2,
					64:      1,
					65:      2,
					5:       5,
					6:       2,
					7:       7,
					9:       1,
					23:      2,
					34:      2,
					74:      1,
					345:     3,
					436:     1,
					453:     1,
					456:     2,
				},
			},
			want:    45,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MedianFromMap(tt.args.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("MedianFromMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MedianFromMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
