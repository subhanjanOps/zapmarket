package grpc

import (
	"context"
	"errors"
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
	listCurrencies usecases.CurrencyLister
	getRates       usecases.RatesGetter
	log            *slog.Logger
}

func NewCurrencyServer(
	listCurrencies usecases.CurrencyLister,
	getRates usecases.RatesGetter,
	log *slog.Logger,
) *CurrencyServer {
	return &CurrencyServer{
		listCurrencies: listCurrencies,
		getRates:       getRates,
		log:            log,
	}
}

// Register registers the CurrencyService with the given gRPC server.
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
		var staleErr *usecases.ErrRatesTooStale
		if errors.As(err, &staleErr) {
			return nil, status.Errorf(codes.Unavailable, "exchange rates too stale: %v", err)
		}
		s.log.Error("grpc GetRates", "error", err, "base", base)
		return nil, status.Errorf(codes.Internal, "get rates: %v", err)
	}
	return &currencypb.GetRatesResponse{
		Base:  ratesDTO.Base,
		AsOf:  ratesDTO.AsOf.Format("2006-01-02T15:04:05Z"),
		Date:  ratesDTO.Date,
		Stale: ratesDTO.Stale,
		Rates: ratesDTO.Rates,
	}, nil
}
