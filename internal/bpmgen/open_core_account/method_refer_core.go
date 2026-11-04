package open_core_account

import (
	"context"

	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// ReferCore - метод задачи ReferCore «Передать разбирательство»
func ReferCore(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	data.Outcome = "REFERRED_CORE"
	return nil
}
