package handlers

import (
	"context"
	"net/url"

	"github.com/nicolas-vp/go-bpm-bank/bpm"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// PollOpening - обработчик service task с jobType "pollOpening" во всех процессах, где он встречается
// (job.ProcessName, job.ElementId). Вызывается асинхронно из очереди задач и при ошибке повторяется,
// поэтому должен быть идемпотентным (ключ - job.InstanceUid + job.ElementId)
func PollOpening(ctx context.Context, svc *services.Services, job bpm.Job, data *model.DataModel) error {
	var out coreResult
	if err := svc.Call(ctx, "GET", "/v1/account-openings/"+url.PathEscape(data.OperationID), "", nil, &out); err != nil {
		return err
	}
	return applyCore(data, out)
}
