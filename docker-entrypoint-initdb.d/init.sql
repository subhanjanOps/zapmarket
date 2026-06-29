CREATE DATABASE userauth;
CREATE DATABASE cart;
CREATE DATABASE settlement;
CREATE DATABASE reviews;
CREATE DATABASE promotions;
CREATE DATABASE logistics;
CREATE DATABASE wishlist;
CREATE DATABASE apigateway;
CREATE DATABASE ordermgmt;
CREATE DATABASE inventory;
CREATE DATABASE payment;
CREATE DATABASE notification;
CREATE DATABASE productcatalog;
CREATE DATABASE currency;

-- userauth and productcatalog schemas are now owned by golang-migrate:
-- see services/auth-service/migrations and
-- services/product-catalog-service/migrations. Only database creation
-- (which migrate cannot do) stays here for those two.
\connect userauth
CREATE EXTENSION IF NOT EXISTS pgcrypto;

\connect productcatalog
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- inventory schema is now owned by golang-migrate:
-- see services/inventory-service/migrations.
\connect inventory
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- payment schema is now owned by golang-migrate:
-- see services/payment-service/migrations.
\connect payment
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ordermgmt schema is owned by golang-migrate:
-- see services/order-management-service/migrations.
\connect ordermgmt
CREATE EXTENSION IF NOT EXISTS pgcrypto;

\connect notification
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Notification service is stateless; no primary tables are created here.

-- apigateway schema is owned by golang-migrate:
-- see services/api-gateway/migrations.
\connect apigateway
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- cart schema is owned by golang-migrate:
-- see services/cart-service/migrations.
\connect cart
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- settlement schema is owned by golang-migrate:
-- see services/settlement-service/migrations.
\connect settlement
CREATE EXTENSION IF NOT EXISTS pgcrypto;

\connect reviews
CREATE EXTENSION IF NOT EXISTS pgcrypto;

\connect promotions
CREATE EXTENSION IF NOT EXISTS pgcrypto;

\connect logistics
CREATE EXTENSION IF NOT EXISTS pgcrypto;

\connect wishlist
CREATE EXTENSION IF NOT EXISTS pgcrypto;
