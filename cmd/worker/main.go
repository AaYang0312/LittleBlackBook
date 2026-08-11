package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"xbs/internal/feed"
	"xbs/internal/interaction"
	"xbs/internal/note"
	"xbs/internal/pkg/cache"
	"xbs/internal/pkg/config"
	"xbs/internal/pkg/db"
	"xbs/internal/pkg/mq"
	"xbs/internal/pkg/snowflake"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	if err := snowflake.Init(cfg.Snowflake.Node); err != nil {
		slog.Error("init snowflake", "err", err)
		os.Exit(1)
	}
	gormDB, err := db.NewMySQL(cfg.MySQL.DSN)
	if err != nil {
		slog.Error("connect mysql", "err", err)
		os.Exit(1)
	}
	rdb := db.NewRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	m, err := mq.New(cfg.RabbitMQ.URL)
	if err != nil {
		slog.Error("connect rabbitmq", "err", err)
		os.Exit(1)
	}
	defer m.Close()

	noteSvc := note.NewService(note.NewRepository(gormDB), nil, nil, rdb, cache.New(rdb))
	interactionSvc := interaction.NewService(interaction.NewRepository(gormDB), rdb, m, noteSvc)

	if err := m.Consume(mq.QueueLikeEvent, func(body []byte) error {
		var ev mq.LikeEvent
		if err := json.Unmarshal(body, &ev); err != nil {
			return err // 坏消息重试 3 次后进 DLQ
		}
		return interactionSvc.ApplyLikeEvent(context.Background(), ev)
	}); err != nil {
		slog.Error("consume like_event", "err", err)
		os.Exit(1)
	}
	slog.Info("worker started")
	feedSvc := feed.NewService(rdb, noteSvc, interactionSvc, 500)
	if err := m.Consume(mq.QueueFeedFanout, func(body []byte) error {
		var ev mq.FanoutEvent
		if err := json.Unmarshal(body, &ev); err != nil {
			return err
		}
		return feedSvc.HandleFanout(context.Background(), ev)
	}); err != nil {
		slog.Error("consume feed_fanout", "err", err)
		os.Exit(1)
	}
	select {}
}
