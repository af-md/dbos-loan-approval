package src

import (
	"context"
	"fmt"
	"time"

	"github.com/dbos-inc/dbos-transact-go/dbos"
)

var (
	processOrderWf = dbos.WithWorkflow(LoanProcessWorkflow)
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

func LoanProcessWorkflow(ctx context.Context, loanApp LoanApplication) (string, error) {
	// check stock
	_, err := dbos.RunAsStep(ctx, Credit_Check, loanApp)
	if err != nil {
		return "", err
	}

	fmt.Println("this is normal code")

	sum := 2 + 2

	fmt.Println(sum)

	_, err = dbos.RunAsStep(ctx, Document_Verification, loanApp)
	if err != nil {
		return "", nil
	}

	return "Order is ready for collection", nil
}

func Credit_Check(ctx context.Context, loanApp LoanApplication) (string, error) {
	fmt.Printf("checking stock for order ID: %d \n", loanApp.ApplicationID)
	return "stock found", nil
}

func Document_Verification(ctx context.Context, loanApp LoanApplication) (string, error) {
	fmt.Printf("sending email to customer ID: %d", loanApp.ApplicationID)
	return "email sent", nil
}
