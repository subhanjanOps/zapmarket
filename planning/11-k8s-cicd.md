# Stage 11 — Kubernetes Readiness & CI/CD

Corresponds to checklist **Phase 16 + Phase 17**.

## Goal

Make the platform deployable to a real cluster and add automated CI gates, now that the
service set and infra (Kafka, Redis, observability) are all finalized from earlier stages.

## Preconditions
- Stages 1-10 merged. Deliberately last functional stage before pure hardening (Stage 12)
  because k8s manifests and CI pipelines churn heavily if the underlying services/infra
  are still changing shape.

## Tasks

### 11.1 Kubernetes manifests
- [ ] `infra/k8s/` per `design.md`'s project structure: one Deployment + Service per
      microservice (6 services + gateway), ConfigMaps for non-secret env vars, Secrets
      for `JWT_SECRET_KEY`, DB passwords, gateway API keys.
- [ ] Each service already has a `Dockerfile` (confirmed for `auth-service` and
      `product-catalog-service`; the four newly-built services from Stages 3-6 need one
      each, matching the existing Dockerfile pattern).
- [ ] StatefulSets or managed-service references for Postgres, Kafka, Redis — decide
      whether dev/staging clusters run these in-cluster (StatefulSet) or rely on managed
      cloud equivalents; this is an infra-cost decision for the user, not a default to
      assume silently.
- [ ] Horizontal Pod Autoscaler (HPA) configs for the gateway and any service with
      variable load (Catalog, Order).
- [ ] Ingress resource routing to the API Gateway only — no service gets a separate
      Ingress, consistent with the single-ingress design from Stage 8.

### 11.2 CI/CD — Pull Request pipeline
- [ ] GitHub Actions workflow: `go vet`, lint (`golangci-lint`), `go test ./...` across
      every module in `go.work`, run on every PR.
- [ ] Security scan (`govulncheck` at minimum; `gosec` if the user wants static analysis
      too) on every PR.

### 11.3 CI/CD — Main branch pipeline
- [ ] Build a Docker image per service on merge to `main`, tagged with commit SHA.
- [ ] Push to whichever registry the user has access to (GHCR is the path of least
      friction for a GitHub-hosted repo with no extra account setup — confirm before
      assuming).
- [ ] Deploy step — only build this once the user confirms a target cluster exists;
      don't generate deploy credentials/secrets workflows speculatively.

## Out of scope
- Multi-environment (staging/prod) promotion workflows — single-environment deploy is
  the starting point; promotion pipelines are a Stage 12+ or later concern once there's
  an actual second environment to promote to.

## Definition of done
- `kubectl apply -k infra/k8s/` (or equivalent) brings up the full platform in a local
  cluster (kind/minikube) with all services reaching `Ready`.
- A PR with a deliberately failing test is blocked from merging by the CI gate.
- A merge to `main` produces tagged images visible in the registry.
