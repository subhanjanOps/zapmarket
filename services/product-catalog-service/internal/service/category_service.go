package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/domain"
	appErr "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/errors"
)

//go:generate mockgen -source=category_service.go -destination=../mocks/category_service.go -package=mocks

type CategoryService interface {
	CreateCategory(ctx context.Context, category *domain.Category) error
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error)
	GetCategoryList(ctx context.Context, filters map[string]string, limit, offset int) ([]*domain.Category, error)
	UpdateCategory(ctx context.Context, category *domain.Category) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}

type categoryService struct {
	categoryRepo CategoryRepository
	logger       *slog.Logger
}

func NewCategoryService(repo CategoryRepository, logger *slog.Logger) CategoryService {
	return &categoryService{
		categoryRepo: repo,
		logger:       logger,
	}
}

func (cs *categoryService) CreateCategory(ctx context.Context, category *domain.Category) error {
	if category.Slug == "" {
		return appErr.ValidationError("category slug is required", nil)
	}

	if category.Name == "" {
		return appErr.ValidationError("category name is required", nil)
	}

	cs.logger.Info("creating category", "slug", category.Slug, "name", category.Name)

	return cs.categoryRepo.CreateCategory(ctx, category)
}

func (cs *categoryService) GetCategoryByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	if id == uuid.Nil {
		return nil, appErr.ValidationError("category id is required", nil)
	}

	cs.logger.Info("fetching category by id", "id", id)

	return cs.categoryRepo.GetCategoryByID(ctx, id)
}

func (cs *categoryService) GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	if slug == "" {
		return nil, appErr.ValidationError("category slug is required", nil)
	}

	cs.logger.Info("fetching category by slug", "slug", slug)

	return cs.categoryRepo.GetCategoryBySlug(ctx, slug)
}

func (cs *categoryService) GetCategoryList(ctx context.Context, filters map[string]string, limit, offset int) ([]*domain.Category, error) {
	cs.logger.Info("fetching category list", "limit", limit, "offset", offset)

	return cs.categoryRepo.GetCategoryList(ctx, filters, limit, offset)
}

func (cs *categoryService) UpdateCategory(ctx context.Context, category *domain.Category) error {
	if category.ID == uuid.Nil {
		return appErr.ValidationError("category id is required", nil)
	}

	if category.Name == "" {
		return appErr.ValidationError("category name is required", nil)
	}

	if category.Slug == "" {
		return appErr.ValidationError("category slug is required", nil)
	}

	cs.logger.Info("updating category", "id", category.ID, "slug", category.Slug)

	return cs.categoryRepo.UpdateCategory(ctx, category)
}

func (cs *categoryService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return appErr.ValidationError("category id is required", nil)
	}

	cs.logger.Info("deleting category", "id", id)

	return cs.categoryRepo.DeleteCategory(ctx, id)
}
