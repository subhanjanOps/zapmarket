// Package currencypb contains the Go message types for the currency gRPC service.
// These types match the currency.proto definition and are used for JSON-encoded gRPC
// until protoc-generated stubs replace this file.
//
// To regenerate from proto (requires protoc + protoc-gen-go + protoc-gen-go-grpc):
//   cd services/currency-service/proto
//   protoc --go_out=. --go-grpc_out=. currency.proto
package currencypb

// CurrencyProto is the gRPC message for a single currency entry.
type CurrencyProto struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Flag     string `json:"flag"`
	Decimals int32  `json:"decimals"`
}

// ListCurrenciesRequest is the request message for ListCurrencies.
type ListCurrenciesRequest struct{}

// ListCurrenciesResponse is the response message for ListCurrencies.
type ListCurrenciesResponse struct {
	Currencies []*CurrencyProto `json:"currencies"`
}

// GetRatesRequest is the request message for GetRates.
type GetRatesRequest struct {
	Base string `json:"base"`
}

// GetRatesResponse is the response message for GetRates.
type GetRatesResponse struct {
	Base  string             `json:"base"`
	AsOf  string             `json:"as_of"`
	Date  string             `json:"date"`
	Stale bool               `json:"stale"`
	Rates map[string]float64 `json:"rates"`
}
