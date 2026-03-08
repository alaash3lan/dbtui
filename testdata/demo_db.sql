-- Demo database for dbtui screenshots
CREATE DATABASE IF NOT EXISTS dbtui_demo;
USE dbtui_demo;

-- Products table
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS customers;
DROP TABLE IF EXISTS categories;

CREATE TABLE categories (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    category_id INT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    stock INT DEFAULT 0,
    sku VARCHAR(50) UNIQUE NOT NULL,
    status ENUM('active','discontinued','draft') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE customers (
    id INT AUTO_INCREMENT PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(20),
    city VARCHAR(100),
    country VARCHAR(50) DEFAULT 'US',
    is_premium BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    customer_id INT NOT NULL,
    status ENUM('pending','processing','shipped','delivered','cancelled') DEFAULT 'pending',
    total DECIMAL(12,2) NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    shipped_at TIMESTAMP NULL,
    FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE order_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    unit_price DECIMAL(10,2) NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT,
    INDEX idx_order_product (order_id, product_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed categories
INSERT INTO categories (name, description) VALUES
('Electronics', 'Phones, laptops, and accessories'),
('Clothing', 'Apparel and fashion items'),
('Books', 'Fiction, non-fiction, and textbooks'),
('Home & Garden', 'Furniture, decor, and outdoor'),
('Sports', 'Equipment and athletic wear');

-- Seed products
INSERT INTO products (name, category_id, price, stock, sku, status) VALUES
('MacBook Pro 16"', 1, 2499.99, 45, 'ELEC-MBP16', 'active'),
('iPhone 15 Pro', 1, 1199.00, 120, 'ELEC-IP15P', 'active'),
('AirPods Pro', 1, 249.00, 300, 'ELEC-APP2', 'active'),
('Samsung Galaxy S24', 1, 999.99, 85, 'ELEC-SGS24', 'active'),
('Sony WH-1000XM5', 1, 349.99, 60, 'ELEC-SNWH5', 'active'),
('iPad Air', 1, 599.00, 95, 'ELEC-IPAD5', 'active'),
('Dell XPS 15', 1, 1899.00, 30, 'ELEC-DXPS15', 'active'),
('Leather Jacket', 2, 189.50, 25, 'CLTH-LJ001', 'active'),
('Denim Jeans Slim', 2, 69.99, 200, 'CLTH-DJ001', 'active'),
('Cotton T-Shirt Pack', 2, 29.99, 500, 'CLTH-TS003', 'active'),
('Running Shoes Pro', 2, 129.00, 80, 'CLTH-RS001', 'active'),
('Winter Parka', 2, 249.00, 15, 'CLTH-WP001', 'discontinued'),
('Clean Code', 3, 42.50, 150, 'BOOK-CC001', 'active'),
('Design Patterns', 3, 48.99, 90, 'BOOK-DP001', 'active'),
('The Pragmatic Programmer', 3, 52.00, 110, 'BOOK-PP001', 'active'),
('DDIA', 3, 45.00, 75, 'BOOK-DDIA1', 'active'),
('Atomic Habits', 3, 16.99, 300, 'BOOK-AH001', 'active'),
('Standing Desk Oak', 4, 599.00, 20, 'HOME-SD001', 'active'),
('Ergonomic Chair', 4, 449.00, 35, 'HOME-EC001', 'active'),
('LED Desk Lamp', 4, 45.99, 150, 'HOME-DL001', 'active'),
('Plant Pot Set', 4, 24.99, 200, 'HOME-PP001', 'active'),
('Yoga Mat Premium', 5, 34.99, 180, 'SPRT-YM001', 'active'),
('Resistance Bands Set', 5, 19.99, 250, 'SPRT-RB001', 'active'),
('Dumbbells 20kg Pair', 5, 89.00, 40, 'SPRT-DB020', 'active'),
('Fitness Tracker', 5, 149.99, 100, 'SPRT-FT001', 'active');

-- Seed customers
INSERT INTO customers (first_name, last_name, email, phone, city, country, is_premium) VALUES
('Alice', 'Johnson', 'alice.j@example.com', '+1-555-0101', 'New York', 'US', TRUE),
('Bob', 'Smith', 'bob.smith@example.com', '+1-555-0102', 'San Francisco', 'US', FALSE),
('Clara', 'Martinez', 'clara.m@example.com', '+1-555-0103', 'Austin', 'US', TRUE),
('David', 'Lee', 'david.lee@example.com', '+44-20-7946', 'London', 'UK', FALSE),
('Emma', 'Wilson', 'emma.w@example.com', '+1-555-0105', 'Chicago', 'US', TRUE),
('Frank', 'Brown', 'frank.b@example.com', '+49-30-1234', 'Berlin', 'DE', FALSE),
('Grace', 'Davis', 'grace.d@example.com', '+1-555-0107', 'Seattle', 'US', FALSE),
('Hassan', 'Ahmed', 'hassan.a@example.com', '+971-4-5678', 'Dubai', 'AE', TRUE),
('Ivy', 'Chen', 'ivy.chen@example.com', '+86-21-9876', 'Shanghai', 'CN', FALSE),
('James', 'Taylor', 'james.t@example.com', '+1-555-0110', 'Boston', 'US', TRUE),
('Karen', 'White', 'karen.w@example.com', '+1-555-0111', 'Denver', 'US', FALSE),
('Liam', 'Garcia', 'liam.g@example.com', '+34-91-1234', 'Madrid', 'ES', FALSE);

-- Seed orders
INSERT INTO orders (customer_id, status, total, notes, created_at, shipped_at) VALUES
(1, 'delivered', 2749.99, 'Gift wrap requested', '2025-01-15 10:30:00', '2025-01-17 14:00:00'),
(2, 'shipped', 69.99, NULL, '2025-02-01 09:15:00', '2025-02-03 11:00:00'),
(3, 'processing', 1448.00, 'Express delivery', '2025-02-20 16:45:00', NULL),
(1, 'delivered', 297.99, NULL, '2025-01-28 12:00:00', '2025-01-30 09:00:00'),
(4, 'pending', 48.99, NULL, '2025-03-01 08:00:00', NULL),
(5, 'delivered', 689.00, 'Office purchase', '2025-01-10 11:30:00', '2025-01-13 10:00:00'),
(6, 'cancelled', 249.00, 'Customer changed mind', '2025-02-14 15:20:00', NULL),
(7, 'shipped', 94.98, NULL, '2025-02-25 13:00:00', '2025-02-27 16:00:00'),
(8, 'delivered', 3648.99, 'VIP client', '2025-01-05 09:00:00', '2025-01-08 12:00:00'),
(3, 'processing', 129.00, NULL, '2025-03-05 10:00:00', NULL),
(9, 'pending', 45.99, NULL, '2025-03-07 14:30:00', NULL),
(10, 'delivered', 549.98, 'Birthday gift', '2025-02-10 11:00:00', '2025-02-13 09:00:00'),
(11, 'shipped', 34.99, NULL, '2025-03-02 16:00:00', '2025-03-04 10:00:00'),
(12, 'pending', 178.98, NULL, '2025-03-08 08:45:00', NULL),
(5, 'delivered', 1199.00, NULL, '2025-02-05 12:30:00', '2025-02-08 14:00:00');

-- Seed order items
INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES
(1, 1, 1, 2499.99), (1, 3, 1, 249.00),
(2, 9, 1, 69.99),
(3, 2, 1, 1199.00), (3, 3, 1, 249.00),
(4, 3, 1, 249.00), (4, 10, 1, 29.99), (4, 23, 1, 19.99),
(5, 14, 1, 48.99),
(6, 18, 1, 599.00), (6, 24, 1, 89.00),
(7, 12, 1, 249.00),
(8, 22, 1, 34.99), (8, 20, 1, 45.99), (8, 17, 1, 16.99),
(9, 1, 1, 2499.99), (9, 5, 1, 349.99), (9, 6, 1, 599.00), (9, 7, 1, 1899.00),
(10, 11, 1, 129.00),
(11, 20, 1, 45.99),
(12, 15, 1, 52.00), (12, 13, 1, 42.50), (12, 16, 1, 45.00), (12, 14, 1, 48.99),
(13, 22, 1, 34.99),
(14, 24, 1, 89.00), (14, 25, 1, 149.99),
(15, 2, 1, 1199.00);

SELECT 'Demo database created successfully!' AS status;
