package conclude_contract

import (
	"context"
	"fmt"

	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// RecordConsent - метод задачи RecordConsent «Зафиксировать исход»
func RecordConsent(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	if !data.ContractAccepted {
		data.Outcome = "WITHDRAWN"
		return nil
	}
	if data.SignedDocumentRef == "" {
		return fmt.Errorf("нет ссылки на учебное подтверждение подписания")
	}
	return nil
}
