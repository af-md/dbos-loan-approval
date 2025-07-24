package main

import (
	"context"
	"database/sql"
	_ "embed" // Add this
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/dbos-inc/dbos-transact-go/dbos"
	_ "github.com/lib/pq"
	"google.golang.org/genai"
)

//go:embed schema/schema.sql
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
	CVText        string    `json:"cv_text"`
}

type LLMCreditAssessment struct {
	Score      int      `json:"score"`
	Reasoning  string   `json:"reasoning"`
	RedFlags   []string `json:"red_flags"`
	Positives  []string `json:"positives"`
	Confidence float64  `json:"confidence"`
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

	creditCheck, err := dbos.RunAsStep(ctx, CreditCheck, loanApp)
	if err != nil {
		return "", err
	}

	if !creditCheck.Approved {
		return fmt.Sprintf("Your loan application was rejected when checking your credit score. \n Your credi score was: %d", creditCheck.CreditScore), nil
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

		fmt.Println("Notification received from approval workflow")

		if response != StatusApproved {
			return "Loan application: Rejected", nil
		}
	}

	return fmt.Sprintf("Loan application: %s", StatusApproved), nil
}

func ApprovalWorkflow(ctx context.Context, workflowID string) (string, error) {
	// Receive loan application
	// Send approval back to the waiting loan workflow
	err := dbos.Send(ctx, dbos.WorkflowSendInput{
		DestinationID: workflowID,
		Topic:         "review-request",
		Message:       StatusApproved,
	})

	if err != nil {
		return "", err
	}

	fmt.Printf("✅ Sent approval decision '%s' to workflow: %s\n", StatusApproved, workflowID)
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
	fmt.Printf("Check if there are any duplicates: %s\n", loanApp.ApplicantName)

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
	fmt.Printf("Performing credit check for: %s using a LLM\n", loanApp.ApplicantName)

	prompt := fmt.Sprintf(`You are an experienced credit analyst working for a bank. 
Your job is to assess loan applicants based on documents they submit.
Always respond with valid JSON in this exact format:
{
  "score": <number 1-100>,
  "reasoning": "<brief explanation>",
  "red_flags": ["<concerning item>"],
  "positives": ["<good indicator>"],
  "confidence": <number 0.0-1.0>
}

Please analyze this document for creditworthiness:

--- DOCUMENT CONTENT ---
%s
--- END DOCUMENT ---

Consider: employment history, income stability, professional background, and any red flags.`, loanApp.CVText)

	// Create Gemini client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})
	if err != nil {
		return &CreditCheckResult{}, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	response, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(prompt), nil)
	if err != nil {
		log.Fatal(err)
	}

	var aiResponse string
	if len(response.Candidates) > 0 && len(response.Candidates[0].Content.Parts) > 0 {
		partOne := response.Candidates[0].Content.Parts[0]
		if partOne.Text != "" {
			aiResponse = string(partOne.Text)
		}
	}

	cleanedResponse := strings.TrimSpace(aiResponse)
	cleanedResponse = strings.ReplaceAll(cleanedResponse, "```json", "")
	cleanedResponse = strings.ReplaceAll(cleanedResponse, "```", "")
	cleanedResponse = strings.ReplaceAll(cleanedResponse, "`", "") // Remove any remaining backticks
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	// Parse the JSON response
	var assessment LLMCreditAssessment
	err = json.Unmarshal([]byte(cleanedResponse), &assessment)
	if err != nil {
		return &CreditCheckResult{}, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Check if score is higher than 60
	approved := assessment.Score > 60

	return &CreditCheckResult{
		CreditScore: assessment.Score,
		Approved:    approved,
	}, nil
}

func DocumentVerification(ctx context.Context, loanApp LoanApplication) (*DocumentVerificationResult, error) {
	fmt.Printf("Performing document verification for: %s\n", loanApp.ApplicantName)

	return &DocumentVerificationResult{
		Status:   "complete",
		Verified: true,
	}, nil
}

type NotificationResult struct {
	service string
}

func SendDecisionNotification(ctx context.Context, loanApp LoanApplication) (*NotificationResult, error) {
	fmt.Printf("Sending email notification via Twilio to customer: %s\n", loanApp.ApplicantName)

	return &NotificationResult{
		service: "Twilio",
	}, nil
}

// Application Submission
// Initial Validation
// Credit Check
// Document Verification
// Risk Assessment
// Decision Logic
// Manual Review (if needed)
// Final Approval/Rejection
// Notification & Setup
