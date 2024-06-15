package pages

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/go-arrower/arrower/contexts/admin/internal/application"
)

func formatAsDateOrTimeToday(t time.Time) string {
	now := time.Now()
	isToday := t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()

	createdAt := t.Format("2006.01.02 15:04")
	if isToday {
		createdAt = t.Format("15:04")
	}

	return createdAt
}

const (
	timeDay  = time.Hour * 24
	timeYear = timeDay * 365
)

func TimeAgo(t time.Time) string {
	if t.IsZero() {
		return "unclear"
	}

	switch timeSince := time.Duration(time.Since(t).Nanoseconds()); {
	case timeSince < time.Minute:
		return "now"
	case timeSince < 90*time.Minute:
		minutes := int(math.Round(float64(timeSince / time.Minute)))
		if minutes == 1 {
			return fmt.Sprintf("%d minute ago", minutes)
		}

		return fmt.Sprintf("%d minutes ago", minutes)
	case timeSince < timeDay:
		hours := int(math.Round(float64(timeSince / time.Hour)))
		if hours == 1 {
			return fmt.Sprintf("%d hour ago", hours)
		}

		return fmt.Sprintf("%d hours ago", hours)
	case timeSince < timeYear:
		days := int(math.Round(float64(timeSince / timeDay)))
		if days == 1 {
			return fmt.Sprintf("%d day ago", days)
		}

		return fmt.Sprintf("%d days ago", days)
	default:
		years := int(math.Round(float64(timeSince / timeYear)))
		if years == 1 {
			return fmt.Sprintf("%d year ago", years)
		}

		return fmt.Sprintf("%d years ago", years)
	}
}

func prettyJobPayloadAsFormattedJSON(p []byte) string {
	return prettyJSON(p)
}

func prettyJobPayloadDataAsFormattedJSON(payload application.JobPayload) string {
	b, _ := json.Marshal(payload.JobData)

	return prettyJSON(b)
}

func prettyJSON(str []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, str, "", "  "); err != nil {
		return ""
	}

	return prettyJSON.String()
}
