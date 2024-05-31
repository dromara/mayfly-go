package model

import (
	"database/sql/driver"
	"fmt"
	"mayfly-go/pkg/utils/timex"
	"strings"
	"time"
)

type JsonTime struct {
	time.Time
}

func NewJsonTime(t time.Time) JsonTime {
	return JsonTime{
		Time: t,
	}
}

func NowJsonTime() JsonTime {
	return JsonTime{
		Time: time.Now(),
	}
}

func (j JsonTime) MarshalJSON() ([]byte, error) {
	var stamp = fmt.Sprintf("\"%s\"", j.Format(timex.DefaultDateTimeFormat))
	return []byte(stamp), nil
}

func (j *JsonTime) UnmarshalJSON(b []byte) error {
	s := strings.ReplaceAll(string(b), "\"", "")
	// t, err := time.Parse(timex.DefaultDateTimeFormat, s)
	t, err := time.ParseInLocation(timex.DefaultDateTimeFormat, s, time.Local)
	if err != nil {
		return err
	}
	*j = NewJsonTime(t)
	return nil
}

func (j JsonTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if j.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return j.Time, nil
}

func (j *JsonTime) Scan(v interface{}) error {
	value, ok := v.(time.Time)
	if ok {
		*j = JsonTime{Time: value}
		return nil
	}
	return fmt.Errorf("can not convert %v to timestamp", v)
}
