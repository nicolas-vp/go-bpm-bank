package collect_application

import (
	"context"
	"fmt"
	"time"

	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
)

// ValidateApplication - метод задачи ValidateApplication «Проверить границы примера»
func ValidateApplication(ctx context.Context, svc *services.Services, data *model.DataModel) error {
	birth, err := time.Parse("2006-01-02", data.BirthDate)
	if err != nil {
		return fmt.Errorf("неверная дата рождения: %w", err)
	}
	data.Outcome = "CONTINUE"
	if data.Citizenship != "RU" || birth.After(time.Now().AddDate(-18, 0, 0)) {
		data.Outcome = "REFERRED"
	}
	return nil
}
