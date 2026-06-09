# ⚡ ZapMarket

ZapMarket is a scalable, cloud-native e-commerce platform built using a Go monorepo architecture.  
The system is designed around microservices, event-driven communication, and modern distributed system patterns.

---

# 🏗️ Architecture Overview

The platform follows a modular microservice architecture powered by:

- **Go Workspaces (`go.work`)** for monorepo management
- **gRPC** for internal service communication
- **Kafka** for asynchronous event streaming
- **PostgreSQL** for transactional persistence
- **Redis** for caching and distributed coordination
- **Elasticsearch** for product search and indexing
- **MinIO** for object storage
- **Debezium CDC** for change data capture pipelines

The complete architecture and design decisions are documented in:

```text
design.md
```

---

# 📂 Repository Structure

```text
zapmarket/
│
├── services/
│   ├── auth-service/
│   ├── product-catalog-service/
│   ├── order-management-service/
│   ├── inventory-service/
│   ├── payment-service/
│   └── notification-service/
│
├── pkg/
│   ├── proto/
│   ├── logger/
│   ├── middleware/
│   ├── kafka/
│   ├── redis/
│   ├── database/
│   └── utils/
│
├── deploy/
├── scripts/
├── docker-compose.yml
├── go.work
├── design.md
├── AGENTS.md
└── README.md
```

---

# 🚀 Services

| Service | Responsibility |
|---|---|
| `auth-service` | Authentication, authorization, JWT, user identity |
| `product-catalog-service` | Product management, categories, search indexing |
| `order-management-service` | Order lifecycle and orchestration |
| `inventory-service` | Inventory tracking and stock reservation |
| `payment-service` | Payment processing and transaction workflows |
| `notification-service` | Email, SMS, and event notifications |

---

# 🧩 Shared Packages

The `pkg/` directory contains reusable shared modules across services:

- gRPC protobuf definitions
- Kafka producers/consumers
- Database helpers
- Redis utilities
- Logging and tracing
- Middleware and interceptors
- Shared DTOs and contracts

---

# ⚙️ Prerequisites

Ensure the following are installed:

- Go `1.22+`
- Docker
- Docker Compose

---

# 🛠️ Getting Started

## 1. Clone the Repository

```bash
git clone https://github.com/your-org/zapmarket.git

cd zapmarket
```

---

## 2. Sync Go Workspaces

```bash
go work sync
```

---

## 3. Start Infrastructure Dependencies

```bash
docker compose up -d
```

This will start:

- PostgreSQL
- Kafka
- Zookeeper
- Redis
- Elasticsearch
- MinIO
- Debezium

---

# 🗄️ Database Notes

The PostgreSQL initialization scripts automatically enable:

```sql
pgcrypto
```

This allows usage of:

```sql
gen_random_uuid()
```

across all service databases.

---

# 📦 MinIO Access

| Component | URL |
|---|---|
| MinIO API | `http://localhost:9000` |
| MinIO Console | `http://localhost:9001` |

---

# 🔌 Service Development

Each service inside `services/` is an independent Go module.

Typical service structure:

```text
service-name/
├── cmd/
├── internal/
├── api/
├── configs/
├── migrations/
├── Dockerfile
├── go.mod
└── main.go
```

---

# 🧪 Current Status

✅ Repository scaffolding completed  
✅ Go workspace configured  
✅ Local infrastructure setup available  

🚧 Service implementations are currently in progress.

---

# 📡 Communication Patterns

| Communication Type | Technology |
|---|---|
| Synchronous | gRPC |
| Asynchronous | Kafka Events |
| Search | Elasticsearch |
| Caching | Redis |
| Object Storage | MinIO |

---

# 📋 Planned Features

- Distributed tracing
- API Gateway
- Rate limiting
- Service discovery
- Event sourcing support
- Saga orchestration
- Multi-tenant support
- Observability stack
- CI/CD pipelines
- Kubernetes deployment manifests

---

# 🤝 Development Workflow

## Generate Protobufs

```bash
make proto
```

## Run a Service

```bash
cd services/auth-service

go run .
```

## Run Tests

```bash
go test ./...
```

---

# 📖 Documentation

| File | Description |
|---|---|
| `design.md` | System architecture and design |
| `AGENTS.md` | AI agent and repository conventions |

---

# 🧠 Engineering Principles

- Domain-driven service boundaries
- Event-driven architecture
- High cohesion, low coupling
- Infrastructure abstraction
- Cloud-native deployment readiness
- Horizontal scalability
- Observability-first design

---

# 📄 License

This project is licensed under the MIT License.
