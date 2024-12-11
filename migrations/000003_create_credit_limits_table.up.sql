-- 000003_create_credit_limits_table.up.sql
CREATE TABLE IF NOT EXISTS credit_limits (
    id CHAR(36) PRIMARY KEY,
    customer_id CHAR(36) NOT NULL,
    tenor_month INT NOT NULL,
    limit_amount DECIMAL(15,2) NOT NULL,
    used_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (customer_id) REFERENCES customers(id)
    );

CREATE INDEX idx_credit_limits_customer_id ON credit_limits(customer_id);