package timex

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNullTime_UnmarshalJSON(t *testing.T) {
	zero := time.Time{}
	now := time.Now()
	bytesNow, err := now.MarshalJSON()
	require.NoError(t, err)
	tests := []struct {
		name    string
		want    NullTime
		bytes   []byte
		wantErr error
	}{
		{
			name:  "zero",
			want:  NewNullTime(zero),
			bytes: nil,
		},
		{
			name:  "now",
			want:  NewNullTime(now),
			bytes: bytesNow,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &NullTime{}
			err := got.UnmarshalJSON(tt.bytes)
			require.ErrorIs(t, err, tt.wantErr)
			if err != nil {
				return
			}
			require.Equal(t, tt.want.Valid, got.Valid)
			require.True(t, got.Time.Equal(tt.want.Time))
		})
	}
}

func TestNullTime_MarshalJSON(t *testing.T) {
	zero := time.Time{}
	now := time.Now()
	bytes, err := now.MarshalJSON()
	require.NoError(t, err)
	tests := []struct {
		name     string
		nullTime NullTime
		want     []byte
		wantErr  error
	}{
		{
			name:     "zero",
			nullTime: NewNullTime(zero),
			want:     []byte("null"),
		},
		{
			name:     "now",
			nullTime: NewNullTime(now),
			want:     bytes,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.nullTime.MarshalJSON()
			require.ErrorIs(t, err, tt.wantErr)
			if err != nil {
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}
