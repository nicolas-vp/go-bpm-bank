package handlers

import (
	"context"

	"github.com/nicolas-vp/go-bpm-bank/bpm"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// PrepareContract - обработчик service task с jobType "prepareContract" во всех процессах, где он встречается
// (job.ProcessName, job.ElementId). Вызывается асинхронно из очереди задач и при ошибке повторяется,
// поэтому должен быть идемпотентным (ключ - job.InstanceUid + job.ElementId)
func PrepareContract(ctx context.Context, svc *services.Services, job bpm.Job, data *model.DataModel) error {
	return effect(ctx, svc, data, "contract")
}
