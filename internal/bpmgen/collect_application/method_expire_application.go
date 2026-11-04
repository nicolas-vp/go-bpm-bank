package collect_application

import (
	"context"

	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// ExpireApplication - метод задачи ExpireApplication «Закрыть незаполненное заявление»
func ExpireApplication(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	data.Outcome = "CLOSED_INCOMPLETE"
	return nil
}
