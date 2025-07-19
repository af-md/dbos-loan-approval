package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dbos-inc/dbos-transact-go/dbos"

	"dbos-loan-approval/src"
)

var (
	processOrderWf   = dbos.WithWorkflow(src.LoanProcessWorkflow)
	processOrderWfV2 = dbos.WithWorkflow(src.LoanProcessWorkflowV2)
	approveOrderWf   = dbos.WithWorkflow(src.ApprovalWorkflow)
)

func init() {
	gob.Register(src.DuplicateCheckResult{})
	gob.Register(src.SaveResult{})
	gob.Register(src.DocumentVerificationResult{})
	gob.Register(src.CreditCheckResult{})
}

func submitLoanApplicationHanlder(w http.ResponseWriter, r *http.Request) {
	var loanApp src.LoanApplication
	if err := json.NewDecoder(r.Body).Decode(&loanApp); err != nil {
		http.Error(w, "Invalid Loan Application JSON", http.StatusBadRequest)
		return
	}

	loanApp.SubmittedAt = time.Now()

	handle, err := processOrderWf(context.Background(), loanApp)
	if err != nil {
		panic(err)
	}

	result, err := handle.GetResult(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Printf("Result: %s\n", result)

	json.NewEncoder(w).Encode(map[string]string{"result": result})
}

func approvalHandler(w http.ResponseWriter, r *http.Request) {

	workflowID := r.URL.Query().Get("workflow_id")
	if workflowID == "" {
		http.Error(w, "Missing workflow_id parameter", http.StatusBadRequest)
		return
	}

	fmt.Printf("APPROVE FOR WORKFLOW ID: %s", workflowID)

	handle, err := approveOrderWf(r.Context(), workflowID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := handle.GetResult(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"result": result})
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

	http.HandleFunc("/submit-loan", submitLoanApplicationHanlder)
	http.HandleFunc("/approve", approvalHandler)

	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", nil)
}
