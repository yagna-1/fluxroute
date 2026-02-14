package unit

import (
	"testing"

	"github.com/your-org/fluxroute/internal/billing"
)

func TestRateCardInvoice(t *testing.T) {
	rate, err := billing.NewRateCard(2.5)
	if err != nil {
		t.Fatalf("new rate card failed: %v", err)
	}

	invoice := rate.Invoice("tenant-a", 2500)
	if invoice.TenantID != "tenant-a" {
		t.Fatalf("unexpected tenant id: %q", invoice.TenantID)
	}
	if invoice.AmountUSD != 6.25 {
		t.Fatalf("unexpected amount: %f", invoice.AmountUSD)
	}
}

func TestRateCardRejectsNegativePrice(t *testing.T) {
	if _, err := billing.NewRateCard(-1.0); err == nil {
		t.Fatal("expected error for negative rate")
	}
}
