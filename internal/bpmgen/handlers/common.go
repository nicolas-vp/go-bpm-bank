package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/model"
	"github.com/nicolas-vp/go-bpm-bank/internal/bpmgen/services"
	"net/url"
)

func check(ctx context.Context, s *services.Services, d *model.DataModel, kind string) error {
	in := map[string]any{"applicationId": d.ApplicationID, "scenario": d.Scenario, "documentsComplete": d.DocumentsComplete, "taxResidency": d.TaxResidency}
	var out map[string]string
	if err := s.Call(ctx, "POST", "/v1/checks/"+kind, "", in, &out); err != nil {
		return err
	}
	if out["status"] == "" || out["requestId"] == "" {
		return fmt.Errorf("неполный ответ проверки %s", kind)
	}
	switch kind {
	case "document":
		d.DocumentStatus = out["status"]
		d.DocumentEvidence = out
	case "screening":
		d.ScreeningStatus = out["status"]
		d.ScreeningEvidence = out
	case "tax":
		d.TaxStatus = out["status"]
		d.TaxEvidence = out
	}
	return nil
}

type coreResult struct {
	OperationID   string `json:"operationId"`
	AccountID     string `json:"accountId"`
	AccountNumber string `json:"accountNumber"`
	Status        string `json:"status"`
	RegisteredAt  string `json:"registeredAt"`
}

func applyCore(d *model.DataModel, v coreResult) error {
	if v.OperationID == "" {
		return fmt.Errorf("АБС не вернула operationId")
	}
	if v.Status != "PENDING" && v.Status != "OPENED" && v.Status != "REJECTED" {
		return fmt.Errorf("неизвестный статус АБС %q", v.Status)
	}
	d.OperationID = v.OperationID
	d.AccountID = v.AccountID
	d.AccountNumber = v.AccountNumber
	d.CoreStatus = v.Status
	d.RegisteredAt = v.RegisteredAt
	return nil
}

func submit(ctx context.Context, s *services.Services, d *model.DataModel) error {
	key := d.ApplicationID + ":opening:v1"
	var out coreResult
	// Сверка закрывает потерю ответа и рестарт после успешного внешнего действия.
	if err := s.Call(ctx, "GET", "/v1/account-openings?requestId="+url.QueryEscape(key), "", nil, &out); err == nil {
		return applyCore(d, out)
	} else {
		var response *services.HTTPError
		if !errors.As(err, &response) || response.Status != 404 {
			return err
		}
	}
	in := map[string]string{"applicationId": d.ApplicationID, "customerId": d.CustomerID, "contractId": d.ContractID, "scenario": d.Scenario}
	if err := s.Call(ctx, "POST", "/v1/account-openings", key, in, &out); err != nil {
		return err
	}
	return applyCore(d, out)
}

func effect(ctx context.Context, s *services.Services, d *model.DataModel, kind string) error {
	in := map[string]string{"applicationId": d.ApplicationID, "accountId": d.AccountID, "scenario": d.Scenario}
	var out map[string]string
	if err := s.Call(ctx, "POST", "/v1/effects/"+kind, d.ApplicationID+":"+kind+":v1", in, &out); err != nil {
		return err
	}
	if out["status"] != "CONFIRMED" {
		return fmt.Errorf("эффект %s не подтверждён", kind)
	}
	switch kind {
	case "contract":
		d.ContractID = out["id"]
	case "archive":
		d.ArchiveStatus = out["status"]
	case "tax-report":
		d.TaxReportStatus = out["status"]
	case "notification":
		d.NotificationStatus = out["status"]
	}
	return nil
}
