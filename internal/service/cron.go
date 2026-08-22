package service

import (
	"time"

	"github.com/robfig/cron/v3"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// NextCronRun returns the next activation time for a cron expression in the
// given IANA timezone (defaulting to UTC on invalid timezone).
func NextCronRun(expr, timezone string, now time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	sched, err := cronParser.Parse(expr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(now.In(loc)), nil
}
