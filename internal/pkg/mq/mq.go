package mq

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QueueFeedFanout = "feed_fanout"
	QueueLikeEvent  = "like_event"
	QueueLikeDLQ    = "like_event.dlq"

	dlxLike = "like_event.dlx"
)

type FanoutEvent struct {
	NoteID   int64 `json:"note_id"`
	AuthorID int64 `json:"author_id"`
	Ts       int64 `json:"ts"` // 毫秒时间戳，ZSet score
}

type LikeEvent struct {
	Kind   string `json:"kind"` // "like" | "collect"
	NoteID int64  `json:"note_id"`
	UserID int64  `json:"user_id"`
	Delta  int    `json:"delta"` // +1 | -1
}

type MQ struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func New(url string) (*MQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare(dlxLike, "fanout", true, false, false, false, nil); err != nil {
		return nil, err
	}
	if _, err := ch.QueueDeclare(QueueLikeDLQ, true, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := ch.QueueBind(QueueLikeDLQ, "", dlxLike, false, nil); err != nil {
		return nil, err
	}
	if _, err := ch.QueueDeclare(QueueLikeEvent, true, false, false, false,
		amqp.Table{"x-dead-letter-exchange": dlxLike}); err != nil {
		return nil, err
	}
	if _, err := ch.QueueDeclare(QueueFeedFanout, true, false, false, false, nil); err != nil {
		return nil, err
	}
	return &MQ{conn: conn, ch: ch}, nil
}

func (m *MQ) Publish(ctx context.Context, queue string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return m.ch.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         b,
	})
}

// Consume 进程内重试 3 次，仍失败则 nack(false,false)。
// like_event 声明了 DLX，会进入 like_event.dlq；feed_fanout 无 DLX，消息丢弃（README 演进项）。
func (m *MQ) Consume(queue string, handler func(body []byte) error) error {
	msgs, err := m.ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for d := range msgs {
			var err error
			for attempt := 1; attempt <= 3; attempt++ {
				if err = handler(d.Body); err == nil {
					break
				}
				slog.Warn("consume retry", "queue", queue, "attempt", attempt, "err", err)
				time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
			}
			if err != nil {
				slog.Error("consume failed, drop/dlq", "queue", queue, "err", err)
				_ = d.Nack(false, false)
			} else {
				_ = d.Ack(false)
			}
		}
	}()
	return nil
}

func (m *MQ) Close() error {
	_ = m.ch.Close()
	return m.conn.Close()
}
