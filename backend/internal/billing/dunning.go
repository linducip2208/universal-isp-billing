// Dunning: overdue state machine. Pure decision functions consumed by the
// automation engine and scheduler — no network I/O here.
//
// Timeline per invoice:
//
//	due date -> [grace] -> reminder -> warning -> suspend -> terminate
//
// Payment at any point -> reactivation decision.
package billing

import "time"

type DunningStage string

const (
	StageCurrent   DunningStage = "current"
	StageGrace     DunningStage = "grace"
	StageReminder  DunningStage = "reminder"
	StageWarning   DunningStage = "warning"
	StageSuspend   DunningStage = "suspend"
	StageTerminate DunningStage = "terminate"
)

type DunningPolicy struct {
	ReminderAfter  time.Duration // after due
	WarningAfter   time.Duration
	SuspendAfter   time.Duration // after grace expiry; 0 = at grace expiry
	TerminateAfter time.Duration
}

func DefaultPolicy() DunningPolicy {
	return DunningPolicy{
		ReminderAfter:  24 * time.Hour,
		WarningAfter:   3 * 24 * time.Hour,
		TerminateAfter: 30 * 24 * time.Hour,
	}
}

// StageOf returns where an unpaid invoice sits right now.
// Suspend/terminate are measured from grace expiry; reminders from due date.
func StageOf(inv *Invoice, now time.Time, pol DunningPolicy) DunningStage {
	if inv.Status == InvoicePaid || inv.Status == InvoiceVoid {
		return StageCurrent
	}
	if !now.After(inv.DueAt) {
		return StageCurrent
	}
	if inv.GraceUntil != nil && !now.After(*inv.GraceUntil) {
		return StageGrace
	}
	graceEnd := inv.DueAt
	if inv.GraceUntil != nil {
		graceEnd = *inv.GraceUntil
	}
	sinceGrace := now.Sub(graceEnd)
	switch {
	case sinceGrace >= pol.TerminateAfter:
		return StageTerminate
	case sinceGrace >= pol.SuspendAfter:
		return StageSuspend
	case now.Sub(inv.DueAt) >= pol.WarningAfter:
		return StageWarning
	case now.Sub(inv.DueAt) >= pol.ReminderAfter:
		return StageReminder
	default:
		return StageGrace
	}
}

// AgingBucket for receivables reports.
func AgingBucket(inv *Invoice, now time.Time) string {
	if inv.Status == InvoicePaid || inv.Status == InvoiceVoid {
		return "paid"
	}
	d := int(now.Sub(inv.DueAt).Hours() / 24)
	switch {
	case d <= 0:
		return "current"
	case d <= 7:
		return "1-7"
	case d <= 30:
		return "8-30"
	default:
		return "30+"
	}
}
