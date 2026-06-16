# Stage 2b — Product Image Upload via MinIO

**Priority: inserted ahead of Stage 3 at explicit user request (2026-06-16).**
Not in the original checklist phases — this is new scope, sequenced here because
`product-catalog-service`'s image endpoints already exist (Stage 2 territory) but only
accept a pre-existing URL string; there's no way to actually upload a file today.

## Goal

`POST /api/v1/products/{product_id}/images` accepts a real image file (multipart
upload), stores it in MinIO (the bucket already provisioned by `minio-init` in
`docker-compose.yml`), and persists the resulting public URL — replacing the current
"client must already have a URL" contract.

## Preconditions
- MinIO + `minio-init` already in `docker-compose.yml` (done) — the `zapmarket` bucket
  exists by the time `product-catalog-service` starts, since `minio-init` is a one-shot
  job gated on `minio`'s healthcheck.
- Stage 1 conventions (domain contracts, migrations) — this feature must follow them,
  not bypass them.

## Current state (verified against repo, 2026-06-16)

- `domain.ProductImage` has only `ID, ProductID, SKUId, URL, Position, ...` — no object
  key, no content type, no size.
- `CreateProductImageRequest` is JSON `{product_id, sku_id, url}` — the caller supplies
  a URL directly; nothing is actually uploaded anywhere.
- No `pkg/storage` (or equivalent) exists. No MinIO client dependency anywhere in the
  Go code yet — only the infra container.
- No `MINIO_*` env vars in `pkg/config` or any service's `.env.example`.

## Tasks

### 2b.1 `pkg/storage` — MinIO/S3 client wrapper
- [ ] New module `pkg/storage` (own `go.mod`, added to `go.work`), wrapping
      `github.com/minio/minio-go/v7`. Use the real MinIO SDK rather than hand-rolling S3
      signing — it talks the S3 API, so swapping MinIO for real AWS S3 later is a config
      change, not a rewrite.
- [ ] `storage.New(cfg)` — constructs a client from endpoint/access key/secret/TLS flag.
- [ ] `storage.Client.Upload(ctx, key string, r io.Reader, size int64, contentType string) (objectKey string, err error)`.
- [ ] `storage.Client.Delete(ctx, key string) error`.
- [ ] `storage.Client.PublicURL(key string) string` — builds the externally-reachable
      URL for an object (`{MINIO_PUBLIC_URL}/{bucket}/{key}`), kept separate from the
      internal Docker-network endpoint used for the upload itself (`minio:9000` inside
      Compose vs. `localhost:9000` from a browser/Swagger UI on the host).

### 2b.2 Config
- [ ] Add to `pkg/config`: `MinIOEndpoint` (internal, e.g. `minio:9000` in Docker),
      `MinIOPublicURL` (external, e.g. `http://localhost:9000`), `MinIOAccessKey`,
      `MinIOSecretKey`, `MinIOBucket` (default `zapmarket`), `MinIOUseSSL` (default
      `false` for local dev).
- [ ] Add the same vars to `product-catalog-service/.env.example` and the
      `product-catalog-service` block in `docker-compose.yml`.

### 2b.3 Bucket public-read policy
- [ ] Extend the existing `minio-init` one-shot job in `docker-compose.yml` to also run
      `mc anonymous set download local/zapmarket` after bucket creation, so uploaded
      product images are servable directly via their URL without auth — consistent with
      "product images are public catalog content," same trust model as the rest of the
      read-only catalog endpoints.

### 2b.4 Domain & migration
- [ ] Add `ObjectKey string` to `domain.ProductImage` (nullable on existing rows) so
      deletion can target the exact MinIO object instead of trying to parse a key back
      out of a URL — URLs are a derived/display concern, the object key is the source of
      truth for storage operations.
- [ ] Migration `0002_add_object_key.up/down.sql` in
      `services/product-catalog-service/migrations/`: `ALTER TABLE product_images ADD
      COLUMN object_key TEXT;` (down: `DROP COLUMN`).

### 2b.5 Contracts
- [ ] Add `contracts.ObjectStorage` interface (`Upload`, `Delete`) to
      `internal/domain/contracts` — `productImageService` depends on this interface, not
      on the concrete `pkg/storage.Client`, per the Stage 1 convention.

### 2b.6 Service layer
- [ ] `productImageService.CreateProductImage` becomes upload-aware: takes the raw file
      bytes/reader + content type (not a finished `domain.ProductImage` with a URL
      already set), generates an object key (`products/{product_id}/{uuid}{ext}`),
      calls `ObjectStorage.Upload`, builds the public URL, then persists the row.
- [ ] `productImageService.DeleteProductImage`: fetch the row first (need its
      `object_key`), call `ObjectStorage.Delete` (best-effort — log on failure rather
      than blocking the DB delete; an orphaned MinIO object is a cheap cleanup problem,
      a DB row that can't be deleted because storage hiccuped is a worse one), then soft-delete the row.
- [ ] Validate content type against an allow-list (`image/png`, `image/jpeg`,
      `image/webp`, `image/gif`) and reject anything else with
      `pkgerrors.NewValidation`.

### 2b.7 HTTP handler
- [ ] `CreateProductImage` changes from JSON-body to `multipart/form-data`: parse with
      `r.ParseMultipartForm` (cap via `http.MaxBytesReader`, e.g. 5MB limit — reject
      oversized uploads before they hit storage), read `sku_id` as an optional form
      field, read the `file` part, sniff content type (`http.DetectContentType` on the
      first 512 bytes, not trusting the client-supplied `Content-Type` header alone).
- [ ] Update the Swagger annotations: `@Accept mpfd`, `@Param file formData file true`.
- [ ] Keep the response shape identical (`domain.ProductImage` with `url` populated) so
      existing consumers of the read endpoints (`GetImagesByProductID` etc.) don't need
      to change.

### 2b.8 Wiring
- [ ] `main.go`: construct `storage.New(cfg)`, pass into
      `service.NewProductImageService(imageRepo, storageClient, log)`.

## Out of scope
- Image resizing/thumbnailing — store the original as uploaded; a resize pipeline is a
  separate, larger feature if ever needed.
- CDN/caching in front of MinIO — fine for local/dev; revisit if/when this goes to a
  real environment.
- Multi-file batch upload in one request — one file per request, matching the existing
  one-image-per-call endpoint shape.

## Definition of done
- `POST /api/v1/products/{product_id}/images` with a real `multipart/form-data` image
  file returns `201` with a `domain.ProductImage` whose `url` is a working, directly
  fetchable link to the uploaded file (verify with a plain `curl` of the returned URL,
  no auth header, returns the image bytes with `200`).
- The object actually exists in the `zapmarket` MinIO bucket under
  `products/{product_id}/...` (verify via `mc ls local/zapmarket --recursive` or the
  MinIO console at `localhost:9001`).
- Uploading a non-image file (e.g. a `.txt`) is rejected with `400 INVALID_CONTENT_TYPE`
  before anything is written to MinIO.
- Uploading a file over the size cap is rejected with `400` before the full body is
  read into memory.
- `DeleteProductImage` removes both the DB row and the MinIO object — verify the object
  is gone from the bucket after deletion.
- `go build ./...` and `go vet ./...` pass clean for `product-catalog-service` and the
  new `pkg/storage` module.
