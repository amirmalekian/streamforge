package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"streamforge/internal/config"
	"streamforge/internal/database"
	"streamforge/internal/downloader"
	"streamforge/internal/jobs"
	"streamforge/internal/queue"
	"streamforge/internal/redis"
	"streamforge/internal/worker"
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
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("failed to close redis client: %v", err)
		}
	}()

	rabbitConn, err := queue.Connect(cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer func() {
		if err := rabbitConn.Close(); err != nil {
			log.Printf("failed to close rabbitmq connection: %v", err)
		}
	}()

	repo := database.NewRepository(dbPool.Pool)
	jobSvc := jobs.NewService(repo, redisClient)
	queueSvc := queue.NewService(rabbitConn, cfg.Queue)

	workerDownloader := downloader.NewYTDLPDownloader("/tmp/streamforge")
	workerPool := worker.NewPool(cfg.Worker.WorkerCount, jobSvc, redisClient, workerDownloader)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workerPool.Start(ctx)
	defer workerPool.Stop()

	go func() {
		if err := queueSvc.Consume(ctx, workerPool.JobChan()); err != nil {
			log.Printf("Queue consumer error: %v", err)
		}
	}()

	log.Printf("Worker started with %d workers", cfg.Worker.WorkerCount)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down worker...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		workerPool.Stop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Worker stopped gracefully")
	case <-shutdownCtx.Done():
		log.Println("Worker forced to shutdown")
	}
}
