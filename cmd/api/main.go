package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"streamforge/internal/api"
	"streamforge/internal/auth"
	"streamforge/internal/config"
	"streamforge/internal/database"
	"streamforge/internal/jobs"
	"streamforge/internal/middleware"
	"streamforge/internal/queue"
	"streamforge/internal/redis"
	"streamforge/internal/worker"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	dbPool := database.Connect(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Name:     cfg.Database.Name,
		MaxConns: cfg.Database.MaxConns,
	})
	defer dbPool.Close()

	redisClient := redis.Connect(cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB)
	defer redisClient.Close()

	rabbitConn, err := queue.Connect(cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	repo := database.NewRepository(dbPool.Pool)
	authSvc := auth.NewService(repo, cfg.JWT)
	jobSvc := jobs.NewService(repo, redisClient)
	queueSvc := queue.NewService(rabbitConn, cfg.Queue)

	workerPool := worker.NewPool(cfg.Worker.WorkerCount, jobSvc, redisClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerPool.Start(ctx)
	defer workerPool.Stop()

	go func() {
		if err := queueSvc.Consume(ctx, workerPool.JobChan()); err != nil {
			log.Printf("Queue consumer error: %v", err)
		}
	}()

	router := setupRouter(cfg, authSvc, jobSvc, workerPool, redisClient)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter(
	cfg *config.Config,
	authSvc *auth.Service,
	jobSvc *jobs.Service,
	workerPool *worker.Pool,
	redisClient *redis.Client,
) *gin.Engine {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.RateLimiter(redisClient, cfg.RateLimit))

	api.RegisterRoutes(r, authSvc, jobSvc, workerPool, redisClient)

	return r
}
