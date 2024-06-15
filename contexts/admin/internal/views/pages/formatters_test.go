package pages_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/contexts/admin/internal/views/pages"
)

func TestTimeAgo(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		time     time.Time
		expected string
	}{
		"empty":           {time.Time{}, "unclear"},
		"lt a minute ago": {time.Now().Add(-30 * time.Second), "now"},
		"1m ago":          {time.Now().Add(-time.Minute), "1 minute ago"},
		"2m ago":          {time.Now().Add(-2 * time.Minute), "2 minutes ago"},
		"~1h ago":         {time.Now().Add(-90 * time.Minute), "1 hour ago"},
		"2h ago":          {time.Now().Add(-2 * time.Hour), "2 hours ago"},
		"1 day ago":       {time.Now().Add(-24 * time.Hour), "1 day ago"},
		"2 days ago":      {time.Now().Add(-24 * 2 * time.Hour), "2 days ago"},
		"1 year ago":      {time.Now().Add(-365 * 24 * time.Hour), "1 year ago"},
		"2 years ago":     {time.Now().Add(-365 * 2 * 24 * time.Hour), "2 years ago"},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, pages.TimeAgo(tt.time))
		})
	}
}
