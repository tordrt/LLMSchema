-- Test schema for PostgreSQL
-- Run this to create a test database:
-- createdb testdb
-- psql testdb < test_postgres_schema.sql
-- Then test with: ./llmschema --db-url "postgres://localhost/testdb"

-- Drop tables if they exist
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS users;

-- Drop enum types if they exist
DROP TYPE IF EXISTS user_status;
DROP TYPE IF EXISTS product_category;
DROP TYPE IF EXISTS order_status;

-- Create enum types
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'banned');
CREATE TYPE product_category AS ENUM ('electronics', 'clothing', 'food', 'books');
CREATE TYPE order_status AS ENUM ('pending', 'processing', 'shipped', 'delivered', 'cancelled');

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    status user_status DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock INT DEFAULT 0,
    category product_category NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_category ON products(category);
CREATE INDEX idx_price ON products(price);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    total_amount DECIMAL(10, 2) NOT NULL,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status order_status DEFAULT 'pending',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_date ON orders(user_id, order_date);
CREATE INDEX idx_status ON orders(status);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    unit_price DECIMAL(10, 2) NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id),
    UNIQUE (order_id, product_id)
);

-- Insert some test data
INSERT INTO users (username, email, status) VALUES
    ('alice', 'alice@example.com', 'active'),
    ('bob', 'bob@example.com', 'active'),
    ('charlie', 'charlie@example.com', 'inactive');

INSERT INTO products (name, description, price, stock, category) VALUES
    ('Laptop', 'High-performance laptop', 999.99, 10, 'electronics'),
    ('T-Shirt', 'Cotton t-shirt', 19.99, 100, 'clothing'),
    ('Coffee Beans', 'Premium arabica beans', 12.99, 50, 'food');

INSERT INTO orders (user_id, total_amount, status) VALUES
    (1, 999.99, 'delivered'),
    (2, 32.98, 'processing');

INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES
    (1, 1, 1, 999.99),
    (2, 2, 1, 19.99),
    (2, 3, 1, 12.99);