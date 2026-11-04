// Package bankapi — прикладные ограничения учебного банка поверх стандартного API go-bpm.
package bankapi

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/nicolas-vp/go-bpm-bank/engine"
	"strings"
	"time"
)

var demoGroups = map[string][]string{"operator": {"operators"}, "aml": {"compliance"}, "chief": {"supervisors"}}

type Tasks struct{ *engine.UserTaskService }

func deny(message string) error { return fmt.Errorf("%w: %s", engine.ErrNotCandidate, message) }

// Демо-пользователь выбирается явно. Это не промышленная аутентификация.
// Группы из запроса клиента игнорируются, чтобы им нельзя было приписать себе supervisors.
func (t *Tasks) Claim(ctx context.Context, id uuid.UUID, user string, _ []string) error {
	groups, ok := demoGroups[user]
	if !ok {
		return deny("неизвестный учебный пользователь")
	}
	return t.UserTaskService.Claim(ctx, id, user, groups)
}

var fields = map[string]string{
	"IdentifyApplicant": "applicantName birthDate citizenship passportRef address purpose taxResidency identityVerified documentsComplete",
	"ReviewClient":      "reviewDecision legalBasisCode reason",
	"AuthorizeClient":   "reviewDecision legalBasisCode reason authorized",
	"ConfirmContract":   "contractAccepted signedDocumentRef",
}

func Validate(task *engine.UserTask, user string, v map[string]any) error {
	if _, ok := demoGroups[user]; !ok {
		return deny("неизвестный учебный пользователь")
	}
	allowed, ok := fields[task.ElementId]
	if !ok {
		return deny("задача не принадлежит учебному сценарию")
	}
	for k, val := range v {
		if !strings.Contains(" "+allowed+" ", " "+k+" ") {
			return deny("изменение служебного поля запрещено: " + k)
		}
		if k == "identityVerified" || k == "documentsComplete" || k == "authorized" || k == "contractAccepted" {
			if _, ok := val.(bool); !ok {
				return deny("поле должно быть boolean: " + k)
			}
		} else {
			if _, ok := val.(string); !ok {
				return deny("поле должно быть string: " + k)
			}
		}
	}
	text := func(k string) string { s, _ := v[k].(string); return strings.TrimSpace(s) }
	flag := func(k string) (bool, bool) { b, ok := v[k].(bool); return b, ok }
	switch task.ElementId {
	case "IdentifyApplicant":
		for _, k := range []string{"applicantName", "birthDate", "citizenship", "passportRef", "address", "purpose"} {
			if text(k) == "" {
				return deny("обязательно поле " + k)
			}
		}
		date, err := time.Parse("2006-01-02", text("birthDate"))
		if err != nil || date.After(time.Now()) {
			return deny("неверная дата рождения")
		}
		if b, ok := flag("identityVerified"); !ok || !b {
			return deny("сначала подтвердите личную идентификацию")
		}
		if _, ok := flag("documentsComplete"); !ok {
			return deny("укажите documentsComplete")
		}
	case "ReviewClient", "AuthorizeClient":
		decision := text("reviewDecision")
		if decision != "CONTINUE" && decision != "REQUEST_INFO" && decision != "DECLINE" && decision != "REFER" {
			return deny("недопустимое решение")
		}
		if text("reason") == "" {
			return deny("нужно обоснование решения")
		}
		if decision == "DECLINE" && (text("legalBasisCode") != "NK142_4_NO_TAX_DATA" || task.Variables["taxStatus"] != "REVIEW") {
			return deny("нет применимого основания отказа в учебном каталоге")
		}
		if task.ElementId == "AuthorizeClient" {
			reviewer, _ := task.Variables["reviewedBy"].(string)
			if decision == "CONTINUE" && task.Variables["screeningStatus"] == "PEP" && reviewer == "" {
				return deny("для учебного ПДЛ сначала нужно рассмотрение первого сотрудника")
			}
			if _, ok := flag("authorized"); !ok {
				return deny("укажите authorized")
			}
			if task.Variables["reviewedBy"] == user {
				return deny("согласующий должен отличаться от проверяющего")
			}
		}
	case "ConfirmContract":
		accepted, ok := flag("contractAccepted")
		if !ok {
			return deny("укажите contractAccepted")
		}
		if accepted && text("signedDocumentRef") == "" {
			return deny("нужна ссылка на учебное подтверждение подписания")
		}
	}
	return nil
}

func (t *Tasks) Complete(ctx context.Context, id uuid.UUID, user string, v map[string]any) error {
	task, err := t.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if err := Validate(task, user, v); err != nil {
		return err
	}
	if task.ElementId == "ReviewClient" {
		v["reviewedBy"] = user
	}
	if task.ElementId == "AuthorizeClient" {
		v["authorizedBy"] = user
	}
	return t.UserTaskService.Complete(ctx, id, user, v)
}
