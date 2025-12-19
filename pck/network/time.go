package network

import "time"

type TimeUnit string

const (
	Year        TimeUnit = "year"
	Month       TimeUnit = "month"
	Day         TimeUnit = "day"
	Hour        TimeUnit = "hour"
	Minute      TimeUnit = "minute"
	Second      TimeUnit = "second"
	Millisecond TimeUnit = "millisecond"
	Microsecond TimeUnit = "microsecond"
)

func (m TimeUnit) ToDuration(n int) time.Duration {
	switch m {
	case Year:
		return time.Hour * 24 * 365 * time.Duration(n)
	case Month:
		return time.Hour * 24 * 30 * time.Duration(n)
	case Day:
		return time.Hour * 24 * time.Duration(n)
	case Hour:
		return time.Hour * time.Duration(n)
	case Minute:
		return time.Minute * time.Duration(n)
	case Second:
		return time.Second * time.Duration(n)
	case Millisecond:
		return time.Millisecond * time.Duration(n)
	case Microsecond:
		return time.Microsecond * time.Duration(n)
	default:
		return 0
	}
}
