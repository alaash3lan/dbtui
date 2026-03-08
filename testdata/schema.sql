CREATE DATABASE IF NOT EXISTS dbtui_test;
USE dbtui_test;

-- customers table
CREATE TABLE IF NOT EXISTS customers (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100),
    city VARCHAR(50)
);

-- products table
CREATE TABLE IF NOT EXISTS products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50),
    price DECIMAL(10,2),
    stock INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- orders table
CREATE TABLE IF NOT EXISTS orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    customer_id INT,
    order_date DATE,
    total DECIMAL(10,2),
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);

-- order_items table
CREATE TABLE IF NOT EXISTS order_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id INT,
    product_id INT,
    quantity INT,
    price DECIMAL(10,2),
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Seed data: customers (10 rows, Ivy Chen has NULL email)
INSERT INTO customers (name, email, city) VALUES
    ('Alice Smith', 'alice@example.com', 'New York'),
    ('Bob Johnson', 'bob@example.com', 'London'),
    ('Charlie Brown', 'charlie@example.com', 'Tokyo'),
    ('Diana Prince', 'diana@example.com', 'Paris'),
    ('Eve Adams', 'eve@example.com', 'Berlin'),
    ('Frank Miller', 'frank@example.com', 'Sydney'),
    ('Grace Lee', 'grace@example.com', 'Toronto'),
    ('Henry Wilson', 'henry@example.com', 'Chicago'),
    ('Ivy Chen', NULL, 'San Francisco'),
    ('Jack Davis', 'jack@example.com', 'Seattle');

-- Seed data: products (12 rows)
INSERT INTO products (name, category, price, stock) VALUES
    ('Quantum Laptop', 'Electronics', 1299.99, 50),
    ('Nebula Phone', 'Electronics', 899.99, 120),
    ('Atlas Headphones', 'Electronics', 249.99, 200),
    ('Horizon Tablet', 'Electronics', 599.99, 75),
    ('Apex Keyboard', 'Accessories', 149.99, 300),
    ('Zenith Mouse', 'Accessories', 79.99, 450),
    ('Prism Monitor', 'Electronics', 449.99, 60),
    ('Echo Speaker', 'Audio', 199.99, 180),
    ('Bolt Charger', 'Accessories', 39.99, 500),
    ('Nova Camera', 'Photography', 799.99, 40),
    ('Titan Drone', 'Photography', 1199.99, 25),
    ('Flux Cable', 'Accessories', 19.99, 1000);

-- Seed data: orders
INSERT INTO orders (customer_id, order_date, total) VALUES
    (1, '2024-01-15', 1549.98),
    (2, '2024-01-20', 899.99),
    (3, '2024-02-01', 449.99),
    (1, '2024-02-10', 229.98),
    (5, '2024-02-15', 1999.98),
    (8, '2024-03-01', 79.99);

-- Seed data: order_items
INSERT INTO order_items (order_id, product_id, quantity, price) VALUES
    (1, 1, 1, 1299.99),
    (1, 6, 1, 249.99),
    (2, 2, 1, 899.99),
    (3, 7, 1, 449.99),
    (4, 5, 1, 149.99),
    (4, 6, 1, 79.99),
    (5, 1, 1, 1299.99),
    (5, 10, 1, 699.99),
    (6, 6, 1, 79.99);
