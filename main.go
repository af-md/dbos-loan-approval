package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/dbos-inc/dbos-transact-go/dbos"

	"dbos-loan-approval/src"
)

var (
	processOrderWf = dbos.WithWorkflow(src.LoanProcessWorkflow)
)

func init() {
	gob.Register(src.DuplicateCheckResult{})
	gob.Register(src.SaveResult{})
	gob.Register(src.DocumentVerificationResult{})
	gob.Register(src.CreditCheckResult{})
}

func main() {
	err := dbos.Launch()
	if err != nil {
		panic(err)
	}

	defer dbos.Shutdown()

	// init database
	err = src.InitializeSchema()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize schema: %v", err))
	}

	loanApp := src.LoanApplication{
		ApplicationID: "LOAN-2024-004",
		ApplicantName: "John Doe",
		Email:         "john.doe@example.com",
		Phone:         "+1-555-0123",
		LoanAmount:    50000,
		LoanPurpose:   "personal",
		AnnualIncome:  75000,
		SubmittedAt:   time.Now(),
	}

	handle, err := processOrderWf(context.Background(), loanApp)
	if err != nil {
		panic(err)
	}

	result, err := handle.GetResult(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Printf("Result: %s\n", result)

	time.Sleep(2 * time.Second)
}
