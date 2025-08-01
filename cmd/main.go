package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"pot-play-storage/internal/config"
	"pot-play-storage/internal/handler"
	"pot-play-storage/internal/repository"
	"pot-play-storage/internal/service"
	"pot-play-storage/pkg/storage"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 로컬 설정 파일이 있으면 사용, 없으면 기본 설정 사용
	if _, err := os.Stat("configs/config.local.yaml"); err == nil {
		viper.SetConfigFile("configs/config.local.yaml")
	} else {
		viper.SetConfigFile("configs/config.yaml")
	}
	if err := viper.ReadInConfig(); err != nil {
		logger.Fatal("config load failed", zap.Error(err))
	}

	db, err := config.NewDBPool(viper.GetStringMap("database"))
	if err != nil {
		logger.Fatal("db connect failed", zap.Error(err))
	}
	defer db.Close()

	cache := config.NewRedis(viper.GetStringMap("redis"))

	storageType := viper.GetString("storage.type")
	var st storage.Storage
	switch storageType {
	case "local":
		st, err = storage.NewLocalStorage(viper.GetString("storage.local_path"))
	case "seaweedfs":
		masterURL := viper.GetString("storage.seaweedfs.master_url")
		// Use simple implementation for development
		st, err = storage.NewSeaweedFSSimpleStorage(masterURL)
	default:
		err = fmt.Errorf("unsupported storage type: %s", storageType)
	}
	if err != nil {
		logger.Fatal("storage init failed", zap.Error(err))
	}

	repo := repository.NewFileRepository(db, cache, logger)
	svc := service.NewStorageService(st, repo, logger)
	hdl := handler.NewFileHandler(svc, logger)

	r := gin.Default()
	
	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "pot-play-storage",
			"version": "1.0.0",
		})
	})
	
	api := r.Group("/api/v1/files")
	{
		api.POST("", hdl.Upload)
		api.GET("/:id", hdl.View)
		api.GET("/:id/download", hdl.Download)
		api.DELETE("/:id", hdl.Delete)
		api.GET("", hdl.List)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", viper.GetInt("server.port")),
		Handler: r,
	}

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error {
		return srv.ListenAndServe()
	})
	g.Go(func() error {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sig:
			return srv.Shutdown(context.Background())
		}
	})

	if err := g.Wait(); err != nil {
		logger.Error("server exit", zap.Error(err))
	}
}