package timex

import (
	"database/sql"
	"encoding/json"
	"time"
)

func DefaultFormat(time time.Time) string {
	return time.Format("2006-01-02 15:04:05")
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
