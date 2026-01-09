# Kubernetes App Store - Implementation Plan

**Project Name**: `appstore`  
**CRD Domain**: `appstore.bitpipe.no`

## Overview
A web-based "App Store" for Kubernetes where developer teams deploy standardized infrastructure apps (PostgreSQL, MongoDB, MinIO, OpenBao, Valkey, etc.) via Helm charts.

## Architecture
```
Svelte Frontend → Go Backend API (net/http) → RabbitMQ → K8s Operator (Kubebuilder) → Helm → Apps
                         ↓
                    Keycloak (Azure IDP)
```

## Confirmed Requirements
- **Operator**: Go with Kubebuilder/Operator SDK
- **Frontend**: Svelte (SvelteKit)
- **Backend**: Go with standard `net/http` (no frameworks)
- **Auth**: Keycloak (existing, connected to Azure IDP)
- **Charts**: Predefined catalog of org-adapted charts
- **MVP**: Single cluster
- **Future**: Multi-cluster (one team = one cluster, operator per cluster)

---

## Project Structure (Monorepo)

```
appstore/
├── operator/                    # Kubernetes Operator (Go + Kubebuilder)
│   ├── cmd/main.go
│   ├── api/v1alpha1/
│   │   └── appdeployment_types.go
│   ├── internal/
│   │   ├── controller/
│   │   ├── helm/
│   │   └── rabbitmq/
│   └── config/crd/
├── backend/                     # Go Backend API
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/
│   │   ├── auth/
│   │   ├── catalog/
│   │   ├── deployment/
│   │   └── rabbitmq/
│   └── pkg/models/
├── frontend/                    # SvelteKit Frontend
│   ├── src/
│   │   ├── lib/
│   │   │   ├── stores/
│   │   │   ├── api/
│   │   │   └── components/
│   │   └── routes/
├── charts/                      # App catalog charts
│   ├── catalog.yaml
│   └── apps/
├── deploy/                      # Deployment manifests
└── docker-compose.yaml          # Local dev environment
```

---

## Phased Implementation

### Phase 1: Operator Foundation
**Goal**: Working operator that deploys Helm charts from CRDs

**Tasks**:
1. Initialize monorepo structure with `operator/` and `backend/` directories
2. Initialize Kubebuilder project: `kubebuilder init --domain bitpipe.no --repo appstore/operator`
3. Create AppDeployment CRD: `kubebuilder create api --group appstore --version v1alpha1 --kind AppDeployment`
4. Define CRD types in `api/v1alpha1/appdeployment_types.go`
5. Implement Helm client wrapper in `internal/helm/client.go`
6. Implement reconciliation loop in `internal/controller/appdeployment_controller.go`
7. Test with local k8s (kind/minikube) - deploy PostgreSQL from CR

**Key Files**:
- `operator/api/v1alpha1/appdeployment_types.go` - CRD schema
- `operator/internal/controller/appdeployment_controller.go` - Reconciler
- `operator/internal/helm/client.go` - Helm SDK integration

---

### Phase 2: RabbitMQ Integration
**Goal**: Operator consumes messages, backend publishes them

**Tasks**:
1. Add RabbitMQ to docker-compose.yaml for local dev
2. Define message schemas in `backend/pkg/models/messages.go`
3. Implement RabbitMQ consumer in operator (`internal/rabbitmq/consumer.go`)
4. Message handler creates AppDeployment CRs
5. Initialize backend Go project
6. Implement RabbitMQ publisher (`internal/rabbitmq/publisher.go`)
7. Test end-to-end: HTTP request → RabbitMQ → Operator → Helm release

**Key Files**:
- `backend/pkg/models/messages.go` - Shared message types
- `operator/internal/rabbitmq/consumer.go` - Queue consumer
- `backend/internal/rabbitmq/publisher.go` - Queue publisher

---

### Phase 3: Backend API & Auth
**Goal**: REST API with Keycloak authentication

**Tasks**:
1. Set up HTTP router in `internal/api/router.go` using `net/http`
2. Implement Keycloak JWT validation middleware
3. Implement catalog endpoints (`GET /api/v1/catalog`, etc.)
4. Implement deployment endpoints (`POST/GET/PUT/DELETE /api/v1/deployments`)
5. Add team-based authorization from Keycloak claims
6. Create catalog.yaml with 3-5 initial apps

**API Endpoints**:
```
GET  /api/v1/catalog                    # List apps
GET  /api/v1/catalog/{appName}          # App details
POST /api/v1/deployments                # Create deployment
GET  /api/v1/deployments                # List my deployments
GET  /api/v1/deployments/{id}           # Get deployment
PUT  /api/v1/deployments/{id}           # Update deployment
DELETE /api/v1/deployments/{id}         # Delete deployment
```

**Key Files**:
- `backend/internal/api/router.go` - HTTP routing
- `backend/internal/auth/middleware.go` - JWT validation
- `backend/internal/deployment/handler.go` - Deployment handlers
- `charts/catalog.yaml` - App catalog definition

---

### Phase 4: Svelte Frontend
**Goal**: Functional UI for browsing and deploying apps

**Tasks**:
1. Initialize SvelteKit project with TypeScript
2. Implement Keycloak auth store (`lib/stores/auth.ts`)
3. Create API client with token handling (`lib/api/client.ts`)
4. Build catalog browse page (home)
5. Build app detail page with deployment form
6. Build deployments list page
7. Build deployment detail/status page

**Routes**:
```
/                           # Catalog browse
/apps/[appName]             # App detail + deploy form
/deployments                # My deployments
/deployments/[id]           # Deployment detail
```

**Key Files**:
- `frontend/src/lib/stores/auth.ts` - Keycloak integration
- `frontend/src/lib/api/client.ts` - Authenticated API client
- `frontend/src/routes/+page.svelte` - Catalog page
- `frontend/src/routes/apps/[appName]/+page.svelte` - Deploy form

---

### Phase 5: Production Readiness
**Goal**: Testing, logging, deployment

**Tasks**:
1. Add structured logging to all components
2. Add error handling and user-friendly messages
3. Write integration tests
4. Create Helm chart for deploying the App Store itself
5. Documentation

---

## CRD: AppDeployment

```yaml
apiVersion: appstore.bitpipe.no/v1alpha1
kind: AppDeployment
metadata:
  name: my-postgres
  namespace: team-alpha
spec:
  appName: postgresql          # From catalog
  chartVersion: "15.2.0"       # Optional, defaults to latest
  teamId: "team-alpha"
  requestedBy: "user@example.com"
  releaseName: "my-postgres"   # Optional, auto-generated
  values:                      # Helm values override
    primary:
      persistence:
        size: 10Gi
status:
  phase: Deployed              # Pending|Installing|Deployed|Failed
  helmReleaseName: my-postgres
  helmReleaseRevision: 1
  deployedChartVersion: "15.2.0"
```

---

## RabbitMQ Message Format

```json
{
  "type": "deployment.request",
  "id": "uuid",
  "timestamp": "2024-01-15T10:00:00Z",
  "source": "backend-api",
  "payload": {
    "requestId": "uuid",
    "teamId": "team-alpha",
    "userId": "user-123",
    "appName": "postgresql",
    "namespace": "team-alpha",
    "values": { ... }
  }
}
```

---

## First Step: Phase 1 - Operator Foundation

When ready to implement, we'll start with:

1. Create project directory structure
2. Initialize Kubebuilder operator
3. Define AppDeployment CRD types
4. Implement basic Helm client
5. Implement reconciler that installs Helm charts
6. Test with local Kubernetes cluster

This gives us the core functionality before adding messaging and UI layers.

---

## Questions Resolved
- [x] Tech stack: Go operator, Go backend (net/http), Svelte frontend
- [x] Auth: Keycloak (existing, Azure IDP)
- [x] Scope: MVP single cluster, future multi-cluster
- [x] Charts: Predefined org catalog
