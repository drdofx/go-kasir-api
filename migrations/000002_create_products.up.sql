CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price INTEGER NOT NULL CHECK (price >= 0),
    stock INTEGER NOT NULL CHECK (stock >= 0),
    category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS products_category_id_idx ON products(category_id);
