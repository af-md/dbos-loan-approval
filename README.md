# DBOS Loan Approval Demo

A loan application processing system demonstrating DBOS workflows with AI-powered credit assessment using Google's Gemini.

## What is this?

This project shows how to use [DBOS Transact](https://github.com/dbos-inc/dbos-transact-go) to build durable, fault-tolerant workflows in Go. It implements a loan application process with AI-enhanced decision making:

1. **Duplicate Check** - Prevents processing the same application twice
2. **Save Application** - Persists loan details to database
3. **AI Credit Check** - Uses Google Gemini to analyze applicant CVs and assess creditworthiness
4. **Document Verification** - Validates required documents
5. **Manual Approval** - High-value loans (>$3000) require human approval via workflow messaging
6. **Final Decision** - Approve or reject based on all checks

## Key DBOS Features Demonstrated

- **Durable Workflows**: If the process crashes, it automatically resumes from the last completed step
- **Workflow Steps**: Each operation is checkpointed and recoverable
- **Send/Recv Messaging**: Workflows can communicate with each other for manual approval processes
- **Database Integration**: Automatic PostgreSQL schema management and connection handling
- **Error Handling**: Built-in retry logic and failure recovery
- **AI Integration**: LLM-powered decision making within durable workflows
- **Structured Logging**: Comprehensive observability with slog integration

## Prerequisites

### System Requirements
- Go 1.23+
- PostgreSQL 12+
- Google AI API key for Gemini

### PostgreSQL Setup

#### macOS (with Homebrew)
```bash
# Install PostgreSQL
brew install postgresql

# Start PostgreSQL service
brew services start postgresql

# Create database and user
createdb dbos
psql dbos -c "CREATE USER postgres WITH SUPERUSER PASSWORD 'your_password';"

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
    go build -o loanapp
   ```

   ```bash
   PGPASSWORD=$YOURPASSOWRD ./loanapp
   ```

Each step is:
- **Durable**: Survives crashes and restarts
- **Observable**: Progress is tracked in PostgreSQL
- [add more!!]

## About DBOS

DBOS (Database-Oriented Operating System) makes it easy to build reliable applications by providing durable workflows, queues, and other primitives backed by PostgreSQL.

Learn more: [docs.dbos.dev](https://docs.dbos.dev/)