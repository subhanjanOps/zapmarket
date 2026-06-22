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

// CurrencyServer implements the CurrencyService gRPC methods.
type CurrencyServer struct {
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

// Register registers the CurrencyService with the given gRPC server using ServiceDesc.
// This approach does not require protoc-generated registration code.
func (s *CurrencyServer) Register(srv *grpc.Server) {
	srv.RegisterService(&CurrencyServiceDesc, s)
}

// CurrencyServiceDesc is the gRPC service descriptor for CurrencyService.
// Matches the currency.proto definition.
var CurrencyServiceDesc = grpc.ServiceDesc{
	ServiceName: "currency.CurrencyService",
	HandlerType: (*CurrencyServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListCurrencies",
			Handler:    listCurrenciesHandler,
		},
		{
			MethodName: "GetRates",
			Handler:    getRatesHandler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/currency.proto",
}

func listCurrenciesHandler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	req := new(currencypb.ListCurrenciesRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*CurrencyServer).handleListCurrencies(ctx, req)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/currency.CurrencyService/ListCurrencies"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(*CurrencyServer).handleListCurrencies(ctx, req.(*currencypb.ListCurrenciesRequest))
	}
	return interceptor(ctx, req, info, handler)
}

func getRatesHandler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	req := new(currencypb.GetRatesRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(*CurrencyServer).handleGetRates(ctx, req)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/currency.CurrencyService/GetRates"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(*CurrencyServer).handleGetRates(ctx, req.(*currencypb.GetRatesRequest))
	}
	return interceptor(ctx, req, info, handler)
}

func (s *CurrencyServer) handleListCurrencies(ctx context.Context, _ *currencypb.ListCurrenciesRequest) (*currencypb.ListCurrenciesResponse, error) {
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

func (s *CurrencyServer) handleGetRates(ctx context.Context, req *currencypb.GetRatesRequest) (*currencypb.GetRatesResponse, error) {
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
