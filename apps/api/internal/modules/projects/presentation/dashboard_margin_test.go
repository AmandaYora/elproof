package presentation

import "testing"

// Without finance.margin.view the dashboard keeps no trace of what the WO pays
// its vendors and venues; client payments and client-paid ones stay.
func TestHideSupplierCosts(t *testing.T) {
	weeks := []cashFlowWeekResponse{{WeekStart: "2027-01-04", ExpectedIn: 10, ExpectedOut: 7}}
	due := []supplierPaymentDueResponse{{ProjectID: 1, Remaining: 5}}
	resp := dashboardResponse{
		CashFlow: &weeks, SupplierPaymentsDue: &due,
		IncompletePayments: []dashboardPaymentResponse{{ID: 1, Payer: "WO"}, {ID: 2, Payer: "Client"}, {ID: 3}},
	}
	hideSupplierCosts(&resp)
	if (*resp.CashFlow)[0].ExpectedOut != 0 || (*resp.CashFlow)[0].ExpectedIn != 10 {
		t.Errorf("cash flow = %+v, want money in only", (*resp.CashFlow)[0])
	}
	if resp.SupplierPaymentsDue != nil {
		t.Error("supplier payments due still sent")
	}
	if len(resp.IncompletePayments) != 2 || resp.IncompletePayments[0].ID != 2 || resp.IncompletePayments[1].ID != 3 {
		t.Errorf("incomplete payments = %+v, want 2 and 3", resp.IncompletePayments)
	}
}
