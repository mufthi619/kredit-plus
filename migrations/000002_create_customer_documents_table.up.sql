-- 000002_create_customer_documents_table.up.sql
CREATE TABLE IF NOT EXISTS customer_documents (
    id CHAR(36) PRIMARY KEY,
    customer_id CHAR(36) NOT NULL,
    document_type VARCHAR(50) NOT NULL CHECK (document_type IN ('ktp', 'selfie')),
    document_url VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (customer_id) REFERENCES customers(id)
    );

CREATE INDEX idx_customer_documents_customer_id ON customer_documents(customer_id);