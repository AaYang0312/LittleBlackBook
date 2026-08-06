package main

import (
	"log/slog"
	"os"
	"time"
	"xbs/internal/pkg/config"
	"xbs/internal/pkg/db"
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/middleware"
	"xbs/internal/pkg/response"
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
	if err := r.Run(cfg.Server.Addr); err != nil {
		slog.Error("server exit", "err", err)
	}
}
