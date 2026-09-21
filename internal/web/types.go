package web

import (
	"jevai/internal/account"
	"jevai/internal/billing"
	"jevai/internal/ledger"
	"jevai/internal/store"
)

// DashData is everything the dashboard renders.
type DashData struct {
	User    *account.User
	Recent  []store.AuditMeta
	Usage   ledger.Usage
	Gate    billing.Status
	IsAdmin bool
	BaseURL string           // e.g. https://jevai.glicc.id (for building full invite links)
	Invites []account.Invite // admin only
}
