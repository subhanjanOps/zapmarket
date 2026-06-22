package grpc

import (
	"context"
	"encoding/json"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zapmarket/zapmarket/services/currency-service/application/usecases"
	"github.com/zapmarket/zapmarket/services/currency-service/proto/currencypb"
)

// CurrencyServer implements the generated CurrencyServiceServer interface.
type CurrencyServer struct {
	currencypb.UnimplementedCurrencyServiceServer
	listCurrencies *usecases.ListCurrenciesUseCase
	getRates       *usecases.GetRatesUseCase
	log            *slog.Logger
}

func NewCurrencyServer(
	listCurrencies *usecases.ListCurrenciesUseCase,
	getRates *usecases.GetRatesUseCase,
	log *slog.Logger,
) *CurrencyServer {
	return &CurrencyServer{
		listCurrencies: listCurrencies,
		getRates:       getRates,
		log:            log,
	}
}

// Register registers the CurrencyService with the given gRPC server using the generated descriptor.
func (s *CurrencyServer) Register(srv *grpc.Server) {
	currencypb.RegisterCurrencyServiceServer(srv, s)
}

func (s *CurrencyServer) ListCurrencies(ctx context.Context, _ *currencypb.ListCurrenciesRequest) (*currencypb.ListCurrenciesResponse, error) {
	currencies, err := s.listCurrencies.Execute(ctx)
	if err != nil {
		s.log.Error("grpc ListCurrencies", "error", err)
		return nil, status.Errorf(codes.Internal, "list currencies: %v", err)
	}
	resp := &currencypb.ListCurrenciesResponse{
		Currencies: make([]*currencypb.CurrencyProto, len(currencies)),
	}
	for i, c := range currencies {
		resp.Currencies[i] = &currencypb.CurrencyProto{
			Code:     c.Code,
			Name:     c.Name,
			Flag:     c.Flag,
			Decimals: int32(c.Decimals),
		}
	}
	return resp, nil
}

func (s *CurrencyServer) GetRates(ctx context.Context, req *currencypb.GetRatesRequest) (*currencypb.GetRatesResponse, error) {
	base := req.Base
	if base == "" {
		base = "USD"
	}
	ratesDTO, err := s.getRates.Execute(ctx, base)
	if err != nil {
		s.log.Error("grpc GetRates", "error", err, "base", base)
		return nil, status.Errorf(codes.Unavailable, "get rates: %v", err)
	}
	return &currencypb.GetRatesResponse{
		Base:  ratesDTO.Base,
		AsOf:  ratesDTO.AsOf.Format("2006-01-02T15:04:05Z"),
		Date:  ratesDTO.Date,
		Stale: ratesDTO.Stale,
		Rates: ratesDTO.Rates,
	}, nil
}

// jsonCodec is used to register JSON encoding for gRPC when no proto codec is present.
// Call RegisterJSONCodec before creating the gRPC server when using this package.
var _ = json.Marshal // ensure encoding/json is imported
