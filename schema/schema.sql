

CREATE TABLE IF NOT EXISTS loan_applications (
    application_id VARCHAR(255) PRIMARY KEY,
    applicant_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(255),
    loan_amount DECIMAL(12,2),
    loan_purpose VARCHAR(50),
    annual_income DECIMAL(12,2),
    status VARCHAR(50) DEFAULT 'SUBMITTED',
    submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);