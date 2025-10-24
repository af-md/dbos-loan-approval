package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/google/uuid"
)

func init() {
	gob.Register(DuplicateCheckResult{})
	gob.Register(SaveResult{})
	gob.Register(DocumentVerificationResult{})
	gob.Register(CreditCheckResult{})
}

var (
	dbosContext dbos.DBOSContext
)

// this could be built using key business values
func generateIdempotencyKey() string {
	return uuid.New().String()
}

func submitLoanApplicationHanlder(w http.ResponseWriter, r *http.Request) {
	var loanApp LoanApplication
	if err := json.NewDecoder(r.Body).Decode(&loanApp); err != nil {
		http.Error(w, "Invalid Loan Application JSON", http.StatusBadRequest)
		return
	}

	loanApp.SubmittedAt = time.Now()

	idempotencyKey := generateIdempotencyKey()

	handle, err := dbos.RunWorkflow(dbosContext, LoanProcessWorkflow, loanApp, dbos.WithWorkflowID(idempotencyKey))
	if err != nil {
		panic(err)
	}

	result, err := handle.GetResult()
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

	idempotencyKey := generateIdempotencyKey()

	fmt.Printf("APPROVE FOR WORKFLOW ID: %s", workflowID)

	handle, err := dbos.RunWorkflow(dbosContext, ApprovalWorkflow, workflowID, dbos.WithWorkflowID(idempotencyKey))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := handle.GetResult()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"result": result})
}

func main() {

	password := url.QueryEscape(os.Getenv("PGPASSWORD"))
	if password == "" {
		password = "defaultpassword"
		//panic(fmt.Errorf("PGPASSWORD environment variable not set"))
	}

	if os.Getenv("DBOS_DATABASE_URL") == "" {
		databaseURL := fmt.Sprintf("postgres://postgres:%s@localhost:5432/postgres?sslmode=disable", password)
		os.Setenv("DBOS_DATABASE_URL", databaseURL)
	}

	databaseURL := os.Getenv("DBOS_DATABASE_URL")

	var err error
	dbosContext, err = dbos.NewDBOSContext(context.Background(), dbos.Config{
		AppName:     "loan-app",
		DatabaseURL: databaseURL,
		AdminServer: true,
	})
	if err != nil {
		panic(err)
	}

	// register workflows
	dbos.RegisterWorkflow(dbosContext, LoanProcessWorkflow)
	dbos.RegisterWorkflow(dbosContext, ApprovalWorkflow)

	err = dbosContext.Launch()
	if err != nil {
		panic(err)
	}

	defer dbosContext.Shutdown(10 * time.Second)

	// init database
	err = InitializeSchema()
	if err != nil {
		fmt.Printf("Panicking because schema initialization failed %v", err)
		panic(fmt.Sprintf("Failed to initialize schema: %v", err))
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "healthy", "message": "app is running"}`)
	})

	http.HandleFunc("/submit-loan", submitLoanApplicationHanlder)
	http.HandleFunc("/approve", approvalHandler)

	fmt.Println("Server starting on :3000")
	http.ListenAndServe(":3000", nil)
}
