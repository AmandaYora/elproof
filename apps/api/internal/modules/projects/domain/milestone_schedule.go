package domain

import "time"

// WIB is Asia/Jakarta (UTC+7, no daylight saving). ADR-0007: "today" is the
// Indonesian calendar day, whatever timezone the server itself runs in.
var WIB = time.FixedZone("WIB", 7*60*60)

// dayOf is the calendar day of t as written (DATE columns are read back as
// midnight of that day), as a UTC midnight, so days compare as dates.
func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// TodayWIB is the Indonesian calendar day of the instant asOf.
func TodayWIB(asOf time.Time) time.Time {
	return dayOf(asOf.In(WIB))
}

// MilestoneTarget is when a step of the Timeline that follows the event date is
// due: the event date minus its offset (H-14 is 14 days before; a negative
// offset is H+, after the event). A step that would fall before the booking date
// on a short-notice project is due on the booking date instead (it should have
// happened already, so it is the first thing to do). The pull is only made when
// the booking date is on or before the event date: for a project recorded after
// its event (a historical one) the steps keep their real offsets, so H-0 is
// always the event day itself.
func MilestoneTarget(eventDate, bookingDate time.Time, daysBeforeEvent int) time.Time {
	event := dayOf(eventDate)
	target := event.AddDate(0, 0, -daysBeforeEvent)
	booking := dayOf(bookingDate)
	if !booking.After(event) && target.Before(booking) {
		return booking
	}
	return target
}

// FollowsEventDate: the step's target is worked out from the event date (it came
// from the workspace's Jadwal bawaan and nobody set its date by hand) and it is
// still to be done, so it moves when the event or booking date moves.
func (m ProjectMilestone) FollowsEventDate() bool {
	return m.DaysBeforeEvent != nil && m.Status != MilestoneCompleted && m.Status != MilestoneCancelled
}

// ShiftMilestones returns the steps whose target changes for the new event and
// booking dates (with the new target set), leaving out the ones fixed by hand
// and the finished or cancelled ones.
func ShiftMilestones(milestones []ProjectMilestone, eventDate, bookingDate time.Time) []ProjectMilestone {
	var moved []ProjectMilestone
	for _, m := range milestones {
		if !m.FollowsEventDate() {
			continue
		}
		target := MilestoneTarget(eventDate, bookingDate, *m.DaysBeforeEvent)
		if dayOf(m.TargetDate).Equal(target) {
			continue
		}
		m.TargetDate = target
		moved = append(moved, m)
	}
	return moved
}

// StampCompletion keeps a step's completed date in line with its status after
// the status changed from `before` and the caller did not send a completed date
// itself: becoming Completed stamps today (when it has no date yet), leaving
// Completed clears the date. It reports whether the date changed.
func StampCompletion(before, after MilestoneStatus, completed **time.Time, today time.Time) bool {
	if before == after {
		return false
	}
	switch {
	case after == MilestoneCompleted && *completed == nil:
		d := dayOf(today)
		*completed = &d
		return true
	case after != MilestoneCompleted && *completed != nil:
		*completed = nil
		return true
	}
	return false
}
