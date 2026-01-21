-- Sample E-Commerce Data
-- Insert sample products, orders, and reviews for demonstration

-- Insert sample products (electronics category)
INSERT INTO products (name, description, category, brand, price, stock_quantity, rating, review_count) VALUES
('Pro Wireless Headphones', 'Premium noise-cancelling wireless headphones with 30-hour battery life. Features active noise cancellation, transparency mode, and premium audio drivers for exceptional sound quality.', 'Electronics', 'AudioMax', 299.99, 150, 4.7, 1245),
('Ultra 4K Smart TV 55"', 'Crystal clear 4K UHD display with HDR10+ support. Smart TV features include voice control, streaming apps, and seamless smartphone integration.', 'Electronics', 'VisionTech', 799.99, 45, 4.5, 892),
('Gaming Laptop Pro', 'High-performance gaming laptop with RTX 4080 graphics, 32GB RAM, and 1TB NVMe SSD. 165Hz display for smooth gameplay.', 'Electronics', 'GameForce', 1899.99, 25, 4.8, 567),
('Wireless Earbuds Elite', 'Compact wireless earbuds with premium sound quality and 8-hour battery life. Water-resistant design perfect for workouts.', 'Electronics', 'AudioMax', 149.99, 300, 4.4, 2341),
('Smart Watch Series 5', 'Advanced fitness tracking, heart rate monitoring, and GPS. Always-on display with 5-day battery life.', 'Electronics', 'TechWear', 399.99, 200, 4.6, 1876);

-- Insert sample products (home & kitchen category)
INSERT INTO products (name, description, category, brand, price, stock_quantity, rating, review_count) VALUES
('Robot Vacuum Pro', 'Intelligent robot vacuum with mapping technology, automatic dirt disposal, and app control. Perfect for pet owners.', 'Home & Kitchen', 'CleanBot', 549.99, 75, 4.3, 654),
('Smart Coffee Maker', 'Programmable coffee maker with WiFi connectivity. Brew your perfect cup from your smartphone.', 'Home & Kitchen', 'BrewMaster', 199.99, 120, 4.2, 432),
('Air Purifier HEPA', 'Medical-grade HEPA air purifier for rooms up to 500 sq ft. Removes 99.97% of airborne particles.', 'Home & Kitchen', 'PureAir', 279.99, 90, 4.7, 789),
('Instant Pot Multi-Cooker', 'Versatile 8-quart multi-cooker with 11 cooking programs. Pressure cook, slow cook, steam, and more.', 'Home & Kitchen', 'InstaCook', 129.99, 200, 4.8, 3456),
('Smart Thermostat', 'Energy-saving smart thermostat with learning capability and remote control via app.', 'Home & Kitchen', 'EcoTemp', 249.99, 150, 4.5, 1123);

-- Insert sample products (sports & outdoors)
INSERT INTO products (name, description, category, brand, price, stock_quantity, rating, review_count) VALUES
('Mountain Bike Pro', 'Full-suspension mountain bike with carbon frame and hydraulic disc brakes. Perfect for trail riding.', 'Sports & Outdoors', 'TrailBlazer', 1499.99, 30, 4.6, 234),
('Yoga Mat Premium', 'Extra thick eco-friendly yoga mat with alignment guides. Non-slip surface for all yoga styles.', 'Sports & Outdoors', 'ZenFit', 49.99, 500, 4.4, 1567),
('Camping Tent 4-Person', 'Waterproof 4-person tent with easy setup. Features mesh windows and rainfly for all-weather camping.', 'Sports & Outdoors', 'OutdoorPro', 199.99, 80, 4.5, 432),
('Running Shoes Ultra', 'Lightweight running shoes with responsive cushioning and breathable mesh upper. Ideal for marathon training.', 'Sports & Outdoors', 'SpeedRun', 159.99, 250, 4.7, 2134),
('Fitness Tracker Band', 'Slim fitness tracker with heart rate monitor, sleep tracking, and 7-day battery life.', 'Sports & Outdoors', 'FitLife', 79.99, 400, 4.3, 987);

-- Insert sample orders for analytics
INSERT INTO orders (customer_id, product_id, quantity, unit_price, total_price, status, region, created_at) VALUES
(1001, 1, 1, 299.99, 299.99, 'delivered', 'North America', NOW() - INTERVAL '30 days'),
(1002, 3, 1, 1899.99, 1899.99, 'delivered', 'Europe', NOW() - INTERVAL '28 days'),
(1003, 5, 2, 399.99, 799.98, 'delivered', 'Asia Pacific', NOW() - INTERVAL '25 days'),
(1004, 9, 1, 129.99, 129.99, 'delivered', 'North America', NOW() - INTERVAL '22 days'),
(1005, 4, 3, 149.99, 449.97, 'shipped', 'Europe', NOW() - INTERVAL '20 days'),
(1006, 2, 1, 799.99, 799.99, 'delivered', 'North America', NOW() - INTERVAL '18 days'),
(1007, 6, 1, 549.99, 549.99, 'delivered', 'Asia Pacific', NOW() - INTERVAL '15 days'),
(1008, 14, 2, 159.99, 319.98, 'delivered', 'Europe', NOW() - INTERVAL '12 days'),
(1009, 1, 1, 299.99, 299.99, 'shipped', 'North America', NOW() - INTERVAL '10 days'),
(1010, 8, 1, 279.99, 279.99, 'delivered', 'Asia Pacific', NOW() - INTERVAL '8 days'),
(1011, 11, 1, 1499.99, 1499.99, 'processing', 'Europe', NOW() - INTERVAL '5 days'),
(1012, 7, 2, 199.99, 399.98, 'shipped', 'North America', NOW() - INTERVAL '3 days'),
(1013, 3, 1, 1899.99, 1899.99, 'processing', 'Asia Pacific', NOW() - INTERVAL '2 days'),
(1014, 10, 1, 249.99, 249.99, 'pending', 'Europe', NOW() - INTERVAL '1 day'),
(1015, 15, 1, 79.99, 79.99, 'pending', 'North America', NOW());

-- Insert sample reviews
INSERT INTO reviews (product_id, customer_id, rating, title, content, helpful_votes) VALUES
(1, 1001, 5, 'Best headphones ever!', 'The noise cancellation is incredible. I use these for work calls and music, and the quality is exceptional. Battery lasts forever!', 45),
(1, 1002, 4, 'Great sound, minor comfort issue', 'Sound quality is top-notch, but they get a bit uncomfortable after 4+ hours of continuous use.', 23),
(3, 1003, 5, 'Gaming beast!', 'This laptop handles everything I throw at it. Cyberpunk runs at max settings without breaking a sweat. Worth every penny.', 89),
(3, 1004, 5, 'Perfect for game development', 'Using this for Unity and Unreal development. Compiles are lightning fast and the display is gorgeous.', 56),
(9, 1005, 5, 'Life changing kitchen gadget', 'I use my Instant Pot almost every day. From soups to yogurt, it does it all perfectly.', 112),
(9, 1006, 4, 'Great but learning curve', 'Took me a few tries to get the timing right, but now I love it. The slow cooker function is my favorite.', 34),
(14, 1007, 5, 'Marathon approved!', 'Ran my first marathon in these shoes. Zero blisters and great support throughout.', 78),
(14, 1008, 4, 'Good but runs small', 'Great shoes, but order half a size up. The cushioning is perfect for long runs.', 42);
