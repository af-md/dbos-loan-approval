package loan

import (
	"context"
	"fmt"
	"time"

	"github.com/dbos-inc/dbos-transact-go/dbos"

	"dbos-loan-approval/src"
)

var (
	processOrderWf = dbos.WithWorkflow(src.LoanProcessWorkflow)
)

func main() {
	err := dbos.Launch()
	if err != nil {
		panic(err)
	}

	defer dbos.Shutdown()

	loanApp := src.LoanApplication{
		ApplicationID: "LOAN-2024-001",
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

	fmt.Println(handle)

	time.Sleep(2 * time.Second)
}
