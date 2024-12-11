-- 000006_create_transaction_details_table.up.sql
CREATE TABLE IF NOT EXISTS transaction_details (
    id CHAR(36) PRIMARY KEY,
    transaction_id CHAR(36) NOT NULL,
    installment_number INT NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    due_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'paid', 'overdue')),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id)
    );

CREATE INDEX idx_transaction_details_transaction_id ON transaction_details(transaction_id);