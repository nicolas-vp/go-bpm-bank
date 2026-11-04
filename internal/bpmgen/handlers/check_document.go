package handlers

import (
	"context"

	"github.com/nicolas-vp/go-bpm-bank/bpm"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// CheckDocument - обработчик service task с jobType "checkDocument" во всех процессах, где он встречается
// (job.ProcessName, job.ElementId). Вызывается асинхронно из очереди задач и при ошибке повторяется,
// поэтому должен быть идемпотентным (ключ - job.InstanceUid + job.ElementId)
func CheckDocument(ctx context.Context, svc *services.Services, job bpm.Job, data *model.DataModel) error {
	return check(ctx, svc, data, "document")
}
