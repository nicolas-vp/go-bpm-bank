package handlers

import (
	"context"
	"fmt"

	"github.com/nicolas-vp/go-bpm-bank/bpm"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// ResolveCustomer - обработчик service task с jobType "resolveCustomer" во всех процессах, где он встречается
// (job.ProcessName, job.ElementId). Вызывается асинхронно из очереди задач и при ошибке повторяется,
// поэтому должен быть идемпотентным (ключ - job.InstanceUid + job.ElementId)
func ResolveCustomer(ctx context.Context, svc *services.Services, job bpm.Job, data *model.DataModel) error {
	var out map[string]string
	if err := svc.Call(ctx, "POST", "/v1/customers/resolve", data.ApplicationID+":customer:v1", map[string]string{"applicationId": data.ApplicationID, "applicantName": data.ApplicantName}, &out); err != nil {
		return err
	}
	data.CustomerID = out["customerId"]
	if data.CustomerID == "" {
		return fmt.Errorf("пустой customerId")
	}
	return nil
}
