# Local Development Guide

---

## Prerequisites

- **Go 1.22+**
- **Docker Desktop & Docker Compose v5+**
- **Make** (or PowerShell / Git Bash)

---

## Quickstart

1. **Start Local Infrastructure Containers**:
   ```bash
   make dev-up
   ```
   *Launches Postgres (5432), Redis (6379), Kafka (9092), Centrifugo (8000), APISIX (9080/9090), and Jaeger (16686).*

2. **Sync Go Workspace Dependencies**:
   ```bash
   make tidy
   ```

3. **Build Microservices**:
   ```bash
   make build
   ```

4. **Stop Local Infrastructure Containers**:
   ```bash
   make dev-down
   ```
