package tools

import (
	"github.com/robfig/cron/v3"
	"time"
)

func GetNextTime(cronSpec string, t time.Time) (time.Time, error) {
	parseH := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	spec, err := parseH.Parse(cronSpec)
	if err != nil {
		return time.Time{}, err
	}
	return spec.Next(t), nil
}
