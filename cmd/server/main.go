package main

import (
	"log/slog"
	"os"
	"time"
	"xbs/internal/feed"
	"xbs/internal/interaction"
	"xbs/internal/note"
	"xbs/internal/pkg/cache"
	"xbs/internal/pkg/config"
	"xbs/internal/pkg/db"
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/middleware"
	"xbs/internal/pkg/mq"
	"xbs/internal/pkg/response"
	"xbs/internal/pkg/snowflake"
	"xbs/internal/pkg/storage"
	"xbs/internal/user"

	"github.com/gin-gonic/gin"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		slog.Error("加载配置文件出错", "err", err)
		os.Exit(1)
	}
	gormDB, err := db.NewMySQL(cfg.MySQL.DSN)
	if err != nil {
		slog.Error("连接MySQL出错", "err", err)
		os.Exit(1)
	}
	rdb := db.NewRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Logger(), gin.Recovery())
	r.GET("healthz", func(c *gin.Context) {
		sqlDB, _ := gormDB.DB()
		if err := sqlDB.Ping(); err != nil {
			response.Fail(c, errs.ErrInternal)
			return
		}
		if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
			response.Fail(c, errs.ErrInternal)
			return
		}
		response.OK(c, gin.H{"status": "ok"})
		//response.OK(c, gin.H{"status": "ok"})
	})
	userSvc := user.NewService(user.NewRepository(gormDB), cfg.JWT.Secret, time.Duration(cfg.JWT.ExpireHours)*time.Hour)
	user.RegisterRoutes(r.Group("/api/v1"), user.NewHandler(userSvc), cfg.JWT.Secret)
	m, err := mq.New(cfg.RabbitMQ.URL)
	if err != nil {
		slog.Error("连接 rabbitmq", "err", err)
		os.Exit(1)
	}
	defer m.Close()
	st, err := storage.NewMinIO(cfg.MinIO.Endpoint, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, cfg.MinIO.Bucket, cfg.MinIO.UseSSL)
	if err != nil {
		slog.Error("连接 MinIO", "err", err)
		os.Exit(1)
	}
	if err := snowflake.Init(cfg.Snowflake.Node); err != nil {
		slog.Error("初始化 Snowflake", "err", err)
		os.Exit(1)
	}
	noteSvc := note.NewService(note.NewRepository(gormDB), st, m, rdb, cache.New(rdb))
	note.RegisterRoutes(r.Group("/api/v1"), note.NewHandler(noteSvc), cfg.JWT.Secret)
	interactionSvc := interaction.NewService(interaction.NewRepository(gormDB), rdb, m, noteSvc)
	interaction.RegisterRoutes(r.Group("/api/v1"), interaction.NewHandler(interactionSvc), cfg.JWT.Secret)
	feedSvc := feed.NewService(rdb, noteSvc, interactionSvc, 500)
	feed.RegisterRoutes(r.Group("/api/v1"), feed.NewHandler(feedSvc), cfg.JWT.Secret)
	if err := r.Run(cfg.Server.Addr); err != nil {
		slog.Error("server exit", "err", err)
	}
}
