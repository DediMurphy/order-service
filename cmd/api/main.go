package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"order-service/internal/config"
	"order-service/internal/handler"
	"order-service/internal/repository"
	"order-service/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := repository.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()
	log.Printf("database siap di %s", cfg.DBPath)

	productRepo := repository.NewProductRepository()
	customerRepo := repository.NewCustomerRepository()
	orderRepo := repository.NewOrderRepository()

	productSvc := service.NewProductService(db, productRepo)
	customerSvc := service.NewCustomerService(db, customerRepo)
	orderSvc := service.NewOrderService(db, orderRepo, productRepo, customerRepo)

	productHandler := handler.NewProductHandler(productSvc)
	customerHandler := handler.NewCustomerHandler(customerSvc)
	orderHandler := handler.NewOrderHandler(orderSvc)

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	handler.RegisterRoutes(r, productHandler, customerHandler, orderHandler)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server jalan di http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}
	log.Println("server stopped")
}
