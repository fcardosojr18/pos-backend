INSERT INTO menu_items (category, name, price_cents) VALUES
('Burgers','Cheeseburger',1299),
('Burgers','Veggie Burger',1199),
('Sides','Fries',399),
('Drinks','Cola',199)
ON CONFLICT DO NOTHING;
