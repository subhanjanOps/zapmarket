package grpc

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	appErr "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/errors"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/service"
	pb "github.com/zapmarket/zapmarket/services/product-catalog-service/proto/productcatalogpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProductCatalogGRPCHandler struct {
	pb.UnimplementedProductCatalogServiceServer
	productSvc service.ProductService
	skuSvc     service.SKUService
	logger     *slog.Logger
}

func NewProductCatalogGRPCHandler(
	productSvc service.ProductService,
	skuSvc service.SKUService,
	logger *slog.Logger,
) *ProductCatalogGRPCHandler {
	return &ProductCatalogGRPCHandler{
		productSvc: productSvc,
		skuSvc:     skuSvc,
		logger:     logger,
	}
}

func (h *ProductCatalogGRPCHandler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	id, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product_id: %s", req.ProductId)
	}

	product, err := h.productSvc.GetProductByID(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetProductResponse{Product: domainProductToProto(product)}, nil
}

func (h *ProductCatalogGRPCHandler) GetSKU(ctx context.Context, req *pb.GetSKURequest) (*pb.GetSKUResponse, error) {
	id, err := uuid.Parse(req.SkuId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sku_id: %s", req.SkuId)
	}

	sku, err := h.skuSvc.GetSKUByID(ctx, id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetSKUResponse{Sku: domainSKUToProto(sku)}, nil
}

func (h *ProductCatalogGRPCHandler) GetSKUsByProduct(ctx context.Context, req *pb.GetSKUsByProductRequest) (*pb.GetSKUsByProductResponse, error) {
	productID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product_id: %s", req.ProductId)
	}

	filters := &domain.SKUFilters{ProductID: &productID}
	if req.ActiveOnly {
		t := true
		filters.IsActive = &t
	}

	skus, err := h.skuSvc.GetSKUList(ctx, filters)
	if err != nil {
		return nil, toGRPCError(err)
	}

	protoSKUs := make([]*pb.SKUProto, 0, len(skus))
	for _, s := range skus {
		protoSKUs = append(protoSKUs, domainSKUToProto(s))
	}

	return &pb.GetSKUsByProductResponse{Skus: protoSKUs}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toGRPCError(err error) error {
	if appError, ok := err.(*appErr.AppError); ok {
		switch appError.Type {
		case appErr.NotFound:
			return status.Error(codes.NotFound, appError.Message)
		case appErr.Validation:
			return status.Error(codes.InvalidArgument, appError.Message)
		case appErr.Conflict:
			return status.Error(codes.AlreadyExists, appError.Message)
		case appErr.Unauthorized:
			return status.Error(codes.PermissionDenied, appError.Message)
		}
	}
	return status.Error(codes.Internal, "internal server error")
}

func domainProductToProto(p *domain.Product) *pb.ProductProto {
	desc := ""
	if p.Description != nil {
		desc = *p.Description
	}
	attrs := ""
	if p.Attributes != nil {
		attrs = string(p.Attributes)
	}
	return &pb.ProductProto{
		Id:          p.ID.String(),
		CategoryId:  p.CategoryID.String(),
		SellerId:    p.SellerID.String(),
		Name:        p.Name,
		Slug:        p.Slug,
		Description: desc,
		Attributes:  attrs,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt.String(),
		UpdatedAt:   p.UpdatedAt.String(),
	}
}

func domainSKUToProto(s *domain.SKU) *pb.SKUProto {
	variantAttrs := ""
	if s.VariantAttrs != nil {
		variantAttrs = string(s.VariantAttrs)
	}

	var comparePrice int64
	if s.ComparePrice != nil {
		comparePrice = *s.ComparePrice
	}

	var weightGrams int32
	if s.WeightGrams != nil {
		weightGrams = *s.WeightGrams
	}

	// Ensure variant_attrs is valid JSON; fall back to "{}" if empty
	if variantAttrs == "" || !json.Valid([]byte(variantAttrs)) {
		variantAttrs = "{}"
	}

	return &pb.SKUProto{
		Id:           s.ID.String(),
		ProductId:    s.ProductID.String(),
		SkuCode:      s.SKUCode,
		VariantAttrs: variantAttrs,
		PriceAmount:  s.PriceAmount,
		ComparePrice: comparePrice,
		Currency:     s.Currency,
		WeightGrams:  weightGrams,
		IsActive:     s.IsActive,
		CreatedAt:    s.CreatedAt.String(),
		UpdatedAt:    s.UpdatedAt.String(),
	}
}
