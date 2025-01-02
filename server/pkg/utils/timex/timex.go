package timex

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

const DefaultDateTimeFormat = "2006-01-02 15:04:05"
const DefaultDateFormat = "2006-01-02"

// DefaultFormat 使用默认格式进行格式化: 2006-01-02 15:04:05
func DefaultFormat(time time.Time) string {
	return time.Format(DefaultDateTimeFormat)
}

// DefaultFormatDate 使用默认格式进行格式化: 2006-01-02
func DefaultFormatDate(time time.Time) string {
	return time.Format(DefaultDateFormat)
}

// TimeNo 获取当前时间编号，格式为20060102150405
func TimeNo() string {
	return time.Now().Format("20060102150405")
}

func NewNullTime(t time.Time) NullTime {
	return NullTime{
		NullTime: sql.NullTime{
			Time:  t,
			Valid: !t.IsZero(),
		},
	}
}

type NullTime struct {
	sql.NullTime
}

func (nt *NullTime) UnmarshalJSON(bytes []byte) error {
	if len(bytes) == 0 {
		nt.NullTime = sql.NullTime{}
		return nil
	}
	var t time.Time
	if err := json.Unmarshal(bytes, &t); err != nil {
		return err
	}
	if t.IsZero() {
		nt.NullTime = sql.NullTime{}
		return nil
	}
	nt.NullTime = sql.NullTime{
		Valid: true,
		Time:  t,
	}
	return nil
}

func (nt *NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid || nt.Time.IsZero() {
		return json.Marshal(nil)
	}
	return json.Marshal(nt.Time)
}

func SleepWithContext(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}
