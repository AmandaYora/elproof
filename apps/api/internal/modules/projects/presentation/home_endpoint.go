package presentation

import (
	"net/http"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

// Response shapes of GET /api/v1/projects/me/home (E5). A section a client's
// portal role may not see is absent (pointer + omitempty), never a null
// placeholder; nothing here carries a staff id, the sales PIC, the venue's
// commercial terms, the booking date, vendor money or internal notes.

type homeProjectResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	BrideName   string `json:"brideName"`
	GroomName   string `json:"groomName"`
	EventDate   string `json:"eventDate"`
	Venue       string `json:"venue"`
	PackageName string `json:"packageName"`
	Status      string `json:"status"`
	PICName     string `json:"picName"`
	PICPhone    string `json:"picPhone"`
}

type homeCountdownResponse struct {
	DaysToEvent int `json:"daysToEvent"`
}

type homeProgressResponse struct {
	OverallPercent int    `json:"overallPercent"`
	Condition      string `json:"condition"`
}

type homeNextTermResponse struct {
	ID        int64  `json:"id"`
	Label     string `json:"label"`
	Amount    int64  `json:"amount"`
	DueDate   string `json:"dueDate"`
	Status    string `json:"status"`
	Remaining int64  `json:"remaining"`
}

type homeFinanceResponse struct {
	ContractValue int64                 `json:"contractValue"`
	Paid          int64                 `json:"paid"`
	Remaining     int64                 `json:"remaining"`
	NextTerm      *homeNextTermResponse `json:"nextTerm"`
	OverdueCount  int                   `json:"overdueCount"`
}

type homePayToResponse struct {
	BankName      string `json:"bankName"`
	AccountNumber string `json:"accountNumber"`
	AccountHolder string `json:"accountHolder"`
}

type homeContactResponse struct {
	WhatsApp string `json:"whatsapp"`
}

type homeActionResponse struct {
	Kind    string  `json:"kind"`
	Title   string  `json:"title"`
	Detail  string  `json:"detail"`
	DueDate *string `json:"dueDate,omitempty"`
	Amount  *int64  `json:"amount,omitempty"`
	Target  string  `json:"target"`
}

type homeProofResponse struct {
	EvidenceID int64  `json:"evidenceId"`
	Type       string `json:"type"`
	FileName   string `json:"fileName"`
}

type homeUpdateResponse struct {
	Kind       string             `json:"kind"`
	Title      string             `json:"title"`
	At         string             `json:"at"`
	Proof      *homeProofResponse `json:"proof"`
	VerifiedBy string             `json:"verifiedBy,omitempty"`
}

type homePICResponse struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type clientHomeResponse struct {
	Project        homeProjectResponse   `json:"project"`
	Countdown      homeCountdownResponse `json:"countdown"`
	Progress       *homeProgressResponse `json:"progress,omitempty"`
	Finance        *homeFinanceResponse  `json:"finance,omitempty"`
	PayTo          *homePayToResponse    `json:"payTo,omitempty"`
	Contact        *homeContactResponse  `json:"contact,omitempty"`
	Actions        []homeActionResponse  `json:"actions"`
	OpenIssueCount *int                  `json:"openIssueCount,omitempty"`
	RecentUpdates  []homeUpdateResponse  `json:"recentUpdates"`
	PIC            homePICResponse       `json:"pic"`
}

func toClientHomeResponse(h *application.ClientHome) clientHomeResponse {
	out := clientHomeResponse{
		Project: homeProjectResponse{
			ID: h.Project.ID, Name: h.Project.Name, BrideName: h.Project.BrideName, GroomName: h.Project.GroomName,
			EventDate: h.Project.EventDate.Format(dateLayout), Venue: h.Project.Venue, PackageName: h.Project.PackageName,
			Status: h.Project.Status, PICName: h.Project.PICName, PICPhone: h.Project.PICPhone,
		},
		Countdown:      homeCountdownResponse{DaysToEvent: h.DaysToEvent},
		OpenIssueCount: h.OpenIssueCount,
		Actions:        make([]homeActionResponse, 0, len(h.Actions)),
		RecentUpdates:  make([]homeUpdateResponse, 0, len(h.RecentUpdates)),
		PIC:            homePICResponse{Name: h.PICName, Phone: h.PICPhone},
	}
	if h.Progress != nil {
		out.Progress = &homeProgressResponse{OverallPercent: h.Progress.OverallPercent, Condition: h.Progress.Condition}
	}
	if h.Finance != nil {
		f := &homeFinanceResponse{
			ContractValue: h.Finance.ContractValue, Paid: h.Finance.Paid, Remaining: h.Finance.Remaining, OverdueCount: h.Finance.OverdueCount,
		}
		if t := h.Finance.NextTerm; t != nil {
			f.NextTerm = &homeNextTermResponse{
				ID: t.ID, Label: t.Label, Amount: t.Amount, DueDate: t.DueDate.Format(dateLayout), Status: t.Status, Remaining: t.Remaining,
			}
		}
		out.Finance = f
	}
	if h.PayTo != nil {
		out.PayTo = &homePayToResponse{BankName: h.PayTo.BankName, AccountNumber: h.PayTo.AccountNumber, AccountHolder: h.PayTo.AccountHolder}
	}
	if h.WhatsApp != "" {
		out.Contact = &homeContactResponse{WhatsApp: h.WhatsApp}
	}
	for _, a := range h.Actions {
		ar := homeActionResponse{Kind: string(a.Kind), Title: a.Title, Detail: a.Detail, Amount: a.Amount, Target: a.Target}
		if a.DueDate != nil {
			d := a.DueDate.Format(dateLayout)
			ar.DueDate = &d
		}
		out.Actions = append(out.Actions, ar)
	}
	for _, u := range h.RecentUpdates {
		ur := homeUpdateResponse{Kind: string(u.Kind), Title: u.Title, At: u.At.Format(time.RFC3339), VerifiedBy: u.VerifiedBy}
		if u.Proof != nil {
			ur.Proof = &homeProofResponse{EvidenceID: u.Proof.EvidenceID, Type: string(u.Proof.Type), FileName: u.Proof.FileName}
		}
		out.RecentUpdates = append(out.RecentUpdates, ur)
	}
	return out
}

// meHome is GET /projects/me/home: the whole Client Portal "Beranda" in one
// call, limited to the sections the client's portal role may see. Same project
// resolution and the same 403/404 answers as GET /projects/me.
func (h *Handler) meHome(w http.ResponseWriter, r *http.Request) {
	tenantID, projectID, ok := h.resolveOwnClientProject(w, r)
	if !ok {
		return
	}
	if h.clientHome == nil {
		response.Error(w, http.StatusForbidden, "Akses client belum dikonfigurasi", nil)
		return
	}
	home, err := h.clientHome.Get(r.Context(), tenantID, projectID, middleware.GrantsFrom(r.Context()).Can, nowWIB())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toClientHomeResponse(home))
}
