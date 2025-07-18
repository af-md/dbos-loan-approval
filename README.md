# DBOS Loan Approval Demo

A simple project demonstrating DBOS workflows using a loan application process.

## What is this?

This project shows how to use [DBOS Transact](https://github.com/dbos-inc/dbos-transact-go) to build durable, fault-tolerant workflows in Go. It implements a basic loan application process with the following steps:

1. **Credit Check** - Verify applicant's credit status
2. **Document Verification** - Validate required documents
3. **Other steps..**
4. **Final Processing** - Complete the loan application

## Key DBOS Features Demonstrated

- **Durable Workflows**: If the process crashes, it automatically resumes from the last completed step
- [add more!!]


## Running the Project

1. **Prerequisites**: 
   - Go 1.23+
   - PostgreSQL running locally
   - Set `PGPASSWORD` environment variable
   - [Needs to more guide to setup Postgres]

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Run the application**:
   ```bash
   PGPASSWORD=$YOURPASSOWRD go run main.go
   ```

## How it Works

The loan application goes through a workflow with multiple steps:

```go
func LoanProcessWorkflow(ctx context.Context, loanApp LoanApplication) (string, error) {
    // Step 1: Credit Check
    _, err := dbos.RunAsStep(ctx, Credit_Check, loanApp)
    if err != nil {
        return "", err
    }

    // Step 2: Document Verification  
    _, err = dbos.RunAsStep(ctx, Document_Verification, loanApp)
    if err != nil {
        return "", err
    }

    return "Loan application processed", nil
}
```

Each step is:
- **Durable**: Survives crashes and restarts
- **Observable**: Progress is tracked in PostgreSQL
- [add more!!]

## About DBOS

DBOS (Database-Oriented Operating System) makes it easy to build reliable applications by providing durable workflows, queues, and other primitives backed by PostgreSQL.

Learn more: [docs.dbos.dev](https://docs.dbos.dev/)