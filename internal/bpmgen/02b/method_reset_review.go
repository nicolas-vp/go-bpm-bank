package process_02b

import (
	"context"

	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// ResetReview - метод задачи ResetReview «Начать новое рассмотрение»
func ResetReview(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	// Старые разрешения не должны подтверждать новое рассмотрение после уточнения анкеты.
	data.ReviewedBy = ""
	data.AuthorizedBy = ""
	data.Authorized = false
	data.ReviewDecision = ""
	data.LegalBasisCode = ""
	data.Reason = ""
	return nil
}
