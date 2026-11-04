package bankapi

import (
	"github.com/nicolas-vp/go-bpm-bank/engine"
	"testing"
)

func TestTaskFieldProtection(t *testing.T) {
	task := &engine.UserTask{ElementId: "ConfirmContract"}
	for _, v := range []map[string]any{
		{"accountOpened": true}, {"contractAccepted": "true"}, {"contractAccepted": true},
	} {
		if err := Validate(task, "operator", v); err == nil {
			t.Fatalf("опасное/неполное заполнение принято: %v", v)
		}
	}
	if err := Validate(task, "operator", map[string]any{"contractAccepted": false}); err != nil {
		t.Fatal(err)
	}
}

func TestDeclineNeedsApplicableBasis(t *testing.T) {
	task := &engine.UserTask{ElementId: "ReviewClient", Variables: map[string]any{"taxStatus": "VALID", "screeningStatus": "MATCH"}}
	v := map[string]any{"reviewDecision": "DECLINE", "legalBasisCode": "NK142_4_NO_TAX_DATA", "reason": "Высокий риск"}
	if err := Validate(task, "aml", v); err == nil {
		t.Fatal("нельзя отказывать физлицу только по риску")
	}
	task.Variables["taxStatus"] = "REVIEW"
	if err := Validate(task, "aml", v); err != nil {
		t.Fatal(err)
	}
}

func TestSecondReviewer(t *testing.T) {
	task := &engine.UserTask{ElementId: "AuthorizeClient", Variables: map[string]any{"reviewedBy": "chief"}}
	if err := Validate(task, "chief", map[string]any{"reviewDecision": "CONTINUE", "authorized": true, "reason": "Проверил"}); err == nil {
		t.Fatal("согласующий совпал с проверяющим")
	}
}

func TestEscalationDoesNotSkipFirstPEPReviewer(t *testing.T) {
	task := &engine.UserTask{ElementId: "AuthorizeClient", Variables: map[string]any{"screeningStatus": "PEP"}}
	if err := Validate(task, "chief", map[string]any{"reviewDecision": "CONTINUE", "authorized": true, "reason": "Эскалация"}); err == nil {
		t.Fatal("эскалация обошла рассмотрение первым сотрудником")
	}
}
