# NexusERP

A lightweight, modular ERP (Enterprise Resource Planning) platform for
small businesses. NexusERP brings product catalog management, inventory,
purchasing, sales, customers, suppliers, invoicing, and operational
reporting into one application.

> **Project status:** Planning / initial development\
> **Repository:** Add your GitHub repository URL\
> **Live demo:** Add deployment URL when available

NexusERP is designed as a portfolio-quality software engineering
project. The focus is not on reproducing every feature of a large ERP
suite, but on implementing a small set of connected business workflows
correctly, with particular attention to relational data modeling,
transaction safety, concurrency, authorization, testing, and deployment.

------------------------------------------------------------------------

## Table of contents

-   [1. Project goals](#1-project-goals)
-   [2. Scope](#2-scope)
-   [3. Core business workflows](#3-core-business-workflows)
-   [4. Technology stack](#4-technology-stack)
-   [5. Architecture](#5-architecture)
-   [6. Domain model](#6-domain-model)
-   [7. API outline](#7-api-outline)
-   [8. Business rules and
    consistency](#8-business-rules-and-consistency)
-   [9. Repository structure](#9-repository-structure)
-   [10. Milestones and implementation
    plan](#10-milestones-and-implementation-plan)
-   [11. Testing strategy](#11-testing-strategy)
-   [12. Security and reliability](#12-security-and-reliability)
-   [13. Local development](#13-local-development)
-   [14. Definition of done](#14-definition-of-done)
-   [15. Future improvements](#15-future-improvements)
-   [16. CV description](#16-cv-description)

------------------------------------------------------------------------

## 1. Project goals

NexusERP aims to provide a simple operational system for a small
business, such as an electronics shop, that needs to:

1.  Maintain a catalog of products and categories.
2.  Track stock across one or more warehouses.
3.  Purchase products from suppliers and receive deliveries.
4.  Sell products to customers and issue invoices.
5.  Track inventory movements and transaction history.
6.  Control employee access using roles and permissions.
7.  Review business activity through reports and a dashboard.

### Engineering goals

The project will demonstrate the ability to:

-   Design a normalized relational database and meaningful constraints.
-   Build a REST API in Go with clear module boundaries.
-   Write and optimize SQL instead of hiding all database behavior
    behind an ORM.
-   Use PostgreSQL transactions to keep related business operations
    consistent.
-   Handle concurrent requests without overselling inventory.
-   Implement authentication, authorization, validation, and audit
    logging.
-   Write unit and integration tests for critical business workflows.
-   Containerize and deploy an application with repeatable environments.

------------------------------------------------------------------------

## 2. Scope

### MVP modules

  -----------------------------------------------------------------------
  Module                              Responsibilities
  ----------------------------------- -----------------------------------
  Authentication and users            Login, user management, roles,
                                      permissions

  Products and categories             Product catalog, SKU, price,
                                      category

  Inventory                           Warehouses, stock balances, stock
                                      movements, adjustments

  Customers                           Customer profiles and sales history

  Suppliers                           Supplier profiles and purchase
                                      history

  Purchasing                          Purchase orders, receiving, partial
                                      deliveries

  Sales                               Sales orders, order items, order
                                      status, stock deduction

  Invoices                            Invoice creation, totals, status,
                                      printable representation

  Reporting                           Sales summaries, low-stock
                                      products, inventory valuation

  Audit logs                          Trace important changes and the
                                      user responsible
  -----------------------------------------------------------------------

### Explicitly out of scope for the first version

To keep the project achievable, the initial release will not implement:

-   Full accounting or general-ledger functionality.
-   Payroll and human resources.
-   Tax filing or country-specific electronic invoicing integrations.
-   Multi-currency accounting.
-   Manufacturing, bills of materials, or production planning.
-   Complex promotions, loyalty programs, or point-of-sale hardware.
-   Microservices or Kubernetes.
-   AI features.

These may be considered after the core workflows are stable.

------------------------------------------------------------------------

## 3. Core business workflows

### 3.1 Purchasing and receiving stock

1.  An authorized employee creates a purchase order.
2.  The employee selects a supplier and adds products and quantities.
3.  The purchase order is submitted.
4.  When goods arrive, an employee records the received quantities.
5.  The system increases stock for the received quantities.
6.  The system records inventory movements and updates the purchase
    order status.

The system must support partial deliveries. Receiving 6 units from an
order of 10 must leave 4 units outstanding.

### 3.2 Selling products

1.  An employee creates a sales order and selects a customer.
2.  The employee adds products and quantities.
3.  The system checks stock availability.
4.  The order is confirmed.
5.  In one database transaction, the system deducts stock, records
    inventory movements, and creates the invoice.
6.  The order and invoice statuses are updated.

If any critical step fails, the transaction must roll back. The
application must not leave a completed sale with unchanged inventory.

### 3.3 Inventory adjustments

Authorized users can record adjustments for reasons such as damaged
goods, stock counts, or corrections. Every adjustment must create an
inventory movement with a reason and actor. Existing movement records
should not be silently overwritten.

### 3.4 Order cancellation

Cancellation rules must be explicit. For example, cancelling a confirmed
sale may require a compensating stock movement that returns the sold
quantity to inventory. The original movement remains in the history.

------------------------------------------------------------------------

## 4. Technology stack

The initial stack is intentionally small and SQL-oriented.

  -----------------------------------------------------------------------
  Layer                   Technology              Purpose
  ----------------------- ----------------------- -----------------------
  Backend language        Go                      Application and
                                                  business logic

  HTTP router             Chi                     Routing and middleware

  Database                PostgreSQL              Relational business
                                                  data

  PostgreSQL driver       pgx                     Go/PostgreSQL
                                                  connectivity

  Query generation        sqlc                    Type-safe Go code
                                                  generated from SQL

  Schema migrations       golang-migrate or Atlas Versioned database
                                                  schema changes

  Frontend                React + TypeScript      Web interface

  API contract            OpenAPI                 API documentation

  Local infrastructure    Docker Compose          Repeatable local
                                                  environment

  Cache / queue           Redis (later milestone) Optional caching and
                                                  background jobs

  Unit tests              Go testing package      Fast business-logic
                                                  tests

  Integration tests       Testcontainers or       Tests against real
                          Docker-based test DB    PostgreSQL

  CI                      GitHub Actions          Automated checks
  -----------------------------------------------------------------------

### Why this stack?

-   **Go** provides a small language and runtime with explicit error
    handling and built-in concurrency primitives.
-   **Chi** keeps the HTTP layer lightweight without imposing a large
    framework.
-   **PostgreSQL** provides relational integrity, transactions,
    constraints, and row-level locking.
-   **sqlc** keeps SQL visible and generates typed database access code.
-   **pgx** provides a PostgreSQL-native driver and transaction support.

### Important implementation choice

Do not introduce an ORM initially. Use SQL migrations, explicit queries,
`pgx`, and `sqlc`. This makes database behavior easier to inspect and
gives the project a stronger database-engineering focus.

------------------------------------------------------------------------

## 5. Architecture

NexusERP will start as a **modular monolith**.

A modular monolith is one deployable backend application organized into
business modules. Modules have clear responsibilities, but critical
workflows can use a single PostgreSQL transaction. This is simpler to
develop and operate than microservices and is appropriate for the
initial scope.

### High-level architecture

``` text
                    React + TypeScript
                            |
                         HTTPS
                            |
                    Go HTTP API (Chi)
                            |
                 Middleware / Validation
                            |
          +-----------------+-----------------+
          |                 |                 |
       Auth             Products          Customers
          |                 |                 |
       Sales            Inventory          Suppliers
          |                 |                 |
       Purchases         Invoices           Reports
          +-----------------+-----------------+
                            |
                    Application services
                            |
                   Repositories / sqlc
                            |
                       PostgreSQL

              Optional later: Redis / workers
```

### Responsibilities

-   **HTTP handlers:** Parse requests, validate transport-level input,
    call application services, and return HTTP responses.
-   **Services:** Enforce business rules and coordinate workflows.
-   **Repositories / generated queries:** Execute database operations.
-   **Database:** Enforce constraints and provide transaction isolation.
-   **Workers (later):** Process non-critical asynchronous work such as
    notifications and scheduled reports.

### Architectural rules

1.  HTTP handlers must not contain business workflows or raw SQL.
2.  Business rules belong in services or domain-focused functions.
3.  Database constraints must protect important invariants even if
    application validation exists.
4.  Transactions should be created at the service/workflow boundary.
5.  Modules should not reach into another module's internal
    implementation unnecessarily.
6.  Use interfaces where they create a useful test seam or decouple a
    real dependency; avoid creating interfaces for every struct by
    default.

------------------------------------------------------------------------

## 6. Domain model

The following is the initial conceptual model. The exact schema will be
refined during the database-design milestone.

### Main entities

  -----------------------------------------------------------------------
  Entity                              Important fields / relationships
  ----------------------------------- -----------------------------------
  users                               id, name, email, password_hash,
                                      role_id, status

  roles                               id, name

  permissions                         id, key, description

  role_permissions                    role_id, permission_id

  categories                          id, name, description

  products                            id, sku, name, category_id,
                                      unit_price, cost_price, active

  customers                           id, name, email, phone, address

  suppliers                           id, name, email, phone, address

  warehouses                          id, name, location

  inventory_balances                  product_id, warehouse_id, quantity

  inventory_movements                 product_id, warehouse_id,
                                      movement_type, quantity_delta,
                                      reference, created_by, created_at

  purchase_orders                     id, supplier_id, status,
                                      order_date, expected_date

  purchase_order_items                purchase_order_id, product_id,
                                      ordered_quantity,
                                      received_quantity, unit_cost

  sales_orders                        id, customer_id, status,
                                      order_date, total_amount

  sales_order_items                   sales_order_id, product_id,
                                      quantity, unit_price

  invoices                            id, sales_order_id, invoice_number,
                                      status, total_amount, issued_at

  audit_logs                          id, actor_id, action, entity_type,
                                      entity_id, before_data, after_data,
                                      created_at
  -----------------------------------------------------------------------

### Inventory ledger

Inventory should not be represented only by a mutable quantity field on
the product.

Use an append-only inventory movement ledger. Each movement records a
change in stock:

  Event               Product     Quantity delta
  ------------------- --------- ----------------
  Opening stock       Laptop                 +20
  Purchase received   Laptop                 +10
  Sale confirmed      Laptop                  -3
  Damaged stock       Laptop                  -1
  Sale returned       Laptop                  +1

The current quantity for a product in a warehouse is the sum of its
movements. For efficient reads, maintain an `inventory_balances` table
as a transactionally updated projection/cache of that ledger.

The balance and movement record must be updated in the same transaction.
Add reconciliation tests or a maintenance query that compares the
balance with the movement sum.

### Important database constraints and indexes

Plan for constraints such as:

-   Unique product SKU.
-   Unique user email.
-   Unique invoice number.
-   Positive ordered and received quantities.
-   Non-negative stock balance, unless a documented business rule
    permits backorders.
-   Foreign keys for all required relationships.
-   Indexes on foreign keys and frequently filtered fields.
-   Indexes on order status/date, product SKU, and inventory
    product/warehouse keys.

Use appropriate `NUMERIC`/decimal handling for monetary values. Do not
use floating-point types for money. Define the currency and rounding
policy explicitly before implementing invoice totals.

------------------------------------------------------------------------

## 7. API outline

All endpoints will be versioned under `/api/v1`.

The following is an initial API plan, not a claim that these endpoints
already exist.

### Authentication and users

  -----------------------------------------------------------------------
  Method                  Endpoint                Purpose
  ----------------------- ----------------------- -----------------------
  POST                    `/auth/login`           Authenticate a user

  POST                    `/auth/refresh`         Refresh an access
                                                  token, if refresh
                                                  tokens are adopted

  POST                    `/auth/logout`          Revoke or invalidate a
                                                  session

  GET                     `/users/me`             Get current user

  GET                     `/users`                List users (permission
                                                  protected)

  POST                    `/users`                Create user

  PATCH                   `/users/{id}`           Update user
  -----------------------------------------------------------------------

### Products and categories

  Method   Endpoint           Purpose
  -------- ------------------ ----------------------
  GET      `/products`        List/search products
  POST     `/products`        Create product
  GET      `/products/{id}`   Get product
  PATCH    `/products/{id}`   Update product
  GET      `/categories`      List categories
  POST     `/categories`      Create category

### Inventory

  Method   Endpoint                   Purpose
  -------- -------------------------- ---------------------------
  GET      `/inventory`               Query stock balances
  GET      `/inventory/movements`     View movement history
  POST     `/inventory/adjustments`   Record a stock adjustment
  GET      `/warehouses`              List warehouses
  POST     `/warehouses`              Create warehouse

### Customers and suppliers

  Method        Endpoint            Purpose
  ------------- ------------------- -----------------------------
  GET / POST    `/customers`        List or create customers
  GET / PATCH   `/customers/{id}`   Retrieve or update customer
  GET / POST    `/suppliers`        List or create suppliers
  GET / PATCH   `/suppliers/{id}`   Retrieve or update supplier

### Purchasing

  Method       Endpoint                           Purpose
  ------------ ---------------------------------- --------------------------------
  GET / POST   `/purchase-orders`                 List or create purchase orders
  GET          `/purchase-orders/{id}`            Get purchase order
  POST         `/purchase-orders/{id}/submit`     Submit purchase order
  POST         `/purchase-orders/{id}/receipts`   Record received quantities

### Sales and invoices

  Method       Endpoint                       Purpose
  ------------ ------------------------------ ---------------------------------
  GET / POST   `/sales-orders`                List or create sales orders
  GET          `/sales-orders/{id}`           Get sales order
  POST         `/sales-orders/{id}/confirm`   Confirm sale and deduct stock
  POST         `/sales-orders/{id}/cancel`    Cancel sale under defined rules
  GET          `/invoices`                    List invoices
  GET          `/invoices/{id}`               Get invoice

### Reports

  Method   Endpoint                         Purpose
  -------- -------------------------------- ----------------------------------
  GET      `/reports/sales-summary`         Sales totals over a date range
  GET      `/reports/top-products`          Best-selling products
  GET      `/reports/low-stock`             Products below reorder threshold
  GET      `/reports/inventory-valuation`   Inventory value summary

Use pagination, filtering, sorting, consistent error responses, and
request validation for collection endpoints.

------------------------------------------------------------------------

## 8. Business rules and consistency

This section is central to the quality of the project.

### 8.1 Safe stock deduction

A sale must not allow two concurrent requests to sell the same final
unit.

Use a database transaction and a concurrency-control strategy such as:

-   Lock the relevant inventory balance row with
    `SELECT ... FOR UPDATE`, then validate and update it; or
-   Use a conditional atomic update that decrements stock only when
    sufficient quantity remains, and verify the affected row count.

Choose one approach, document it, and test it with concurrent requests.

### 8.2 Atomic sales confirmation

The following operations should succeed or fail together:

1.  Validate order status.
2.  Validate and deduct stock.
3.  Insert inventory movement records.
4.  Create the invoice.
5.  Update the sales order status.

If any operation fails, roll back the transaction.

### 8.3 Partial purchase receipts

Track ordered and received quantities separately. A purchase order is
fully received only when all items have been received. Receiving the
same delivery twice must not accidentally increase stock twice.

### 8.4 Idempotency

For operations that may be retried by a client, especially order
confirmation and receipt creation, consider an idempotency key or
another duplicate-request protection mechanism. A retry must not create
duplicate stock movements or invoices.

### 8.5 Auditability

Record important actions with the actor, timestamp, action, affected
entity, and relevant before/after values. Avoid storing passwords,
tokens, or other secrets in audit records.

------------------------------------------------------------------------

## 9. Repository structure

Suggested monorepo structure:

``` text
nexuserp/
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go
│   ├── internal/
│   │   ├── auth/
│   │   ├── users/
│   │   ├── products/
│   │   ├── categories/
│   │   ├── inventory/
│   │   ├── customers/
│   │   ├── suppliers/
│   │   ├── warehouses/
│   │   ├── purchasing/
│   │   ├── sales/
│   │   ├── invoices/
│   │   ├── reports/
│   │   └── audit/
│   ├── db/
│   │   ├── migrations/
│   │   ├── queries/
│   │   └── sqlc/
│   ├── config/
│   ├── tests/
│   ├── Dockerfile
│   ├── go.mod
│   └── sqlc.yaml
├── frontend/
│   ├── src/
│   ├── public/
│   └── package.json
├── deployments/
│   └── docker-compose.yml
├── docs/
│   ├── architecture/
│   ├── database/
│   └── api/
├── .github/
│   └── workflows/
├── .env.example
└── README.md
```

Keep generated sqlc code separate from handwritten SQL queries. Do not
edit generated files directly.

------------------------------------------------------------------------

## 10. Milestones and implementation plan

The milestones are ordered to reduce rework: define the data model
first, build one complete business workflow, then expand the system.

A reasonable target is **6--8 weeks of part-time development**,
depending on prior Go experience and the amount of frontend work. The
estimate is a planning guide, not a deadline.

### Milestone 0 --- Requirements and design

**Goal:** Agree on the business rules and design before writing
application code.

Tasks:

-   [ ] Define the example business and its workflows.
-   [ ] Finalize MVP scope and non-goals.
-   [ ] Draw the ER diagram.
-   [ ] Define entities, relationships, constraints, and indexes.
-   [ ] Define product, order, invoice, and purchase-order statuses.
-   [ ] Decide how stock reservations, sales cancellation, and returns
    work.
-   [ ] Define money precision, currency, and rounding policy.
-   [ ] Draft API routes and request/response examples.
-   [ ] Record important decisions in short architecture decision
    records (ADRs).

**Deliverables:** Requirements, ERD, workflow diagrams, initial API
specification, and a prioritized backlog.

**Acceptance criteria:** The sales and purchasing workflows can be
explained end-to-end, including failure and cancellation cases.

### Milestone 1 --- Go backend foundation

**Goal:** Create a runnable backend with a reliable development
environment.

Tasks:

-   [ ] Install and configure Go.
-   [ ] Create the Go module.
-   [ ] Add Chi and pgx.
-   [ ] Configure PostgreSQL through Docker Compose.
-   [ ] Add environment-based configuration.
-   [ ] Add database connection handling and graceful shutdown.
-   [ ] Add structured logging.
-   [ ] Add health and readiness endpoints.
-   [ ] Set up database migrations.
-   [ ] Set up sqlc and generate code from a sample query.
-   [ ] Add formatting, linting, and basic CI checks.

**Deliverables:** A running Go API, PostgreSQL service, migrations, and
automated build checks.

**Acceptance criteria:** A developer can clone the repository, configure
environment variables, start the services, apply migrations, and call
the health endpoint using documented commands.

### Milestone 2 --- Database and identity

**Goal:** Establish the core schema and protect API access.

Tasks:

-   [ ] Implement users, roles, and permissions.
-   [ ] Implement password hashing using a suitable password-hashing
    algorithm.
-   [ ] Implement login and logout/session behavior.
-   [ ] Add authentication middleware.
-   [ ] Add authorization checks for protected operations.
-   [ ] Add request validation and a consistent error format.
-   [ ] Add audit-log infrastructure.
-   [ ] Write tests for authentication and permission boundaries.

**Deliverables:** Initial schema, authentication endpoints,
authorization middleware, and tests.

**Acceptance criteria:** Unauthenticated users cannot access protected
endpoints, and users cannot perform actions beyond their assigned
permissions.

### Milestone 3 --- Product catalog and master data

**Goal:** Implement the data that business operations depend on.

Tasks:

-   [ ] Create categories.
-   [ ] Create, retrieve, update, and deactivate products.
-   [ ] Enforce unique SKUs.
-   [ ] Store monetary values using decimal-safe database types.
-   [ ] Implement customer management.
-   [ ] Implement supplier management.
-   [ ] Add pagination, filtering, and search.
-   [ ] Add validation and integration tests.

**Deliverables:** Working product, category, customer, and supplier
APIs.

**Acceptance criteria:** Records have valid relationships, duplicate
SKUs are rejected, and list endpoints support predictable pagination and
filtering.

### Milestone 4 --- Inventory management

**Goal:** Build reliable stock tracking.

Tasks:

-   [ ] Implement warehouses.
-   [ ] Implement inventory balances.
-   [ ] Implement the inventory movement ledger.
-   [ ] Add opening-stock functionality.
-   [ ] Add authorized stock adjustments with reasons.
-   [ ] Add stock lookup by product and warehouse.
-   [ ] Add movement history and filtering.
-   [ ] Implement a reconciliation query/test comparing balances with
    movement totals.

**Deliverables:** Inventory APIs, movement history, balance updates, and
reconciliation checks.

**Acceptance criteria:** Every stock change creates a corresponding
movement, and the balance projection remains consistent with the ledger.

### Milestone 5 --- Purchasing workflow

**Goal:** Make it possible to replenish inventory from suppliers.

Tasks:

-   [ ] Create purchase orders and purchase-order items.
-   [ ] Implement draft and submitted states.
-   [ ] Validate supplier and product references.
-   [ ] Record full and partial receipts.
-   [ ] Update inventory balances and movements transactionally.
-   [ ] Prevent duplicate receipt processing.
-   [ ] Track outstanding quantities and purchase status.
-   [ ] Add tests for partial receipt and transaction rollback.

**Deliverables:** Complete purchase-order and receiving workflow.

**Acceptance criteria:** Receiving stock increases inventory exactly
once, supports partial deliveries, and preserves a traceable purchase
history.

### Milestone 6 --- Sales and invoicing workflow

**Goal:** Implement the most important end-to-end business operation.

Tasks:

-   [ ] Create sales orders and order items.
-   [ ] Validate prices and quantities on the server.
-   [ ] Check stock availability.
-   [ ] Confirm sales inside a database transaction.
-   [ ] Prevent overselling under concurrent requests.
-   [ ] Create invoice records.
-   [ ] Implement cancellation and compensating stock movements.
-   [ ] Add duplicate-request protection where appropriate.
-   [ ] Test rollback when any critical step fails.

**Deliverables:** Sales APIs, invoice records, and transaction-safe
order confirmation.

**Acceptance criteria:** A successful sale creates the correct invoice
and stock movements. A failed sale leaves no partial business changes.
Concurrent sales cannot sell more stock than is available.

### Milestone 7 --- Frontend and reporting

**Goal:** Make the system usable and expose useful business information.

Tasks:

-   [ ] Set up React and TypeScript.
-   [ ] Implement login and protected routes.
-   [ ] Build a dashboard layout and navigation.
-   [ ] Build product, customer, and supplier screens.
-   [ ] Build inventory and movement-history screens.
-   [ ] Build purchase-order and receiving screens.
-   [ ] Build sales-order and invoice screens.
-   [ ] Add loading, empty, validation, and error states.
-   [ ] Add sales summary and top-product reports.
-   [ ] Add low-stock and inventory-valuation reports.

**Deliverables:** Usable web interface and first reporting dashboard.

**Acceptance criteria:** A user can complete the core purchasing and
sales workflows through the UI without manually editing the database.

### Milestone 8 --- Quality, deployment, and documentation

**Goal:** Make the project reproducible, testable, and demonstrable.

Tasks:

-   [ ] Expand unit and integration test coverage around critical
    business rules.
-   [ ] Add concurrent-sale tests.
-   [ ] Add database migration checks to CI.
-   [ ] Add API documentation.
-   [ ] Add structured request logging and request IDs.
-   [ ] Review authorization and secret handling.
-   [ ] Add Dockerfiles and production-oriented Compose configuration
    where appropriate.
-   [ ] Deploy the application and database securely.
-   [ ] Add sample/demo data.
-   [ ] Capture screenshots and record a short demo.
-   [ ] Complete architecture, setup, and API documentation.
-   [ ] Add known limitations and future improvements to the README.

**Deliverables:** Deployed demo, automated CI, test suite, screenshots,
and complete documentation.

**Acceptance criteria:** Another developer can follow the README to run
the application, understand the architecture, execute tests, and try the
main workflows.

### Suggested milestone schedule

  -----------------------------------------------------------------------
  Week                    Focus                   Main result
  ----------------------- ----------------------- -----------------------
  1                       Design and Go           ERD, API plan, runnable
                          foundation              backend

  2                       Database, auth, master  Users, products,
                          data                    customers, suppliers

  3                       Inventory               Ledger and stock
                                                  balances

  4                       Purchasing              Purchase orders and
                                                  receiving

  5                       Sales and invoices      Transaction-safe sales
                                                  workflow

  6                       Frontend and reports    Usable application

  7                       Testing and hardening   Reliable workflows and
                                                  CI

  8                       Deployment and          Public demo and
                          documentation           portfolio-ready repo
  -----------------------------------------------------------------------

If time is limited, prioritize milestones 0--6 and a simple frontend. Do
not sacrifice transaction correctness just to add more screens.

------------------------------------------------------------------------

## 11. Testing strategy

Tests should focus on business correctness, not only whether endpoints
return HTTP 200.

### Unit tests

Test business rules in isolation:

-   Invalid product quantities are rejected.
-   Invalid order transitions are rejected.
-   Invoice totals and line totals are calculated correctly.
-   Users without the required permission are denied.
-   Purchase receipt quantities cannot exceed outstanding quantities.

### Integration tests

Run tests against a real PostgreSQL instance:

-   Product creation and SKU uniqueness.
-   Inventory balance and movement updates.
-   Purchase receipt transaction behavior.
-   Sales confirmation and invoice creation.
-   Transaction rollback after an injected failure.
-   Order cancellation and compensating stock movements.

### Concurrency tests

Simulate multiple employees attempting to sell the last available unit.
Verify that no more than the available quantity can be sold and that the
database remains consistent.

### Useful commands

``` bash
# Run all Go tests
go test ./...

# Run tests with race detection
go test -race ./...

# Format Go files
gofmt -w .

# Run static analysis
go vet ./...
```

Add linting and integration-test commands to the repository once the
relevant tools and configuration are selected.

------------------------------------------------------------------------

## 12. Security and reliability

Minimum expectations:

-   Hash passwords; never store plaintext passwords.
-   Keep secrets in environment variables or a secrets manager.
-   Do not commit `.env` files containing real credentials.
-   Validate all incoming data on the server.
-   Enforce authorization on the backend, not only in the frontend.
-   Use parameterized SQL queries.
-   Set sensible request and database timeouts.
-   Use HTTPS in deployed environments.
-   Avoid exposing internal error details to clients.
-   Log important business actions without logging secrets.
-   Use database constraints as a final layer of data protection.
-   Back up production data and document recovery expectations.

For token-based authentication, choose access-token expiry,
refresh-token rotation/revocation, and secure cookie or client-storage
behavior deliberately. Do not assume that JWT alone provides session
revocation.

------------------------------------------------------------------------

## 13. Local development

The exact commands and environment variables will be finalized during
Milestone 1. The intended developer experience is:

``` bash
# Clone the repository
git clone <repository-url>
cd nexuserp

# Configure local environment
cp .env.example .env

# Start infrastructure
docker compose -f deployments/docker-compose.yml up -d

# Run database migrations
# Add the selected migration command here.

# Start the backend
cd backend
go run ./cmd/api
```

Example environment variables:

``` dotenv
APP_ENV=development
HTTP_ADDR=:8080

DATABASE_URL=postgres://nexuserp:nexuserp@localhost:5432/nexuserp?sslmode=disable

# Use a strong, locally generated secret.
AUTH_SECRET=replace-with-a-local-development-secret
```

These values are examples for local development only. Never use the
example credentials or secrets in production.

### API documentation

The API should expose an OpenAPI specification and a convenient local
documentation page once the API contract is implemented.

------------------------------------------------------------------------

## 14. Definition of done

The first portfolio-ready release is complete when:

-   [ ] A user can authenticate and access only permitted functionality.
-   [ ] Products, customers, and suppliers can be managed.
-   [ ] Stock can be received, adjusted, and inspected.
-   [ ] Purchase orders support partial receipts.
-   [ ] Sales orders can be confirmed and invoiced.
-   [ ] Sales and inventory updates are transactionally consistent.
-   [ ] Concurrent sales cannot oversell available stock.
-   [ ] Important actions are recorded in audit logs.
-   [ ] Reports return correct results for seeded data.
-   [ ] Critical business workflows have automated tests.
-   [ ] The application runs through documented Docker-based setup.
-   [ ] CI runs formatting, static checks, and tests.
-   [ ] The application is deployed or has a reproducible demo
    environment.
-   [ ] README, API documentation, architecture diagram, and screenshots
    are available.

------------------------------------------------------------------------

## 15. Future improvements

Only consider these after the MVP is stable:

-   Multi-warehouse stock transfers.
-   Product variants and barcodes.
-   Returns and refunds.
-   Stock reservations and backorders.
-   Reorder-point suggestions.
-   Redis-backed background jobs and notifications.
-   CSV import/export.
-   Advanced reporting and scheduled report delivery.
-   Multi-tenant support for multiple businesses.
-   Observability with metrics and distributed tracing.
-   AI-assisted demand forecasting or natural-language reporting.

AI is intentionally a later extension. The first objective is a
dependable ERP with correct business workflows.

------------------------------------------------------------------------

## 16. CV description

Use a description like the following after the corresponding features
are implemented:

**NexusERP --- Modular Business Management Platform**

-   Designed and developed a modular ERP platform using Go, PostgreSQL,
    and React to manage inventory, purchasing, sales, customers, and
    suppliers.
-   Implemented transaction-safe inventory and sales workflows using
    PostgreSQL transactions and row-level locking to prevent
    overselling.
-   Built a SQL-first data-access layer using pgx and sqlc, with
    database migrations, constraints, and integration tests.
-   Implemented authentication, role-based access control, audit
    logging, and automated CI checks.
-   Containerized the application with Docker and documented its
    architecture and deployment.

Only include technologies and features that are genuinely implemented
and demonstrable.

------------------------------------------------------------------------

## Project principle

**Build fewer features, but make the important workflows correct.**

A small ERP that reliably handles purchasing, inventory, and
sales---with sound database design, concurrency control, tests, and
clear documentation---is a stronger engineering portfolio project than a
much larger collection of unfinished CRUD modules.
