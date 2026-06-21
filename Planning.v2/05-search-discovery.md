# Phase 5 — Search & Discovery

**Goal:** Enable full-text product search with category facets, price range filters, and relevance ranking via Elasticsearch. The current `ILIKE` search in `product-catalog-service` is not scalable beyond tens of thousands of products.

**Current state:** Elasticsearch config exists in `docker-compose.yml` but is commented out. The `product-catalog-service` uses `name ILIKE $1 OR description ILIKE $1` in the DB query.

---

## 5.1 Architecture

```
Product created/updated/deleted
           │
           ▼ (via Kafka outbox events)
[search-sync worker in product-catalog-service]
     Consumes: product.created, product.updated, product.deleted
           │
           ▼
[Elasticsearch index: zapmarket_products]
           │
           ▼
GET /v1/products/search?q=&category=&min_price=&max_price=&sort=
           │ (new search endpoint in product-catalog-service)
           ▼
[Elasticsearch query: multi_match + bool filter + aggregations]
```

The sync worker lives inside `product-catalog-service` as a Kafka consumer goroutine, keeping service boundaries clean. No separate search service is needed at this scale.

---

## 5.2 Elasticsearch Index Design

**Index name:** `zapmarket_products`

**Mapping:**

```json
{
  "mappings": {
    "properties": {
      "id":           { "type": "keyword" },
      "seller_id":    { "type": "keyword" },
      "category_id":  { "type": "keyword" },
      "category_path":{ "type": "keyword" },
      "name":         { "type": "text", "analyzer": "standard", "fields": { "keyword": { "type": "keyword" } } },
      "description":  { "type": "text", "analyzer": "standard" },
      "slug":         { "type": "keyword" },
      "status":       { "type": "keyword" },
      "price_min":    { "type": "long" },
      "price_max":    { "type": "long" },
      "attributes":   { "type": "object", "dynamic": true },
      "in_stock":     { "type": "boolean" },
      "created_at":   { "type": "date" },
      "updated_at":   { "type": "date" }
    }
  },
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  }
}
```

`price_min`/`price_max` are populated from the cheapest/most expensive active SKU for that product (denormalized at index time).

`category_path` stores all ancestor IDs (e.g. `["root-id", "clothing-id", "t-shirts-id"]`) for hierarchical category facets without recursive DB queries.

---

## 5.3 Sync Worker

Add a Kafka consumer to `product-catalog-service` that subscribes to product events from the outbox:

```go
// internal/search/syncer.go
type Syncer struct {
    es     *elasticsearch.Client
    repo   contracts.ProductRepository
    logger *slog.Logger
}

func (s *Syncer) Handle(ctx context.Context, msg kafka.Message) error {
    switch msg.Headers["event_type"] {
    case "product.created", "product.updated":
        return s.upsert(ctx, msg)
    case "product.deleted":
        return s.delete(ctx, msg)
    }
    return nil
}
```

Product outbox events need to be added (currently `product-catalog-service` doesn't write outbox rows). Add `insertOutboxRow` to `CreateProduct`, `UpdateProduct`, and `DeleteProduct` repository methods. This requires the outbox relay from Phase 2.

**Alternative for Phase 5 (if Phase 2 isn't done):** Sync inline in the service layer after each write. Less resilient but unblocks search without Kafka.

---

## 5.4 Search Endpoint

**New endpoint:** `GET /v1/products/search`

Query parameters:

| Param | Type | Description |
|-------|------|-------------|
| `q` | string | Full-text search query |
| `category_id` | UUID | Filter by category (includes subcategories via `category_path`) |
| `min_price` | int | Min price in smallest unit |
| `max_price` | int | Max price in smallest unit |
| `in_stock` | bool | Only show in-stock products |
| `sort` | string | `relevance` (default), `price_asc`, `price_desc`, `newest` |
| `limit` | int | Default 20, max 100 |
| `offset` | int | Pagination offset |

**Elasticsearch query structure:**

```json
{
  "query": {
    "bool": {
      "must": [
        { "multi_match": { "query": "$q", "fields": ["name^3", "description"] } }
      ],
      "filter": [
        { "term": { "status": "ACTIVE" } },
        { "term": { "category_path": "$category_id" } },
        { "range": { "price_min": { "gte": "$min_price", "lte": "$max_price" } } }
      ]
    }
  },
  "aggs": {
    "categories": { "terms": { "field": "category_id", "size": 20 } },
    "price_ranges": { "range": { "field": "price_min", "ranges": [...] } }
  },
  "sort": [...]
}
```

Response includes `aggregations` for facet counts alongside the hit list.

---

## 5.5 Initial Index Population

On first enable, backfill the index from the DB:

```go
// cmd/reindex/main.go (one-shot tool)
func main() {
    // SELECT all ACTIVE products in batches of 500
    // Bulk index to Elasticsearch
}
```

---

## 5.6 docker-compose changes

Uncomment Elasticsearch service. Set `ES_JAVA_OPTS=-Xms256m -Xmx256m` for dev (halved from the commented config).

Add `ELASTICSEARCH_URL=http://zapmarket-elasticsearch:9200` to `product-catalog-service` environment.

---

## 5.7 buyer-ui considerations

There is no buyer-facing UI in the current codebase. Search is exposed via the api-gateway and can be consumed by any future buyer app or a new `buyer-ui` service. The seller-ui `products` page can also switch to the search endpoint for its own product list with the `seller_id` filter.

---

## Acceptance criteria

- Searching "blue shirt" returns products with "blue" in name or description, ranked by relevance
- Filtering by category includes all subcategories
- Index is updated within 5 seconds of a product status change
- `GET /v1/products/search` p99 latency < 100ms for a 50k-product catalogue
- Facet counts are accurate

---

## Estimated effort

3 weeks (ES schema + sync worker: 1 week; search endpoint: 1 week; buyer-facing UI integration: 1 week).
