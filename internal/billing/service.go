package billing

import "fmt"

// RateCard defines pricing in USD per 1000 invocations.
type RateCard struct {
	USDPerThousand float64
}

// Invoice contains basic usage-based billing information.
type Invoice struct {
	TenantID       string  `json:"tenant_id"`
	Invocations    int64   `json:"invocations"`
	USDPerThousand float64 `json:"usd_per_thousand"`
	AmountUSD      float64 `json:"amount_usd"`
}

func NewRateCard(usdPerThousand float64) (RateCard, error) {
	if usdPerThousand < 0 {
		return RateCard{}, fmt.Errorf("negative price is invalid")
	}
	return RateCard{USDPerThousand: usdPerThousand}, nil
}

func (r RateCard) Invoice(tenantID string, invocations int64) Invoice {
	amount := (float64(invocations) / 1000.0) * r.USDPerThousand
	return Invoice{TenantID: tenantID, Invocations: invocations, USDPerThousand: r.USDPerThousand, AmountUSD: amount}
}
