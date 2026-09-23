package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agridispatch/agridispatch/internal/config"
	"github.com/agridispatch/agridispatch/internal/database"
	"github.com/agridispatch/agridispatch/internal/logger"
	"github.com/agridispatch/agridispatch/internal/repository"
	"github.com/agridispatch/agridispatch/internal/router"
	"github.com/agridispatch/agridispatch/internal/service"
	"github.com/agridispatch/agridispatch/internal/ws"
	"github.com/redis/go-redis/v9"
)

func main() {
	log := logger.New()
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		log.Error("load config failed", "err", err)
		os.Exit(1)
	}

	db, err := database.Connect(cfg.DSN(), cfg.DBMaxOpenConns, cfg.DBMaxIdleConns, cfg.DBConnMaxLifetime, cfg.DBRetryCount, cfg.DBRetryInterval)
	if err != nil {
		log.Error("connect database failed", "err", err)
		os.Exit(1)
	}
	if err := database.Seed(db); err != nil {
		log.Error("seed database failed", "err", err)
		os.Exit(1)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPass,
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Error("connect redis failed", "err", err)
		os.Exit(1)
	}
	cancel()
	log.Info("redis connected", "addr", cfg.RedisAddr())

	userRepo := repository.NewUserRepository(db)
	dashboardRepo := repository.NewDashboardRepository(db)
	dispatchRepo := repository.NewDispatchRepository(db)

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpire, log)
	dashboardSvc := service.NewDashboardService(dashboardRepo, redisClient, log)
	dispatchSvc := service.NewDispatchService(dispatchRepo, redisClient, log)
	hub := ws.NewHub(log)

	engine := router.Setup(db, redisClient, authSvc, dashboardSvc, dispatchSvc, hub, cfg, log)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server starting", "port", cfg.ServerPort, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server listen failed", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	_ = redisClient.Close()
	log.Info("server exited")
}
