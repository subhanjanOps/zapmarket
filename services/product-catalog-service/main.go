// @title			Product Catalog Service API
// @version		1.0
// @description	Manages products, SKUs, categories, and images for ZapMarket.
// @host			localhost:8081
// @BasePath		/
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Enter: Bearer <token>
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	_ "github.com/zapmarket/zapmarket/services/product-catalog-service/docs"
	grpchandler "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/handler/grpc"
	httpHandler "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/handler/http"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/middleware"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/repository"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/service"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/pkg/config"
	pb "github.com/zapmarket/zapmarket/services/product-catalog-service/proto/productcatalogpb"
)

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to database")

	// ── Auth middleware ────────────────────────────────────────────────────────
	authMW, err := middleware.NewAuthMiddleware(cfg.AuthServiceAddr, logger)
	if err != nil {
		logger.Error("failed to connect to auth-service", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to auth-service", "addr", cfg.AuthServiceAddr)

	// ── Repositories ───────────────────────────────────────────────────────────
	categoryRepo := repository.NewCategoryRepository(db)
	productRepo := repository.NewProductRepository(db)
	skuRepo := repository.NewSkuRepository(db)
	imageRepo := repository.NewProductImageRepository(db)

	// ── Services ───────────────────────────────────────────────────────────────
	categorySvc := service.NewCategoryService(categoryRepo, logger)
	productSvc := service.NewProductService(productRepo, logger)
	skuSvc := service.NewSKUService(skuRepo, logger)
	imageSvc := service.NewProductImageService(imageRepo, logger)

	// ── HTTP handlers ──────────────────────────────────────────────────────────
	categoryH := httpHandler.NewCategoryHandler(categorySvc)
	productH := httpHandler.NewProductHandler(productSvc)
	skuH := httpHandler.NewSKUHandler(skuSvc)
	imageH := httpHandler.NewProductImageHandler(imageSvc)

	// ── HTTP router ────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)

	r.Route("/api/v1", func(r chi.Router) {

		// ── Public: read-only ──────────────────────────────────────────────────
		r.Get("/categories", categoryH.GetCategoryList)
		r.Get("/categories/slug/{slug}", categoryH.GetCategoryBySlug)
		r.Get("/categories/{id}", categoryH.GetCategoryByID)

		r.Get("/products", productH.GetProductList)
		r.Get("/products/slug/{slug}", productH.GetProductBySlug)
		r.Get("/products/{id}", productH.GetProductByID)
		r.Get("/products/{product_id}/images", imageH.GetImagesByProductID)
		r.Get("/products/{product_id}/images/sku/{sku_id}", imageH.GetImagesBySKUID)

		r.Get("/skus", skuH.GetSKUList)
		r.Get("/skus/{id}", skuH.GetSKUByID)

		// ── Seller or Admin: manage products, SKUs, images ─────────────────────
		r.Group(func(r chi.Router) {
			r.Use(authMW.Authenticate)
			r.Use(authMW.RequireRole("seller", "admin"))

			r.Post("/products", productH.CreateProduct)
			r.Put("/products/{id}", productH.UpdateProduct)
			r.Delete("/products/{id}", productH.DeleteProduct)

			r.Post("/products/{product_id}/images", imageH.CreateProductImage)
			r.Patch("/products/{product_id}/images/{id}/position", imageH.UpdateImagePosition)
			r.Delete("/products/{product_id}/images/{id}", imageH.DeleteProductImage)

			r.Post("/skus", skuH.CreateSKU)
			r.Put("/skus/{id}", skuH.UpdateSKU)
			r.Delete("/skus/{id}", skuH.DeleteSKU)
		})

		// ── Admin only: manage categories ──────────────────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(authMW.Authenticate)
			r.Use(authMW.RequireRole("admin"))

			r.Post("/categories", categoryH.CreateCategory)
			r.Put("/categories/{id}", categoryH.UpdateCategory)
			r.Delete("/categories/{id}", categoryH.DeleteCategory)
		})
	})

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── gRPC server ────────────────────────────────────────────────────────────
	grpcHandler := grpchandler.NewProductCatalogGRPCHandler(productSvc, skuSvc, logger)
	grpcServer := googlegrpc.NewServer()
	pb.RegisterProductCatalogServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		logger.Error("failed to listen for gRPC", "error", err)
		os.Exit(1)
	}

	// ── Start servers ──────────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("starting HTTP server", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	go func() {
		logger.Info("starting gRPC server", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Error("gRPC server error", "error", err)
		}
	}()

	<-quit
	logger.Info("shutting down servers")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}
	grpcServer.GracefulStop()

	logger.Info("servers stopped")
}
