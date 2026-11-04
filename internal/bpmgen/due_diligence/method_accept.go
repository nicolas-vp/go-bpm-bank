package due_diligence

import (
	"context"

	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// Accept - метод задачи Accept «Продолжить оформление»
func Accept(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	data.Outcome = "CONTINUE"
	return nil
}
