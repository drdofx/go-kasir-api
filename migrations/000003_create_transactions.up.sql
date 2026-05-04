CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    total_amount INTEGER NOT NULL CHECK (total_amount >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS transaction_details (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER REFERENCES transactions(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    subtotal INTEGER NOT NULL CHECK (subtotal >= 0)
);

CREATE INDEX IF NOT EXISTS transaction_details_transaction_id_idx ON transaction_details(transaction_id);
CREATE INDEX IF NOT EXISTS transaction_details_product_id_idx ON transaction_details(product_id);
