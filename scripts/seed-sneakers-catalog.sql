-- product_db | psql "postgresql://postgres:postgres@localhost:5434/product_db?sslmode=disable" -f scripts/seed-sneakers-catalog.sql
-- Photo URLs: use https://images.unsplash.com/photo-... (stable). https://source.unsplash.com/... returns 503 — Unsplash Source API is discontinued.

BEGIN;

DELETE FROM inventory WHERE product_id LIKE 'snk_demo_%';
DELETE FROM products WHERE id LIKE 'snk_demo_%';

INSERT INTO products (id, name, photo_url, description, brand, price, currency, category) VALUES
('snk_demo_001', 'Nike Air Zoom Pegasus 41', 'https://images.unsplash.com/photo-1542291026-7eec264c27ff?auto=format&fit=crop&w=1200&q=85', 'Neutral daily trainer. Responsive foam, breathable mesh upper. Road running.', 'Nike', 139.99, 'USD', 'sneakers'),
('snk_demo_002', 'adidas Ultraboost Light', 'https://images.unsplash.com/photo-1587563871167-1ee9c731aefb?auto=format&fit=crop&w=1200&q=85', 'Lifestyle and easy miles. Boost midsole, Continental rubber outsole.', 'adidas', 189.99, 'USD', 'sneakers'),
('snk_demo_003', 'New Balance 990v6 Made in USA', 'https://images.unsplash.com/photo-1539185441755-769473a23570?auto=format&fit=crop&w=1200&q=85', 'Heritage grey dad shoe. ENCAP midsole, premium suede and mesh.', 'New Balance', 209.99, 'USD', 'sneakers'),
('snk_demo_004', 'ASICS Gel-Kayano 31', 'https://images.unsplash.com/photo-1552346154-21d32810aba3?auto=format&fit=crop&w=1200&q=85', 'Stability trainer for overpronators. GEL cushioning, engineered knit upper.', 'ASICS', 164.99, 'USD', 'sneakers'),
('snk_demo_005', 'Brooks Ghost 16', 'https://images.unsplash.com/photo-1606107557195-0e29a4b5b4aa?auto=format&fit=crop&w=1200&q=85', 'Soft neutral road shoe. DNA LOFT v3 cushioning, smooth transitions.', 'Brooks', 139.95, 'USD', 'sneakers'),
('snk_demo_006', 'Hoka Clifton 9', 'https://images.unsplash.com/photo-1514989940723-e8e51635b782?auto=format&fit=crop&w=1200&q=85', 'Lightweight cushioned daily trainer. Meta-Rocker geometry.', 'Hoka', 144.99, 'USD', 'sneakers'),
('snk_demo_007', 'On Cloudmonster 2', 'https://images.unsplash.com/photo-1460353581641-37baddab0fa2?auto=format&fit=crop&w=1200&q=85', 'Max cushioning road shoe. Helion superfoam.', 'On', 179.99, 'USD', 'sneakers'),
('snk_demo_008', 'Saucony Endorphin Speed 4', 'https://images.unsplash.com/photo-1595950653106-6c9ebd614d3a?auto=format&fit=crop&w=1200&q=85', 'Tempo trainer. PWRRUN PB foam, SPEEDROLL tech.', 'Saucony', 169.95, 'USD', 'sneakers'),
('snk_demo_009', 'Mizuno Wave Rider 28', 'https://images.unsplash.com/photo-1515955656352-a1fa3ffcd111?auto=format&fit=crop&w=1200&q=85', 'Snappy neutral trainer. Wave Plate technology.', 'Mizuno', 139.99, 'USD', 'sneakers'),
('snk_demo_010', 'Puma Velocity Nitro 3', 'https://images.unsplash.com/photo-1511556532299-8f662fc26c06?auto=format&fit=crop&w=1200&q=85', 'Versatile road shoe. Nitrofoam midsole.', 'Puma', 129.99, 'USD', 'sneakers'),
('snk_demo_011', 'Nike Vaporfly 3', 'https://images.unsplash.com/photo-1605348532760-6753d2c43329?auto=format&fit=crop&w=1200&q=85', 'Marathon racing shoe. ZoomX foam, carbon plate.', 'Nike', 249.99, 'USD', 'sneakers'),
('snk_demo_012', 'adidas Adizero Adios Pro 3', 'https://images.unsplash.com/photo-1521412644187-c49fa049e84d?auto=format&fit=crop&w=1200&q=85', 'Carbon rods racing shoe. Lightstrike Pro.', 'adidas', 219.99, 'USD', 'sneakers'),
('snk_demo_013', 'Jordan 1 Retro High OG', 'https://images.unsplash.com/photo-1551107696-a4b0c5a0d9a2?auto=format&fit=crop&w=1200&q=85', 'Classic basketball sneaker. Leather upper.', 'Jordan', 179.99, 'USD', 'sneakers'),
('snk_demo_014', 'Converse Chuck 70 High', 'https://images.unsplash.com/photo-1607522370275-f14206abe5d3?auto=format&fit=crop&w=1200&q=85', 'Vintage canvas high-top.', 'Converse', 89.99, 'USD', 'sneakers'),
('snk_demo_015', 'Vans Old Skool', 'https://images.unsplash.com/photo-1525966222134-fcfa99b8ae77?auto=format&fit=crop&w=1200&q=85', 'Skate classic shoe.', 'Vans', 74.99, 'USD', 'sneakers'),
('snk_demo_016', 'Salomon XT-6', 'https://images.unsplash.com/photo-1536766768598-e09213fdcf22?auto=format&fit=crop&w=1200&q=85', 'Trail-meets-street sneaker.', 'Salomon', 189.99, 'USD', 'sneakers'),
('snk_demo_017', 'Merrell Moab Speed 2', 'https://images.unsplash.com/photo-1595341888016-a392ef81b7de?auto=format&fit=crop&w=1200&q=85', 'Hiking crossover shoe.', 'Merrell', 129.99, 'USD', 'sneakers'),
('snk_demo_018', 'Nike Dunk Low Retro', 'https://images.unsplash.com/photo-1516478177764-9fe5bd7e9717?auto=format&fit=crop&w=1200&q=85', 'Court-inspired low profile.', 'Nike', 114.99, 'USD', 'sneakers'),
('snk_demo_019', 'adidas Samba OG', 'https://images.unsplash.com/photo-1518002171953-a080ee817e1f?auto=format&fit=crop&w=1200&q=85', 'Indoor football heritage.', 'adidas', 109.99, 'USD', 'sneakers'),
('snk_demo_020', 'New Balance 2002R', 'https://images.unsplash.com/photo-1504198458649-3128b932f49e?auto=format&fit=crop&w=1200&q=85', 'Chunky lifestyle runner.', 'New Balance', 149.99, 'USD', 'sneakers'),
('snk_demo_021', 'Reebok Club C 85 Vintage', 'https://images.unsplash.com/photo-1612902376491-7a8a99b424e8?auto=format&fit=crop&w=1200&q=85', 'Tennis heritage leather shoe.', 'Reebok', 84.99, 'USD', 'sneakers'),
('snk_demo_022', 'Nike Air Max 90', 'https://images.unsplash.com/photo-1549298916-b41d501d3772?auto=format&fit=crop&w=1200&q=85', 'Visible Air unit lifestyle sneaker.', 'Nike', 129.99, 'USD', 'sneakers'),
('snk_demo_023', 'ASICS Gel-1130', 'https://images.unsplash.com/photo-1441986300917-64674bd600d8?auto=format&fit=crop&w=1200&q=85', 'Y2K runner aesthetic.', 'ASICS', 99.99, 'USD', 'sneakers'),
('snk_demo_024', 'Lacoste Powercourt', 'https://images.unsplash.com/photo-1445205170230-053b83016050?auto=format&fit=crop&w=1200&q=85', 'Minimal court sneaker.', 'Lacoste', 94.99, 'USD', 'sneakers');

INSERT INTO inventory (id, product_id, quantity, reserved) VALUES
('inv_snk_demo_001', 'snk_demo_001', 40, 0),
('inv_snk_demo_002', 'snk_demo_002', 35, 0),
('inv_snk_demo_003', 'snk_demo_003', 22, 0),
('inv_snk_demo_004', 'snk_demo_004', 30, 0),
('inv_snk_demo_005', 'snk_demo_005', 45, 0),
('inv_snk_demo_006', 'snk_demo_006', 38, 0),
('inv_snk_demo_007', 'snk_demo_007', 25, 0),
('inv_snk_demo_008', 'snk_demo_008', 20, 0),
('inv_snk_demo_009', 'snk_demo_009', 33, 0),
('inv_snk_demo_010', 'snk_demo_010', 42, 0),
('inv_snk_demo_011', 'snk_demo_011', 15, 0),
('inv_snk_demo_012', 'snk_demo_012', 18, 0),
('inv_snk_demo_013', 'snk_demo_013', 28, 0),
('inv_snk_demo_014', 'snk_demo_014', 60, 0),
('inv_snk_demo_015', 'snk_demo_015', 55, 0),
('inv_snk_demo_016', 'snk_demo_016', 24, 0),
('inv_snk_demo_017', 'snk_demo_017', 31, 0),
('inv_snk_demo_018', 'snk_demo_018', 50, 0),
('inv_snk_demo_019', 'snk_demo_019', 48, 0),
('inv_snk_demo_020', 'snk_demo_020', 27, 0),
('inv_snk_demo_021', 'snk_demo_021', 65, 0),
('inv_snk_demo_022', 'snk_demo_022', 44, 0),
('inv_snk_demo_023', 'snk_demo_023', 36, 0),
('inv_snk_demo_024', 'snk_demo_024', 40, 0);

COMMIT;
