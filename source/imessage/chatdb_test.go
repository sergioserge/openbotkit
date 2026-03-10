package imessage

import (
	"testing"
	"time"
)

func TestAppleNanosToTime(t *testing.T) {
	tests := []struct {
		name string
		ns   int64
		want time.Time
	}{
		{
			name: "zero",
			ns:   0,
			want: time.Time{},
		},
		{
			name: "apple epoch start (2001-01-01)",
			ns:   0 * 1_000_000_000,
			want: time.Time{},
		},
		{
			name: "known date 2024-01-15 12:00:00 UTC",
			// 2024-01-15 12:00:00 UTC = Unix 1705320000
			// Apple offset = 978307200
			// Apple seconds = 1705320000 - 978307200 = 727012800
			// Apple nanos = 727012800 * 1e9
			ns:   727012800_000_000_000,
			want: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "2020-06-15 10:30:00 UTC",
			// Unix: 1592217000
			// Apple seconds: 1592217000 - 978307200 = 613909800
			ns:   613909800_000_000_000,
			want: time.Date(2020, 6, 15, 10, 30, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appleNanosToTime(tt.ns)
			if !got.Equal(tt.want) {
				t.Errorf("appleNanosToTime(%d) = %v, want %v", tt.ns, got, tt.want)
			}
		})
	}
}

func TestAppleNanosToTimeZero(t *testing.T) {
	got := appleNanosToTime(0)
	if !got.IsZero() {
		t.Errorf("expected zero time, got %v", got)
	}
}
