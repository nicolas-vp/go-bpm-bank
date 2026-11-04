package due_diligence

import (
	"context"

	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// Decline - метод задачи Decline «Зафиксировать правовое основание»
func Decline(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	if data.ReviewDecision == "DECLINE" && data.LegalBasisCode == "NK142_4_NO_TAX_DATA" && data.TaxStatus == "REVIEW" {
		data.Outcome = "DECLINED_WITH_BASIS"
	} else {
		data.Outcome = "REFERRED"
	}
	return nil
}
