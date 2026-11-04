package open_core_account

import (
	"context"
	"fmt"

	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// ConfirmOpened - метод задачи ConfirmOpened «Подтвердить регистрацию»
func ConfirmOpened(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	if data.AccountID == "" || data.RegisteredAt == "" || data.CoreStatus != "OPENED" {
		return fmt.Errorf("нет подтверждения регистрации счёта")
	}
	data.AccountOpened = true
	data.Outcome = "OPENED"
	return nil
}
