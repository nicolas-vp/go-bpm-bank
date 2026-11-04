package bankapi

import (
	"context"
	"github.com/nicolas-vp/go-bpm-bank/bpm"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// Queues ограничивает область записи параллельных проверок.
// Типизированный адаптер сериализует всю модель, включая старые/нулевые поля соседних веток.
// Перед join убираем чужие результаты: их вернёт только ветка-владелец.
func Queues(svc *services.Services) []bpm.Queue {
	queues := bpmgen.Queues(svc)
	owners := map[string]string{"checkDocument": "document", "checkScreening": "screening", "checkTax": "tax"}
	for i := range queues {
		owner, ok := owners[queues[i].TypeID]
		if !ok {
			continue
		}
		handler := queues[i].Handler
		queues[i].Handler = func(ctx context.Context, job bpm.Job, v map[string]any) (map[string]any, error) {
			out, err := handler(ctx, job, v)
			if err != nil {
				return nil, err
			}
			for _, prefix := range []string{"document", "screening", "tax"} {
				if prefix != owner {
					delete(out, prefix+"Status")
					delete(out, prefix+"Evidence")
				}
			}
			return out, nil
		}
	}
	return queues
}
