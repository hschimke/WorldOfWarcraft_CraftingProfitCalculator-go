package blizz_oath

import (
	"testing"
	"time"
)

func TestAccessToken_CheckExpired(t *testing.T) {
	type fields struct {
		Access_token string
		Token_type   string
		Expires_in   uint64
		Scope        string
		Fetched      time.Time
	}
	tests := []struct {
		name        string
		fields      fields
		wantExpired bool
	}{
		{name: "expired",
			fields: fields{
				Expires_in: 1000,
				Fetched:    time.Now().Add(time.Hour * -24),
			},
			wantExpired: true,
		},
		{name: "not expired",
			fields: fields{
				Expires_in: 1000,
				Fetched:    time.Now(),
			},
			wantExpired: true,
		},
		{name: "not expired exact time",
			fields: fields{
				Expires_in: 1000,
				Fetched:    time.Now().Add(time.Duration(1000)),
			},
			wantExpired: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at := &AccessToken{
				Access_token: tt.fields.Access_token,
				Token_type:   tt.fields.Token_type,
				Expires_in:   tt.fields.Expires_in,
				Scope:        tt.fields.Scope,
				Fetched:      tt.fields.Fetched,
			}
			if gotExpired := at.CheckExpired(); gotExpired != tt.wantExpired {
				t.Errorf("AccessToken.CheckExpired() = %v, want %v", gotExpired, tt.wantExpired)
			}
		})
	}
}
