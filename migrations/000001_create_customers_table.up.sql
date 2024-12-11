-- 000001_create_customers_table.up.sql
CREATE TABLE IF NOT EXISTS customers (
    id CHAR(36) PRIMARY KEY,
    nik VARCHAR(16) NOT NULL UNIQUE,
    full_name VARCHAR(100) NOT NULL,
    legal_name VARCHAR(100) NOT NULL,
    birth_place VARCHAR(100) NOT NULL,
    birth_date DATE NOT NULL,
    salary DECIMAL(15,2) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
    );

CREATE INDEX idx_customers_nik ON customers(nik);