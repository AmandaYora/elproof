// Package audit wires the audit-trail module: it owns the audit_log table and
// implements middleware.AuditRecorder, which cmd/server plugs into the
// authenticated middleware chain. No other module reads or writes audit_log.
package audit

import (
	"database/sql"
	"net/http"

	"elproof/internal/modules/audit/application"
	"elproof/internal/modules/audit/infrastructure"
	"elproof/internal/modules/audit/presentation"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
)

type Module struct {
	service *application.Service
	handler *presentation.Handler
}

func NewModule(db *sql.DB) *Module {
	service := application.NewService(infrastructure.NewMySQLAuditRepository(db))
	return &Module{service: service, handler: presentation.NewHandler(service)}
}

// Recorder is what the audit middleware writes to.
func (m *Module) Recorder() middleware.AuditRecorder {
	return m.service
}

func (m *Module) RegisterRoutes(mux *http.ServeMux, authed func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/audit-log", authed(httpx.Method(http.MethodGet, m.handler.List)))
}
