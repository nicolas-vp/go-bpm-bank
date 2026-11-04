// Package model содержит переменные учебных процессов GoBpmBank.
package model

type DataModel struct {
	ApplicationID      string            `json:"applicationId"`
	Scenario           string            `json:"scenario"`
	ApplicantName      string            `json:"applicantName"`
	BirthDate          string            `json:"birthDate"`
	Citizenship        string            `json:"citizenship"`
	PassportRef        string            `json:"passportRef"`
	Address            string            `json:"address"`
	Purpose            string            `json:"purpose"`
	TaxResidency       string            `json:"taxResidency"`
	IdentityVerified   bool              `json:"identityVerified"`
	DocumentsComplete  bool              `json:"documentsComplete"`
	DocumentStatus     string            `json:"documentStatus"`
	DocumentEvidence   map[string]string `json:"documentEvidence"`
	ScreeningEvidence  map[string]string `json:"screeningEvidence"`
	TaxEvidence        map[string]string `json:"taxEvidence"`
	ScreeningStatus    string            `json:"screeningStatus"`
	TaxStatus          string            `json:"taxStatus"`
	Route              string            `json:"route"`
	ReviewDecision     string            `json:"reviewDecision"`
	LegalBasisCode     string            `json:"legalBasisCode"`
	Reason             string            `json:"reason"`
	ReviewedBy         string            `json:"reviewedBy"`
	AuthorizedBy       string            `json:"authorizedBy"`
	Authorized         bool              `json:"authorized"`
	ContractAccepted   bool              `json:"contractAccepted"`
	SignedDocumentRef  string            `json:"signedDocumentRef"`
	ContractID         string            `json:"contractId"`
	CustomerID         string            `json:"customerId"`
	OperationID        string            `json:"operationId"`
	AccountID          string            `json:"accountId"`
	AccountNumber      string            `json:"accountNumber"`
	RegisteredAt       string            `json:"registeredAt"`
	CoreStatus         string            `json:"coreStatus"`
	AccountOpened      bool              `json:"accountOpened"`
	ArchiveStatus      string            `json:"archiveStatus"`
	TaxReportStatus    string            `json:"taxReportStatus"`
	NotificationStatus string            `json:"notificationStatus"`
	Outcome            string            `json:"outcome"`
}
