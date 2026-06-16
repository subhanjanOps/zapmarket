// Package swaggerx standardizes how every service exposes its swag-generated
// OpenAPI spec.
//
// Convention used by every service in this repo:
//   - `swag init` generates `docs/docs.go`, which self-registers its spec
//     with the swaggo/swag package via an init() function. Each service's
//     main.go must blank-import that docs package so the registration runs.
//   - The spec is served as JSON at the literal path "<docs-prefix>/swagger.json"
//     via JSONHandler, registered as an explicit route.
//   - The interactive UI (index.html, swagger-ui-bundle.js, ...) is served by
//     swaggo/http-swagger's Handler, mounted as a catch-all at "<docs-prefix>/*"
//     (or "<docs-prefix>/" for net/http's ServeMux), configured with
//     httpSwagger.URL("<docs-prefix>/swagger.json") so the UI fetches the
//     spec from the literal path above.
//
// This split exists because swaggo/http-swagger's own spec-serving route is
// hardcoded to "doc.json" (not configurable) — see its Handler source, the
// `case "doc.json":` branch. Routing the literal "/swagger.json" path through
// JSONHandler first (registered as a more specific route than the "/*"
// catch-all) keeps the public URL identical across every service regardless
// of that library's internal naming.
package swaggerx

import (
	"net/http"

	"github.com/swaggo/swag"
)

// JSONHandler serves the swag-generated OpenAPI spec registered under
// instanceName (pass "" for the default instance — the common case, one
// spec per service) as JSON.
func JSONHandler(instanceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, err := swag.ReadDoc(instanceName)
		if err != nil {
			http.Error(w, "failed to load API definition: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(doc))
	}
}
