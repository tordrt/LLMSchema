-- Test schema for SQLite

-- Drop tables if they exist
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS users;

-- Users table with CHECK constraint for status (SQLite doesn't have ENUM)
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL,
    status TEXT DEFAULT 'active' CHECK(status IN ('active', 'inactive', 'banned')),
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    price REAL NOT NULL,
    stock INTEGER DEFAULT 0,
    category TEXT NOT NULL CHECK(category IN ('electronics', 'clothing', 'food', 'books')),
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for products
CREATE INDEX idx_category ON products(category);
CREATE INDEX idx_price ON products(price);

CREATE TABLE orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    total_amount REAL NOT NULL,
    order_date TEXT DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'processing', 'shipped', 'delivered', 'cancelled')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Create indexes for orders
CREATE INDEX idx_user_date ON orders(user_id, order_date);
CREATE INDEX idx_status ON orders(status);

CREATE TABLE order_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    unit_price REAL NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id),
    UNIQUE(order_id, product_id)
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
