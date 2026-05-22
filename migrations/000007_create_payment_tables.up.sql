CREATE TABLE payment_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

INSERT INTO payment_types (name) VALUES ('cash'), ('card'), ('qris') ON CONFLICT (name) DO NOTHING;

CREATE TABLE transaction_payments (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER REFERENCES transactions(id) ON DELETE CASCADE,
    payment_type_id INTEGER REFERENCES payment_types(id),
    amount INTEGER NOT NULL CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_transaction_payments_transaction_id ON transaction_payments(transaction_id);
