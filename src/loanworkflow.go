package src

import (
	"context"
	"database/sql"
	_ "embed" // Add this
	"fmt"
	"time"

	"github.com/dbos-inc/dbos-transact-go/dbos"
	_ "github.com/lib/pq"
)

//go:embed schema.sql
var schemaSQL string

const (
	StatusSubmitted = "SUBMITTED"
	StatusApproved  = "APPROVED"
	StatusRejected  = "REJECTED"
)

type LoanApplication struct {
	ApplicationID string    `json:"application_id"`
	ApplicantName string    `json:"applicant_name"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	LoanAmount    float64   `json:"loan_amount"`
	LoanPurpose   string    `json:"loan_purpose"` // "home", "auto", "personal"
	AnnualIncome  float64   `json:"annual_income"`
	SubmittedAt   time.Time `json:"submitted_at"`
}

type DuplicateCheckResult struct {
	IsDuplicate bool
}

type CreditCheckResult struct {
	CreditScore int  `json:"credit_score"`
	Approved    bool `json:"approved"`
}

type DocumentVerificationResult struct {
	Status   string `json:"status"` // "complete", "pending"
	Verified bool   `json:"verified"`
}

type SaveResult struct {
	ApplicationID string
	Saved         bool
}

func LoanProcessWorkflow(ctx context.Context, loanApp LoanApplication) (string, error) {
	// check if already processed
	duplicateCheckResult, err := dbos.RunAsStep(ctx, CheckIfDuplicate, loanApp)
	if err != nil {
		return "", err
	}

	if duplicateCheckResult.IsDuplicate {
		return "already processed", nil
	}

	// Save the application
	_, err = dbos.RunAsStep(ctx, SaveLoanApplication, loanApp)
	if err != nil {
		return "", err
	}

	_, err = dbos.RunAsStep(ctx, CreditCheck, loanApp)
	if err != nil {
		return "", err
	}

	documentResult, err := dbos.RunAsStep(ctx, DocumentVerification, loanApp)
	if err != nil {
		return "", nil
	}

	if !documentResult.Verified {
		return "Application pending - documents need verification", nil
	}

	workflowState, ok := ctx.Value(dbos.WorkflowStateKey).(*dbos.WorkflowState)
	if !ok {
		return "", fmt.Errorf("workflow state not found")
	}

	workflowID := workflowState.WorkflowID

	fmt.Printf("Loan workflow ID: %s\n", workflowID)

	if loanApp.LoanAmount > 3000 {
		// send for manual approval

		topic := "review-request"

		fmt.Println("Notification send for approval")

		// wait for manual approval response
		response, err := dbos.Recv[string](ctx, dbos.WorkflowRecvInput{
			Topic:   topic,
			Timeout: 60 * time.Second,
		})

		if err != nil {
			return "", fmt.Errorf("failed during manual approval process received: %s", err.Error())
		}
		fmt.Println("Notification received from approval")

		if response != "APPROVED" {
			return "Loan application: Rejected", nil
		}
	}

	return "Loan application: Approved", nil
}

func LoanProcessWorkflowV2(ctx context.Context, loanApp LoanApplication) (string, error) {
	workflowState, ok := ctx.Value(dbos.WorkflowStateKey).(*dbos.WorkflowState)
	if !ok {
		return "", fmt.Errorf("workflow state not found")
	}

	workflowID := workflowState.WorkflowID

	fmt.Printf("Loan workflow ID: %s\n", workflowID)

	if loanApp.LoanAmount > 3000 {
		// send for manual approval

		topic := "review-request"

		fmt.Println("Notification send for approval")

		// wait for manual approval response
		response, err := dbos.Recv[string](ctx, dbos.WorkflowRecvInput{
			Topic:   topic,
			Timeout: 60 * time.Second,
		})

		if err != nil {
			return "", fmt.Errorf("failed during manual approval process received: %s", err.Error())
		}
		fmt.Println("Notification received from approval")

		if response != "APPROVED" {
			return "Loan application: Rejected", nil
		}
	}

	return "Loan application: Approved", nil
}

func ApprovalWorkflow(ctx context.Context, workflowID string) (string, error) {
	// Receive loan application
	// Send approval back to the waiting loan workflow
	err := dbos.Send(ctx, dbos.WorkflowSendInput{
		DestinationID: workflowID,
		Topic:         "review-request",
		Message:       "APPROVED",
	})

	if err != nil {
		return "", err
	}

	fmt.Printf("✅ Sent approval decision '%s' to workflow: %s\n", "APPROVED", workflowID)
	return fmt.Sprintf("Approval sent to %s", workflowID), nil
}

func SaveLoanApplication(ctx context.Context, loanApp LoanApplication) (*SaveResult, error) {
	fmt.Printf("Saving loan application: %s\n", loanApp.ApplicationID)

	db, err := getDBConnection()
	if err != nil {
		return &SaveResult{}, fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`
        INSERT INTO loan_applications (application_id, applicant_name, email, phone, loan_amount, loan_purpose, annual_income, status, submitted_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		loanApp.ApplicationID, loanApp.ApplicantName, loanApp.Email, loanApp.Phone,
		loanApp.LoanAmount, loanApp.LoanPurpose, loanApp.AnnualIncome, StatusSubmitted, loanApp.SubmittedAt)

	if err != nil {
		return &SaveResult{}, fmt.Errorf("failed to save application: %w", err)
	}

	return &SaveResult{
		ApplicationID: loanApp.ApplicationID,
		Saved:         true,
	}, nil
}

func CheckIfDuplicate(ctx context.Context, loanApp LoanApplication) (*DuplicateCheckResult, error) {
	db, err := getDBConnection()
	if err != nil {
		return &DuplicateCheckResult{}, fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	// Check if application ID already exists
	var existingID string
	err = db.QueryRow("SELECT application_id FROM loan_applications WHERE application_id = $1", loanApp.ApplicationID).Scan(&existingID)
	if err == sql.ErrNoRows {
		// No duplicate found
		return &DuplicateCheckResult{}, nil
	}

	if err != nil {
		return &DuplicateCheckResult{}, fmt.Errorf("database query failed: %w", err)
	}

	// Duplicate found
	fmt.Printf("Duplicate application found: %s\n", existingID)
	return &DuplicateCheckResult{IsDuplicate: true}, nil
}

func CreditCheck(ctx context.Context, loanApp LoanApplication) (*CreditCheckResult, error) {
	fmt.Printf("Performing credit check for: %s\n", loanApp.ApplicantName)

	return &CreditCheckResult{
		CreditScore: 720,
		Approved:    true,
	}, nil
}

func DocumentVerification(ctx context.Context, loanApp LoanApplication) (*DocumentVerificationResult, error) {
	return &DocumentVerificationResult{
		Status:   "complete",
		Verified: true,
	}, nil
}
