package post_opening

import (
	"context"
	"fmt"

	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// Finish - метод задачи Finish «Завершить оформление»
func Finish(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	if !data.AccountOpened || data.TaxReportStatus != "CONFIRMED" {
		return fmt.Errorf("постобработка не завершена")
	}
	data.Outcome = "OPENED"
	return nil
}
