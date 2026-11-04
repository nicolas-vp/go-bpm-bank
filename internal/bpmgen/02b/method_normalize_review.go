package process_02b

import (
	"context"
	"fmt"

	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// NormalizeReview - метод задачи NormalizeReview «Зафиксировать решение»
func NormalizeReview(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	if data.ReviewDecision == "" {
		return fmt.Errorf("решение не заполнено")
	}
	if data.ScreeningStatus == "PEP" && data.ReviewDecision == "CONTINUE" && !data.Authorized {
		data.ReviewDecision = "REFER"
	}
	return nil
}
