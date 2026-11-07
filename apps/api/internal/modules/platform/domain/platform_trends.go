package domain

import (
	"strconv"
	"time"
)

// TrendPeriod chooses the bucket size of the dashboard charts.
type TrendPeriod string

const (
	// TrendMonth: one bar per day, from the 1st through today.
	TrendMonth TrendPeriod = "month"
	// TrendYear: one bar per month, from January through this month.
	TrendYear TrendPeriod = "year"
)

// wibZone is how the platform counts days: the people reading the charts live
// in Western Indonesia Time, whatever the server's own zone.
var wibZone = time.FixedZone("WIB", 7*60*60)

var monthLabels = [12]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

// TrendPoint is one bar.
type TrendPoint struct {
	Key   string
	Label string
	Value int64
}

// Trends are the two dashboard charts. Bars only run up to today (or this
// month): empty bars for days that have not happened would mislead.
type Trends struct {
	Period     TrendPeriod
	Revenue    []TrendPoint
	NewTenants []TrendPoint
}

// ComputeTrends buckets paid revenue and newly joined tenants. paid must be
// real (gateway-paid) transactions only; joined is every tenant's join time.
// Anything outside the current month (or year, for TrendYear) is ignored.
func ComputeTrends(period TrendPeriod, now time.Time, paid []PaidPayment, joined []time.Time) Trends {
	if period != TrendYear {
		period = TrendMonth
	}
	now = now.In(wibZone)
	t := Trends{Period: period, Revenue: emptyTrend(period, now), NewTenants: emptyTrend(period, now)}
	for _, p := range paid {
		addToTrend(t.Revenue, period, now, p.PaidAt, p.Amount)
	}
	for _, j := range joined {
		addToTrend(t.NewTenants, period, now, j, 1)
	}
	return t
}

func emptyTrend(period TrendPeriod, now time.Time) []TrendPoint {
	if period == TrendMonth {
		points := make([]TrendPoint, now.Day())
		for i := range points {
			points[i] = TrendPoint{Key: "d" + strconv.Itoa(i+1), Label: strconv.Itoa(i + 1)}
		}
		return points
	}
	points := make([]TrendPoint, int(now.Month()))
	for i := range points {
		points[i] = TrendPoint{Key: "m" + strconv.Itoa(i+1), Label: monthLabels[i]}
	}
	return points
}

func addToTrend(points []TrendPoint, period TrendPeriod, now, at time.Time, amount int64) {
	at = at.In(wibZone)
	if at.Year() != now.Year() || at.After(now) {
		return
	}
	switch period {
	case TrendMonth:
		if at.Month() == now.Month() {
			points[at.Day()-1].Value += amount
		}
	case TrendYear:
		points[int(at.Month())-1].Value += amount
	}
}
