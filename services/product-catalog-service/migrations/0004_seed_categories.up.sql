-- Seed common marketplace categories so the seller-ui category picker
-- works out of the box without a manual backoffice setup step.
INSERT INTO categories (name, slug) VALUES
    ('Electronics',        'electronics'),
    ('Fashion',            'fashion'),
    ('Home & Garden',      'home-garden'),
    ('Sports & Outdoors',  'sports-outdoors'),
    ('Books & Media',      'books-media'),
    ('Toys & Games',       'toys-games'),
    ('Health & Beauty',    'health-beauty'),
    ('Automotive',         'automotive'),
    ('Food & Grocery',     'food-grocery'),
    ('Office & Stationery','office-stationery')
ON CONFLICT (slug) DO NOTHING;

-- Subcategories for Electronics
INSERT INTO categories (name, slug, parent_id)
SELECT 'Smartphones',   'smartphones',    id FROM categories WHERE slug = 'electronics'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (name, slug, parent_id)
SELECT 'Laptops',       'laptops',        id FROM categories WHERE slug = 'electronics'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (name, slug, parent_id)
SELECT 'Audio',         'audio',          id FROM categories WHERE slug = 'electronics'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (name, slug, parent_id)
SELECT 'Cameras',       'cameras',        id FROM categories WHERE slug = 'electronics'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (name, slug, parent_id)
SELECT 'Accessories',   'electronics-accessories', id FROM categories WHERE slug = 'electronics'
ON CONFLICT (slug) DO NOTHING;

-- Subcategories for Fashion
INSERT INTO categories (name, slug, parent_id)
SELECT 'Men''s Clothing', 'mens-clothing',   id FROM categories WHERE slug = 'fashion'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (name, slug, parent_id)
SELECT 'Women''s Clothing','womens-clothing', id FROM categories WHERE slug = 'fashion'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (name, slug, parent_id)
SELECT 'Footwear',         'footwear',        id FROM categories WHERE slug = 'fashion'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (name, slug, parent_id)
SELECT 'Bags & Wallets',   'bags-wallets',    id FROM categories WHERE slug = 'fashion'
ON CONFLICT (slug) DO NOTHING;

-- Subcategories for Home & Garden
INSERT INTO categories (name, slug, parent_id)
SELECT 'Furniture',      'furniture',       id FROM categories WHERE slug = 'home-garden'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (name, slug, parent_id)
SELECT 'Kitchen & Dining','kitchen-dining', id FROM categories WHERE slug = 'home-garden'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (name, slug, parent_id)
SELECT 'Bedding',         'bedding',        id FROM categories WHERE slug = 'home-garden'
ON CONFLICT (slug) DO NOTHING;
INSERT INTO categories (name, slug, parent_id)
SELECT 'Garden Tools',    'garden-tools',   id FROM categories WHERE slug = 'home-garden'
ON CONFLICT (slug) DO NOTHING;
