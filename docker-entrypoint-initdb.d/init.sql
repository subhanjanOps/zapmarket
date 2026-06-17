CREATE DATABASE userauth;
CREATE DATABASE apigateway;
CREATE DATABASE ordermgmt;
CREATE DATABASE inventory;
CREATE DATABASE payment;
CREATE DATABASE notification;
CREATE DATABASE productcatalog;

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
