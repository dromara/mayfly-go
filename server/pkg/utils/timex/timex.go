package timex

import "time"

const DefaultDateTimeFormat = "2006-01-02 15:04:05"

func DefaultFormat(time time.Time) string {
	return time.Format(DefaultDateTimeFormat)
}
