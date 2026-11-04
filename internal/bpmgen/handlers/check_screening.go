package handlers

import (
	"context"

	"github.com/nicolas-vp/go-bpm-bank/bpm"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// CheckScreening - обработчик service task с jobType "checkScreening" во всех процессах, где он встречается
// (job.ProcessName, job.ElementId). Вызывается асинхронно из очереди задач и при ошибке повторяется,
// поэтому должен быть идемпотентным (ключ - job.InstanceUid + job.ElementId)
func CheckScreening(ctx context.Context, svc *services.Services, job bpm.Job, data *model.DataModel) error {
	return check(ctx, svc, data, "screening")
}
