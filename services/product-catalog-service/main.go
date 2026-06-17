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
	goredis "github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"google.golang.org/grpc/reflection"

	"github.com/zapmarket/zapmarket/pkg/config"
	"github.com/zapmarket/zapmarket/pkg/database"
	"github.com/zapmarket/zapmarket/pkg/grpcx"
	"github.com/zapmarket/zapmarket/pkg/logger"
	"github.com/zapmarket/zapmarket/pkg/migrate"
	pb "github.com/zapmarket/zapmarket/pkg/proto/catalog"
	"github.com/zapmarket/zapmarket/pkg/storage"
	"github.com/zapmarket/zapmarket/pkg/swaggerx"
	_ "github.com/zapmarket/zapmarket/services/product-catalog-service/docs"
	grpchandler "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/handler/grpc"
	httpHandler "github.com/zapmarket/zapmarket/services/product-catalog-service/internal/handler/http"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/middleware"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/repository"
	"github.com/zapmarket/zapmarket/services/product-catalog-service/internal/service"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv)

	db, err := database.New(cfg)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("connected to database")

	if cfg.MigrateOnBoot {
		if err := migrate.Up(cfg, "migrations"); err != nil {
			log.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
		log.Info("migrations applied")
	}

	// ── Auth middleware ────────────────────────────────────────────────────────
	authMW, err := middleware.NewAuthMiddleware(cfg.AuthServiceAddr, log)
	if err != nil {
		log.Error("failed to connect to auth-service", "error", err)
		os.Exit(1)
	}
	log.Info("connected to auth-service", "addr", cfg.AuthServiceAddr)

	// ── Object storage ────────────────────────────────────────────────────────
	objectStorage, err := storage.New(context.Background(), cfg)
	if err != nil {
		log.Error("failed to connect to object storage", "error", err)
		os.Exit(1)
	}
	log.Info("connected to object storage", "endpoint", cfg.MinIOEndpoint, "bucket", cfg.MinIOBucket)

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb := goredis.NewClient(&goredis.Options{Addr: cfg.RedisURL})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Warn("Redis unavailable — product cache disabled", "addr", cfg.RedisURL, "error", err)
		rdb = nil
	} else {
		defer rdb.Close()
		log.Info("connected to Redis", "addr", cfg.RedisURL)
	}

	// ── Repositories ───────────────────────────────────────────────────────────
	categoryRepo := repository.NewCategoryRepository(db)
	productRepo := repository.NewProductRepository(db)
	skuRepo := repository.NewSkuRepository(db)
	imageRepo := repository.NewProductImageRepository(db)

	// ── Services ───────────────────────────────────────────────────────────────
	categorySvc := service.NewCategoryService(categoryRepo, log)
	productSvc := service.NewProductService(productRepo, log)
	if rdb != nil {
		productSvc = service.NewCachedProductService(productSvc, rdb, log)
	}
	skuSvc := service.NewSKUService(skuRepo, log)
	imageSvc := service.NewProductImageService(imageRepo, objectStorage, log)

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

	// Swagger: spec served from the embedded swag doc (see docs/docs.go,
	// regenerated via `swag init -g main.go`), not a file on disk.
	// See pkg/swaggerx for why these two routes are split.
	r.Get("/v1/docs/swagger.json", swaggerx.JSONHandler(""))
	r.Get("/v1/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/v1/docs/swagger.json"),
	))

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── gRPC server ────────────────────────────────────────────────────────────
	grpcHandler := grpchandler.NewProductCatalogGRPCHandler(productSvc, skuSvc, log)
	grpcServer := grpcx.NewServer()
	pb.RegisterProductCatalogServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Error("failed to listen for gRPC", "error", err)
		os.Exit(1)
	}

	// ── Start servers ──────────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("starting HTTP server", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
		}
	}()

	go func() {
		log.Info("starting gRPC server", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Error("gRPC server error", "error", err)
		}
	}()

	<-quit
	log.Info("shutting down servers")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("HTTP server shutdown error", "error", err)
	}
	grpcServer.GracefulStop()

	log.Info("servers stopped")
}
