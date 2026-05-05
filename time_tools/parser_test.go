package timetools_test

import (
	"testing"
	"time"

	timetools "github.com/guionardo/go/time_tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var defaultLayouts = []string{
	time.DateTime,
	time.ANSIC,
	time.UnixDate,
	time.RubyDate,
	time.RFC822,
	time.RFC822Z,
	time.RFC850,
	time.RFC1123,
	time.RFC1123Z,
	time.RFC3339,
	time.RFC3339Nano,
	time.Kitchen,
	time.Stamp,
	time.StampMilli,
	time.StampMicro,
	time.StampNano,
	time.DateOnly,
	time.TimeOnly,
}

func TestParse(t *testing.T) {
	t.Parallel()

	timetools.SetLayouts(defaultLayouts)

	tests := []struct {
		name    string
		s       string
		want    time.Time
		wantErr bool
	}{
		{
			name:    "parse RFC3339",
			s:       "2024-03-15T10:20:30Z",
			want:    time.Date(2024, time.March, 15, 10, 20, 30, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "parse DateOnly",
			s:       "2024-03-15",
			want:    time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "parse Kitchen",
			s:       "3:04PM",
			want:    time.Date(0, time.January, 1, 15, 4, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "invalid value",
			s:       "not-a-date",
			want:    time.Time{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := timetools.Parse(tt.s)
			if tt.wantErr {
				require.Errorf(t, gotErr, "expected error on parsing %s", tt.s)
			} else {
				require.NoError(t, gotErr, "unexpected error on parsing %s", tt.s)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
