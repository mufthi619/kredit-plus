-- 000005_create_transactions_table.up.sql
CREATE TABLE IF NOT EXISTS transactions (
    id CHAR(36) PRIMARY KEY,
    customer_id CHAR(36) NOT NULL,
    asset_id CHAR(36) NOT NULL,
    contract_number VARCHAR(50) NOT NULL UNIQUE,
    otr_amount DECIMAL(15,2) NOT NULL,
    admin_fee DECIMAL(15,2) NOT NULL,
    interest_amount DECIMAL(15,2) NOT NULL,
    tenor_month INT NOT NULL,
    installment_amount DECIMAL(15,2) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'active', 'completed')),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (customer_id) REFERENCES customers(id),
    FOREIGN KEY (asset_id) REFERENCES assets(id)
    );

CREATE INDEX idx_transactions_customer_id ON transactions(customer_id);
CREATE INDEX idx_transactions_asset_id ON transactions(asset_id);