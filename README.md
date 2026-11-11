# NexusERP

A multi-tenant SaaS ERP for small and medium-sized businesses. NexusERP brings product catalog management, inventory, purchasing, sales, customers, suppliers, invoicing, and operational reporting into one application, while allowing many independent companies to use the same platform without seeing or modifying one another's data.

> **Project status:** Planning / initial development  
> **Repository:** Add your GitHub repository URL  
> **Live demo:** Add deployment URL when available

NexusERP is a portfolio-quality software engineering project. Its focus is not to reproduce every feature of a large ERP suite, but to implement connected business workflows correctly and demonstrate practical SaaS engineering: tenant isolation, relational data modeling, transaction safety, concurrency, observability, testing, and deployment.

## Table of contents

- [1. Product goals and assumptions](#1-product-goals-and-assumptions)
- [2. MVP scope](#2-mvp-scope)
- [3. Core business workflows](#3-core-business-workflows)
- [4. Technology stack](#4-technology-stack)
- [5. Architecture](#5-architecture)
- [6. Multi-tenant database design](#6-multi-tenant-database-design)
- [7. API design](#7-api-design)
- [8. Business consistency and concurrency](#8-business-consistency-and-concurrency)
- [9. Capacity and performance plan](#9-capacity-and-performance-plan)
- [10. Infrastructure and deployment](#10-infrastructure-and-deployment)
- [11. Repository structure](#11-repository-structure)
- [12. Milestones and implementation plan](#12-milestones-and-implementation-plan)
- [13. Testing and load validation](#13-testing-and-load-validation)
- [14. Security, isolation, and reliability](#14-security-isolation-and-reliability)
- [15. Local development](#15-local-development)
- [16. Definition of done](#16-definition-of-done)
- [17. Future improvements](#17-future-improvements)
- [18. CV description](#18-cv-description)

---

## 1. Product goals and assumptions

### Product model

NexusERP is a **shared, multi-tenant SaaS application**. Many companies (tenants) use the same deployed application and PostgreSQL cluster. Each company has its own users, roles, products, customers, suppliers, warehouses, orders, invoices, and reports. A user may belong to more than one company and must explicitly operate within one active company context at a time.

The initial tenancy strategy is **shared database, shared schema, tenant discriminator**: business tables contain a `tenant_id`, and every read and write is scoped to it. This is cost-effective and operationally manageable for the target scale. It is not database-per-customer at the start.

### Capacity target

The initial design target is:

| Measure | Initial target | Meaning |
|---|---:|---|
| Registered users | 10,000+ | Accounts across all tenants; not all are online at once |
| Simultaneously active users | 1,000 | Users with an active application session and making requests during a usage period |
| Sustained API throughput | 200 requests/second | Initial system-wide load-test target |
| Short burst throughput | 500 requests/second | Brief bursts, subject to endpoint mix and latency |
| Read latency | p95 below 300 ms | Ordinary indexed reads under the target test mix |
| Write latency | p95 below 500 ms | Ordinary business writes, excluding long reports and external integrations |
| Availability objective | 99.5% initially | A planning objective, not a measured guarantee or formal SLA |

These are **engineering targets to validate**, not a promise that a particular machine will achieve them. “1,000 active users” is not the same as 1,000 simultaneous database queries. Actual capacity depends on how frequently users act, endpoint complexity, tenant data size, query plans, hardware, and workload distribution. Load tests must model realistic think time and a mix of reads and writes.

### Engineering goals

- Build a modular Go backend with explicit module boundaries.
- Enforce tenant isolation in application code and at the database boundary.
- Model business data with tenant-aware constraints and indexes.
- Use PostgreSQL transactions for critical workflows.
- Prevent overselling and duplicate processing under concurrent requests.
- Keep API instances stateless so they can scale horizontally.
- Measure performance and capacity instead of guessing.
- Provide automated tests, CI, deployment documentation, and observability.

## 2. MVP scope

### Core modules

| Module | Responsibilities |
|---|---|
| Platform identity | Sign-up/invitations, authentication, sessions, users, tenant memberships |
| Tenant administration | Company profile, settings, membership management, tenant-scoped roles and permissions |
| Products and categories | Product catalog, SKU, prices, categories |
| Inventory | Warehouses, stock balances, immutable stock movements, adjustments |
| Customers | Customer profiles and sales history |
| Suppliers | Supplier profiles and purchase history |
| Purchasing | Purchase orders, partial receipts, supplier references |
| Sales | Sales orders, order items, confirmation, cancellation |
| Invoices | Invoice records, totals, status, printable representation |
| Reporting | Sales summaries, low-stock products, inventory valuation |
| Audit | Tenant-scoped record of important actions and actors |

### Explicitly out of scope for the first release

- Full accounting/general ledger, payroll, HR, manufacturing, and advanced tax filing.
- Country-specific e-invoicing integrations and multi-currency accounting.
- Microservices, Kubernetes, Kafka, and database-per-tenant provisioning.
- Complex custom workflow builders, mobile apps, and AI features.
- Active-active multi-region writes.

These are potential later extensions, not prerequisites for the initial SaaS product.

## 3. Core business workflows

### Purchasing and receiving

1. An authorized user creates a purchase order in the active tenant.
2. The user selects a supplier and adds products and quantities.
3. The order is submitted.
4. When goods arrive, the user records received quantities.
5. A database transaction updates inventory balances and appends stock movements.
6. The purchase order status and audit record are updated.

Partial deliveries are supported. Receiving 6 units from an order of 10 leaves 4 outstanding. Repeated requests must not apply the same receipt twice.

### Selling products

1. A user creates a sales order and selects a customer.
2. The user adds products and quantities.
3. The system validates that the order, customer, products, and warehouse belong to the active tenant.
4. On confirmation, a database transaction validates and deducts stock, appends inventory movements, creates an invoice, and updates the order.
5. The transaction commits only if every critical step succeeds.

### Inventory adjustments and cancellation

Every stock change creates an append-only movement with a reason, reference, tenant, and actor. A confirmed sale cancellation or return uses a compensating movement; historical movements are never silently deleted or rewritten.

## 4. Technology stack

| Layer | Technology | Purpose |
|---|---|---|
| Backend | Go | API and business logic |
| HTTP router | Chi | Routing and middleware |
| Database | PostgreSQL | Relational data, transactions, constraints, row-level security |
| PostgreSQL driver | pgx / pgxpool | Connectivity and connection pooling |
| Query generation | sqlc | Typed Go access from explicit SQL |
| Migrations | golang-migrate or Atlas | Versioned schema changes |
| Frontend | React + TypeScript | SaaS web application |
| API contract | OpenAPI | Contract and interactive documentation |
| Cache / rate limits | Redis (introduced when justified) | Short-lived cache, distributed rate limiting, ephemeral coordination |
| Async work | PostgreSQL outbox + Go workers initially | Reliable background tasks |
| Local environment | Docker Compose | Reproducible development stack |
| Integration tests | Testcontainers or Docker PostgreSQL | Tests against real PostgreSQL |
| Load tests | k6 | Throughput, latency, and concurrency validation |
| Metrics | Prometheus + Grafana | Application and infrastructure metrics |
| CI | GitHub Actions | Tests, linting, migrations, builds |

### Why SQL-first

Use explicit SQL, pgx, and sqlc rather than introducing an ORM initially. This keeps tenant predicates, transaction boundaries, locks, and query plans visible. Generated sqlc code must not be edited by hand.

## 5. Architecture

NexusERP starts as a **modular monolith**. One Go application contains modules with clear ownership, but related business operations can share one PostgreSQL transaction. This is easier to develop, test, and operate than microservices, and it is sufficient for the target when designed and measured carefully.

### Logical architecture

```text
 Browser (React + TypeScript)
          |
       HTTPS
          |
 CDN / WAF / Load Balancer
          |
   +------+------+------+
   |      |      |      |
 Go API Go API Go API ...     Stateless replicas
   |      |      |
   +------+------+----------+
          |
  Middleware and request context
  - authentication / session
  - active tenant membership
  - authorization / rate limits
  - request ID / tracing
          |
  Application services (modular monolith)
  - tenants / identity / RBAC
  - catalog / customers / suppliers
  - inventory / purchasing / sales
  - invoices / reports / audit
          |
  Repositories and sqlc queries
          |
   PostgreSQL primary ---- optional read replica
          |
   Outbox table -> Go background workers

 Redis (optional): cache / rate-limit counters
 Object storage (later): exports / uploaded documents
```

### Request and tenant context

1. Authenticate the caller and identify the user from a verified session/token.
2. Resolve the requested active tenant from a trusted route/subdomain or request context.
3. Verify that the authenticated user has an active membership in that tenant.
4. Load tenant-scoped roles/permissions and authorize the requested action.
5. Start the database transaction and set the tenant context transaction-locally for PostgreSQL RLS.
6. Execute tenant-scoped queries and commit/rollback.

A client-supplied `tenant_id` is **not proof of access**. Never trust a tenant identifier from a request without checking membership. Avoid a mutable global “current tenant” variable in the Go process; tenant context belongs to the request and transaction.

### Module and transaction rules

- HTTP handlers parse/validate transport input and call application services.
- Services own business workflows and transaction boundaries.
- Repositories/sqlc own persistence queries; all business queries must be tenant-scoped.
- Cross-module calls use explicit service interfaces or application-level functions, not direct access to another module's internals.
- Critical operations use one short PostgreSQL transaction.
- Do not make network calls to external services while holding database locks or open transactions.
- API instances are stateless: do not store sessions, tenant state, or business state only in process memory.
- Add interfaces where they create a real test seam or boundary; avoid interface-per-struct ceremony.

## 6. Multi-tenant database design

### Tenancy model

Use a shared schema with `tenant_id` on every tenant-owned business row. Platform-level records such as the global user identity and the tenant registry are exceptions. A global user can belong to multiple tenants through `tenant_memberships`.

Core platform tables:

- `tenants`: company/workspace record, slug, status, plan, created timestamp.
- `users`: global identity, normalized email, password hash or external identity reference, status.
- `tenant_memberships`: `(tenant_id, user_id)`, membership status, joined timestamp.
- `roles`: tenant-scoped role definitions (or system roles with an explicit scope).
- `permissions`: stable permission keys, e.g. `inventory.adjust`.
- `role_permissions`: permission grants for a role.
- `membership_roles`: roles assigned to a membership.
- `tenant_settings`: tenant-specific preferences and configuration.

Tenant-owned business tables:

- `categories`, `products`
- `customers`, `suppliers`, `warehouses`
- `inventory_balances`, `inventory_movements`
- `purchase_orders`, `purchase_order_items`, `purchase_receipts`, `purchase_receipt_items`
- `sales_orders`, `sales_order_items`
- `invoices`, `invoice_items`
- `audit_logs`, `idempotency_keys`, `outbox_events`

### Example relationship shape

```text
users
  | 1
  | many
 tenant_memberships ---- tenants
          |                 |
          |                 +---- products / customers / suppliers
          |                 +---- warehouses / inventory
          |                 +---- purchase_orders / sales_orders
          |                 +---- invoices / audit_logs
          |
      membership_roles ---- roles ---- role_permissions ---- permissions
```

### Tenant-aware schema rules

Every tenant-owned table should include `tenant_id NOT NULL`. Use tenant-scoped unique constraints and foreign keys. For example:

```sql
CREATE TABLE products (
    id          UUID NOT NULL,
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    sku         TEXT NOT NULL,
    name        TEXT NOT NULL,
    unit_price  NUMERIC(19, 4) NOT NULL CHECK (unit_price >= 0),
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    UNIQUE (tenant_id, sku)
);

CREATE INDEX products_tenant_created_idx
    ON products (tenant_id, created_at DESC);

CREATE INDEX products_tenant_name_idx
    ON products (tenant_id, name);
```

For references between tenant-owned records, use composite foreign keys so a row cannot reference another tenant's record even if application code has a bug:

```sql
CREATE TABLE customers (
    id UUID NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    PRIMARY KEY (tenant_id, id)
);

ALTER TABLE sales_orders
    ADD CONSTRAINT sales_orders_customer_same_tenant_fk
    FOREIGN KEY (tenant_id, customer_id)
    REFERENCES customers (tenant_id, id);
```

Adapt exact keys and constraints to the final migration design. UUIDs are useful identifiers but do **not** provide tenant isolation by themselves.

### Tenant isolation: defense in depth

Use all of the following:

1. **Authorization:** verify active membership and permission before a tenant operation.
2. **Repository scoping:** every tenant-owned query includes `tenant_id = $tenantID`; avoid unscoped generic repository methods.
3. **Composite constraints:** composite primary/unique keys and foreign keys prevent cross-tenant references.
4. **PostgreSQL Row-Level Security (RLS):** enable policies on tenant-owned tables as a second boundary.
5. **Isolation tests:** explicitly test that Tenant A cannot read, update, delete, or reference Tenant B's rows.

Example RLS policy pattern:

```sql
ALTER TABLE products ENABLE ROW LEVEL SECURITY;
ALTER TABLE products FORCE ROW LEVEL SECURITY;

CREATE POLICY products_tenant_isolation ON products
USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);
```

At the start of **every transaction**, set the tenant value transaction-locally, for example with `SELECT set_config('app.tenant_id', $1, true)`. The final `true` makes the setting local to the transaction, preventing tenant context from leaking through pooled connections. Do not use session-level `SET` on pooled connections. The application database role must not own the protected tables and must not have `BYPASSRLS`; use a separate migration/owner role. RLS is defense in depth, not a substitute for explicit authorization and query scoping.

### Key business tables and fields

| Table | Important fields / relationships |
|---|---|
| `products` | tenant_id, id, sku, name, category_id, unit_price, cost_price, active |
| `customers` | tenant_id, id, name, email, phone |
| `suppliers` | tenant_id, id, name, email, phone |
| `warehouses` | tenant_id, id, name, location |
| `inventory_balances` | tenant_id, warehouse_id, product_id, quantity; unique per tenant/warehouse/product |
| `inventory_movements` | tenant_id, id, warehouse_id, product_id, movement_type, quantity_delta, reference, actor_id, created_at |
| `purchase_orders` | tenant_id, id, supplier_id, status, order_date, expected_date |
| `purchase_order_items` | tenant_id, id, purchase_order_id, product_id, ordered_quantity, received_quantity, unit_cost |
| `sales_orders` | tenant_id, id, customer_id, status, order_date, total_amount |
| `sales_order_items` | tenant_id, id, sales_order_id, product_id, quantity, unit_price |
| `invoices` | tenant_id, id, sales_order_id, invoice_number, status, total_amount, issued_at |
| `audit_logs` | tenant_id, id, actor_id, action, entity_type, entity_id, before_data, after_data, created_at |

Use `NUMERIC`/decimal-safe handling for money, never floating point. Define currency and rounding policy before invoice implementation. Decide whether invoice numbers are unique per tenant or globally; a tenant-scoped sequence/numbering strategy is usually appropriate.

### Indexing and pagination

- Most business indexes begin with `tenant_id`, followed by the filter/order columns used by real queries.
- Add composite indexes for common access patterns, e.g. `(tenant_id, status, created_at DESC)` or `(tenant_id, warehouse_id, product_id)`.
- Use cursor/keyset pagination for large tables; use a stable ordering such as `(created_at, id)` and include tenant scope in the query.
- Do not add indexes speculatively. Validate with `EXPLAIN (ANALYZE, BUFFERS)` and representative tenant data.
- Keep tenant IDs in cache keys and background-job payloads. Never share tenant-specific cached values under a key that omits tenant identity.

## 7. API design

All application endpoints are versioned under `/api/v1`. Tenant context can be expressed as `/api/v1/tenants/{tenantID}/...` or resolved from a verified tenant subdomain/header. Pick one public convention and use it consistently. Regardless of convention, the backend verifies membership; the URL/header is only a selector, not authorization.

Suggested platform endpoints:

| Method | Endpoint | Purpose |
|---|---|---|
| POST | `/auth/login` | Authenticate |
| POST | `/auth/refresh` | Rotate/refresh session credentials |
| POST | `/auth/logout` | Revoke session |
| GET | `/users/me` | Current global user |
| GET | `/me/tenants` | Tenants available to current user |
| POST | `/tenants` | Create a tenant/workspace |
| GET | `/tenants/{tenantID}/members` | List tenant members |
| POST | `/tenants/{tenantID}/invitations` | Invite a member |

Tenant-scoped resources:

- `/api/v1/tenants/{tenantID}/products`
- `/api/v1/tenants/{tenantID}/categories`
- `/api/v1/tenants/{tenantID}/inventory`
- `/api/v1/tenants/{tenantID}/inventory/movements`
- `/api/v1/tenants/{tenantID}/warehouses`
- `/api/v1/tenants/{tenantID}/customers`
- `/api/v1/tenants/{tenantID}/suppliers`
- `/api/v1/tenants/{tenantID}/purchase-orders`
- `/api/v1/tenants/{tenantID}/sales-orders`
- `/api/v1/tenants/{tenantID}/invoices`
- `/api/v1/tenants/{tenantID}/reports/...`

API requirements:

- Validate input and enforce authorization server-side.
- Use consistent error responses and request IDs.
- Support filtering, sorting, and cursor pagination on list endpoints.
- Apply per-user and per-tenant rate limits where appropriate.
- Use idempotency keys for retryable business writes.
- Publish an OpenAPI specification.
- Avoid leaking whether another tenant's resource exists; return a consistent not-found/forbidden policy.

## 8. Business consistency and concurrency

### Safe stock deduction

A sale must not allow concurrent requests to sell the same final unit. Use a transaction and either lock the relevant balance row with `SELECT ... FOR UPDATE` or use a conditional atomic update that only decrements when enough stock remains. Verify affected-row counts. For multi-product orders, lock rows in a consistent sorted order to reduce deadlocks. Keep transactions short.

### Atomic sales confirmation

Within one transaction:

1. Validate order state and tenant ownership.
2. Lock/check required inventory balances.
3. Deduct stock.
4. Insert inventory movements.
5. Create invoice and invoice lines.
6. Update order status and write audit/outbox records.
7. Commit.

Any failure rolls back the whole workflow.

### Idempotency and outbox

For operations such as sales confirmation, purchase receipts, and payment callbacks, persist an idempotency key scoped by tenant and operation. A repeated key returns the original result rather than applying the operation twice.

When a committed transaction must trigger asynchronous work (email, export, webhook), insert an `outbox_events` row in the same transaction. A worker publishes/processes it and marks it complete with retry/backoff. This avoids the dual-write bug where the database commits but event publication fails. Start with PostgreSQL-backed outbox polling; introduce a dedicated broker only when measured throughput or delivery requirements justify it.

### Inventory ledger

`inventory_movements` is the append-only source of truth. `inventory_balances` is a transactionally maintained projection for fast reads. Update both in the same transaction and periodically reconcile balances against the movement ledger.

## 9. Capacity and performance plan

### Workload assumptions

The initial test profile should represent 1,000 simultaneously active users across multiple tenants, with a deliberately uneven tenant distribution (a few larger tenants and many smaller tenants). Start with this approximate scenario:

- 1,000 active user sessions.
- 100–200 sustained requests/second during normal peak activity.
- 500 requests/second during a short burst test.
- A mixed workload: catalog/inventory reads, dashboard/list queries, order creation, purchase receipts, and sales confirmation.
- A small number of report/export requests running concurrently; heavy exports should be asynchronous.

Treat 200 sustained RPS as the initial benchmark target, not as a guaranteed conversion from 1,000 active users. Refine the request rate using observed usage patterns and load-test results.

### Scaling strategy

**Go API:** Run multiple stateless replicas behind a load balancer. Scale replicas based on CPU, memory, request queue/latency, and in-flight requests. Use graceful shutdown and readiness checks so deployments do not drop in-flight work.

**PostgreSQL:** Start with one well-provisioned primary and connection pooling. PostgreSQL is the source of truth. Tune based on query plans, lock waits, CPU, memory, storage latency, and connection utilization. Add a read replica only when read pressure is measured and the application can tolerate replica lag. Inventory decisions and read-after-write flows must use the primary.

**Connection management:** Use `pgxpool` with explicit limits and timeouts. Set the maximum pool size per API replica so the sum of all replica pools plus workers stays within the database's connection budget. Do not let every replica independently open an unbounded pool. Consider PgBouncer if connection churn or connection count becomes a bottleneck.

**Redis:** Add only for measured use cases such as short-lived catalog/config caching or distributed rate limiting. PostgreSQL remains the source of truth. Cache keys must include tenant identity, and invalidation must be explicit. Never cache authorization decisions for longer than the security model safely permits.

**Background workers:** Move slow or non-interactive tasks (reports, exports, notifications, webhook delivery) out of request handlers. Use bounded concurrency, retries with backoff, dead-letter handling, and tenant-aware job payloads. Do not run an unbounded goroutine per task.

**Reports:** Start with indexed queries and date-bounded reports. For expensive reports, enqueue a job and provide a downloadable result. Consider pre-aggregated tables/materialized views only after profiling.

### Initial deployment sizing (starting point only)

A reasonable first staging/production experiment—not a final sizing guarantee—is:

| Component | Starting configuration | Scale trigger |
|---|---|---|
| Load balancer / ingress | Managed HTTPS load balancer | Availability, routing, WAF needs |
| Go API | 2–3 replicas, each initially 2–4 vCPU and 2–4 GB RAM | CPU sustained above ~65–70%, rising p95 latency, or saturation in-flight requests |
| PostgreSQL primary | Managed PostgreSQL, initially 4–8 vCPU and 16–32 GB RAM, SSD storage | CPU/IO saturation, lock waits, slow queries, connection pressure |
| Redis | Small managed instance only when introduced | Cache/rate-limit workload justifies it |
| Worker | 1–2 replicas, bounded concurrency | Queue/outbox lag and task completion time |
| Object storage | Managed object storage when exports/uploads are added | File storage requirement |

These are starting ranges for benchmarking, not universal minimums. Cloud provider, database version, storage IOPS, data size, and query mix can change the result substantially. Prefer managed PostgreSQL with automated backups and point-in-time recovery for a serious hosted deployment.

### Performance engineering rules

- Measure p50, p95, and p99 latency by endpoint and tenant.
- Track throughput, error rate, CPU, memory, DB pool wait time, DB connections, slow queries, lock waits, and worker queue age.
- Set request, query, and transaction timeouts.
- Avoid N+1 queries and unbounded list endpoints.
- Use cursor pagination and bounded response sizes.
- Keep API instances stateless and avoid sticky sessions.
- Test both average tenant data and a “large tenant” dataset to detect noisy-neighbor problems.
- Establish per-tenant rate limits and quotas before one tenant can monopolize shared resources.

## 10. Infrastructure and deployment

### Environments

- **Local:** Docker Compose for PostgreSQL and optional Redis; Go API and React dev server.
- **CI:** lint, unit tests, integration tests against PostgreSQL, migration validation, build, and dependency/security checks.
- **Staging:** production-like topology with seeded multi-tenant data; run smoke and load tests here.
- **Production:** HTTPS load balancer, multiple API replicas, managed PostgreSQL, backups/PITR, secrets manager, centralized logs/metrics, and alerting.

### Production topology

```text
                 Internet
                    |
             DNS / CDN / WAF
                    |
          Managed HTTPS load balancer
                    |
          +---------+---------+
          |         |         |
        API-1     API-2     API-N
          |         |         |
          +---------+---------+
                    |
             PostgreSQL primary
               |          |
          Outbox table   Read replica
               |
         Go worker pool

     Optional Redis / Object storage
```

### Deployment and operations requirements

- Use container images with non-root runtime users and multi-stage builds.
- Keep secrets out of images, source control, and logs; use a secrets manager in hosted environments.
- Apply migrations as a controlled deployment step, not independently from every API replica at startup.
- Use backward-compatible expand/migrate/contract schema changes for rolling deployments.
- Add liveness and readiness endpoints; readiness should fail when the instance cannot safely serve traffic.
- Configure graceful shutdown and bounded request draining.
- Automate backups and test restoration, not just backup creation.
- Use infrastructure-as-code when the deployment target is selected.
- Keep a rollback procedure and document recovery objectives.

Kubernetes is deliberately not required for this target. Start with managed containers or a simple container platform. Add Kubernetes only if operational requirements or deployment complexity justify it.

## 11. Repository structure

```text
nexuserp/
├── backend/
│   ├── cmd/
│   │   ├── api/main.go
│   │   └── worker/main.go
│   ├── internal/
│   │   ├── platform/       # tenant registry, identity, memberships
│   │   ├── auth/           # sessions and authentication
│   │   ├── authorization/  # tenant-scoped RBAC
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
│   │   ├── audit/
│   │   ├── outbox/
│   │   └── platformhttp/   # middleware, request context, health
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
│   ├── docker-compose.yml
│   └── production/         # deployment/IaC configuration later
├── docs/
│   ├── architecture/       # ADRs and diagrams
│   ├── database/            # ERD and isolation model
│   ├── operations/          # runbooks and recovery
│   └── api/
├── load-tests/              # k6 scenarios and datasets
├── .github/workflows/
├── .env.example
└── README.md
```

## 12. Milestones and implementation plan

Plan for **10–12 weeks of focused part-time work** as a realistic first pass. The schedule is an estimate; tenant isolation and concurrency correctness deserve more time than adding extra screens. Keep a working vertical slice at every stage.

### Milestone 0 — Product, tenancy, and architecture design

- [ ] Define the target customer and initial ERP workflows.
- [ ] Confirm shared-schema multi-tenancy and document why.
- [ ] Define tenant lifecycle: creation, invitation, membership removal, suspension.
- [ ] Define user identity vs tenant membership vs tenant role.
- [ ] Draw ERD with `tenant_id` on every tenant-owned entity.
- [ ] Define composite keys/foreign keys and tenant-aware indexes.
- [ ] Decide tenant resolution convention (tenant URL path or subdomain).
- [ ] Write ADRs for RLS, auth/session strategy, outbox, and deployment approach.
- [ ] Define the 1,000-active-user workload and load-test scenarios.

**Deliverables:** ERD, tenancy threat model, architecture diagram, ADRs, workflow diagrams, API outline, capacity assumptions.

**Acceptance:** Explain how every request is associated with a tenant and why one tenant cannot access another tenant's records.

### Milestone 1 — Go foundation and development environment

- [ ] Create Go module and Chi server.
- [ ] Add pgxpool and PostgreSQL Docker Compose service.
- [ ] Add environment-based config, structured logs, request IDs, graceful shutdown.
- [ ] Add liveness/readiness endpoints and HTTP timeouts.
- [ ] Add migrations and sqlc generation.
- [ ] Add CI for formatting, vet, lint, and build.

**Deliverables:** Runnable API, database, migrations, sqlc pipeline, CI.

**Acceptance:** A fresh clone can start locally and pass automated checks.

### Milestone 2 — Identity, tenant lifecycle, and isolation foundation

- [ ] Implement global users and secure password/session handling.
- [ ] Implement tenant creation and tenant membership.
- [ ] Implement invitations and membership status transitions.
- [ ] Implement tenant-scoped roles, permissions, and membership-role assignments.
- [ ] Add tenant resolution and membership-verification middleware.
- [ ] Add transaction helper that sets transaction-local RLS context.
- [ ] Enable RLS on initial tenant-owned tables; use a non-owner application DB role.
- [ ] Add composite tenant-aware keys and foreign keys.
- [ ] Add cross-tenant isolation integration tests, including pooled-connection reuse.

**Deliverables:** Identity and tenant foundation with verified isolation.

**Acceptance:** A user can belong to two tenants; switching context works; access to another tenant's records is denied even when IDs are guessed or requests are manipulated.

### Milestone 3 — Master data and tenant-scoped CRUD

- [ ] Implement categories and products.
- [ ] Implement customers and suppliers.
- [ ] Enforce tenant-scoped unique SKU and appropriate constraints.
- [ ] Implement cursor pagination, filtering, and sorting.
- [ ] Add tenant-aware indexes based on expected queries.
- [ ] Add frontend tenant switcher and protected routes.

**Deliverables:** Tenant-isolated catalog and master-data APIs/UI.

**Acceptance:** Duplicate SKUs are rejected within a tenant but can exist in different tenants; list endpoints never mix tenant data.

### Milestone 4 — Inventory ledger and warehouses

- [ ] Implement warehouses and tenant-scoped inventory balances.
- [ ] Implement append-only inventory movements.
- [ ] Add opening stock and authorized adjustments.
- [ ] Implement atomic balance + movement updates.
- [ ] Add stock lookup and movement history.
- [ ] Add reconciliation query/test.
- [ ] Add concurrency tests for competing stock updates.

**Deliverables:** Correct inventory module.

**Acceptance:** Every stock change is auditable and tenant-isolated; balance projections match the ledger.

### Milestone 5 — Purchasing and receiving

- [ ] Implement purchase orders and line items.
- [ ] Implement submit and partial receipt flows.
- [ ] Ensure supplier/product references are in the same tenant.
- [ ] Update inventory transactionally on receipt.
- [ ] Add tenant-scoped idempotency for receipt submission.
- [ ] Add rollback, duplicate-retry, and partial-receipt tests.

**Deliverables:** End-to-end purchasing workflow.

**Acceptance:** Partial and repeated receipts behave correctly without cross-tenant references or duplicate stock increases.

### Milestone 6 — Sales and invoicing

- [ ] Implement sales orders and line items.
- [ ] Confirm orders in a single transaction.
- [ ] Prevent overselling using row locks or conditional atomic updates.
- [ ] Create invoices and invoice lines transactionally.
- [ ] Implement cancellation and compensating stock movements.
- [ ] Add idempotency and concurrent-sale tests.

**Deliverables:** Transaction-safe sales and invoice workflow.

**Acceptance:** No overselling under concurrency; failed operations leave no partial business changes; tenant boundaries hold throughout the workflow.

### Milestone 7 — Frontend, reporting, and tenant administration

- [ ] Build dashboard, tenant switcher, member/role management.
- [ ] Build product, customer, supplier, warehouse, and inventory screens.
- [ ] Build purchasing, receiving, sales, and invoice screens.
- [ ] Add sales summary, top products, low-stock, and valuation reports.
- [ ] Add loading, empty, validation, and error states.
- [ ] Ensure all frontend requests carry the selected tenant context safely.

**Deliverables:** Usable SaaS UI for the main workflows.

**Acceptance:** Users can switch tenants and complete core workflows without seeing cross-tenant data.

### Milestone 8 — Reliability, background work, and observability

- [ ] Add transactional outbox table and worker process.
- [ ] Move report exports and notifications to background jobs.
- [ ] Add bounded retries, backoff, and failed-job handling.
- [ ] Add Prometheus metrics and Grafana dashboards.
- [ ] Track request latency, errors, DB pool wait, DB connections, query latency, lock waits, and worker lag.
- [ ] Add structured logs and trace/request correlation.
- [ ] Add per-tenant rate limits and basic noisy-neighbor protections.

**Deliverables:** Observable API and reliable asynchronous processing.

**Acceptance:** Background work survives API restarts and failures; key saturation signals are visible.

### Milestone 9 — Performance and 1,000-user load validation

- [ ] Create representative multi-tenant seed data, including large and small tenants.
- [ ] Write k6 scenarios for login/session use, reads, writes, sales, receiving, and reports.
- [ ] Simulate 1,000 active users with realistic think time.
- [ ] Test 100–200 sustained RPS and a 500 RPS burst profile.
- [ ] Measure p50/p95/p99 latency, throughput, errors, CPU, memory, DB pool wait, locks, and slow queries.
- [ ] Run tenant-isolation tests under concurrent load.
- [ ] Profile slow queries and inspect EXPLAIN ANALYZE plans.
- [ ] Tune indexes, connection pools, API replicas, and database resources based on evidence.
- [ ] Record test environment, dataset, results, bottlenecks, and limitations in `docs/`.

**Deliverables:** Repeatable load-test suite and capacity report.

**Acceptance:** The target workload meets the documented latency/error objectives in the test environment, or the report clearly identifies the remaining bottleneck and next tuning step. Do not claim 1,000-user support without publishing measured results.

### Milestone 10 — Production deployment and operational readiness

- [ ] Deploy multiple stateless API replicas behind HTTPS load balancing.
- [ ] Use managed PostgreSQL with automated backups and point-in-time recovery.
- [ ] Configure secrets, network restrictions, TLS, and least-privilege DB roles.
- [ ] Add deployment health checks, graceful shutdown, and rollback procedure.
- [ ] Validate migration compatibility during rolling deployments.
- [ ] Test database restore and document recovery procedure.
- [ ] Add alerting for error rate, latency, saturation, DB health, and worker lag.
- [ ] Publish OpenAPI docs, screenshots, demo data, and a short product demo.
- [ ] Document architecture, setup, load-test results, and known limitations.

**Deliverables:** Deployed, observable, documented portfolio release.

**Acceptance:** Another developer can run the app locally; a reviewer can use the demo; operations and measured capacity are documented.

### Suggested schedule

| Weeks | Focus | Main result |
|---|---|---|
| 1 | Product and tenancy design | ERD, ADRs, capacity plan |
| 2–3 | Go foundation, identity, tenancy, RLS | Secure tenant-aware foundation |
| 4 | Master data | Products, customers, suppliers |
| 5 | Inventory | Ledger and balances |
| 6 | Purchasing | Partial receiving |
| 7 | Sales and invoices | Transaction-safe sales |
| 8–9 | Frontend and reporting | Usable multi-tenant application |
| 10 | Outbox, observability, reliability | Workers and dashboards |
| 11 | Load testing and tuning | 1,000-active-user benchmark report |
| 12 | Deployment and documentation | Portfolio-ready release |

If time is constrained, prioritize milestones 0–6 and a small working frontend. Do not skip tenant isolation, transaction correctness, or cross-tenant tests to add more features.

## 13. Testing and load validation

### Unit and integration tests

Test business rules and run persistence tests against real PostgreSQL:

- Tenant membership and role authorization.
- Cross-tenant read/update/delete/reference attempts.
- RLS behavior with transaction-local context and pooled connections.
- Tenant-scoped unique constraints and composite foreign keys.
- Inventory balance/movement atomicity and reconciliation.
- Partial receipt, duplicate idempotency key, and rollback.
- Sales confirmation, invoice creation, cancellation, and rollback.
- Concurrent sales competing for the last units.
- Outbox retry and duplicate delivery behavior.

### Load tests

Use k6 or an equivalent tool. Include realistic think time rather than issuing continuous requests from every virtual user. Test:

1. Ramp-up to 1,000 active virtual users.
2. A steady-state workload at the target request rate.
3. A short burst to 500 RPS.
4. A mixed read/write workload across many tenants.
5. A noisy-neighbor case where one large tenant performs heavy reads/reports.
6. A recovery test after API replica restart or worker restart.

Capture the test configuration and results. A successful test should report endpoint-level p95/p99, error rate, achieved RPS, and infrastructure saturation. A test that reaches 1,000 virtual users but sends almost no requests is not proof of 1,000-user capacity.

Useful Go commands:

```bash
go test ./...
go test -race ./...
gofmt -w .
go vet ./...
```

## 14. Security, isolation, and reliability

- Hash passwords with a suitable password-hashing algorithm; never store plaintext.
- Use secure, revocable sessions or carefully designed access/refresh token rotation.
- Verify tenant membership and permission on every tenant operation.
- Never trust a client-supplied tenant ID without authorization.
- Use RLS, composite foreign keys, and explicit tenant predicates as defense in depth.
- Use transaction-local tenant context; never leak tenant state across pooled DB connections.
- Run the app with a least-privilege database role that does not own protected tables or bypass RLS.
- Keep secrets in a secrets manager in hosted environments.
- Validate input, use parameterized SQL, and apply request/query timeouts.
- Use HTTPS and secure cookie settings where cookies are used.
- Do not expose internal errors or log credentials, tokens, or sensitive customer data.
- Apply per-tenant quotas/rate limits to reduce noisy-neighbor risk.
- Back up production data, test restoration, and document recovery objectives.
- Treat exports, cache entries, background jobs, audit records, and object-storage paths as tenant-scoped resources too.

## 15. Local development

The exact commands and environment variables will be finalized during Milestone 1. Intended workflow:

```bash
git clone <repository-url>
cd nexuserp
cp .env.example .env
docker compose -f deployments/docker-compose.yml up -d
# Run database migrations using the selected migration tool.
cd backend
go run ./cmd/api
```

Example local-only configuration:

```dotenv
APP_ENV=development
HTTP_ADDR=:8080
DATABASE_URL=postgres://nexuserp:nexuserp@localhost:5432/nexuserp?sslmode=disable
# Generate a local secret; never use this example in production.
AUTH_SECRET=replace-with-a-local-development-secret
```

Never commit real `.env` files or production credentials. The local setup should include a seed command that creates two demo tenants and users so isolation can be tested immediately.

## 16. Definition of done

The first portfolio-ready release is complete when:

- A user can create or join a company and switch between authorized tenants.
- Tenant-owned data is isolated through authorization, scoped queries, database constraints, and RLS.
- Core product, inventory, purchasing, sales, and invoicing workflows work end-to-end.
- Inventory remains consistent under concurrent sales and receiving.
- Retryable writes are idempotent.
- Background jobs are reliable and observable.
- CI runs unit, integration, and migration checks.
- A repeatable load test simulates 1,000 active users and reports measured results.
- Deployment, backups, restore procedure, and operational metrics are documented.
- A new developer can run the project using the README.

## 17. Future improvements

Only consider these after the core product and its measured bottlenecks are understood:

- PostgreSQL read replicas for suitable read-only workloads.
- PgBouncer for connection management.
- Partitioning very large append-only tables such as audit logs or inventory movements.
- Materialized views or dedicated reporting/analytics storage.
- Dedicated message broker if outbox polling no longer meets delivery/throughput needs.
- Per-tenant database/schema isolation for enterprise customers with special compliance or noisy-neighbor requirements.
- Kubernetes if deployment and operations justify its complexity.
- Multi-region read replicas, advanced billing, subscription plans, and tenant usage metering.
- Full accounting, e-invoicing, integrations, and AI-assisted workflows.

Do not introduce microservices, Kafka, Kubernetes, or database-per-tenant architecture just to make the diagram look scalable. Make changes when workload measurements or product requirements justify them.

## 18. CV description

Use wording that accurately reflects what has actually been implemented and measured. Example future-facing description:

> Designed and developed NexusERP, a multi-tenant SaaS ERP platform using Go, PostgreSQL, React, and TypeScript. Implemented tenant isolation with PostgreSQL Row-Level Security and tenant-aware relational constraints, transaction-safe inventory and sales workflows, and stateless API architecture. Built automated integration and load tests to evaluate performance under a 1,000-active-user workload.

Only claim RLS, deployment, load-test results, or 1,000-user performance after those features have been implemented and verified. Replace the final sentence with actual benchmark figures once measured.
