package interaction

import (
	"time"

	"xbs/internal/user"
)

type Like struct {
	ID        int64 `gorm:"primaryKey"`
	UserID    int64 `gorm:"uniqueIndex:uk_user_note,priority:1"`
	NoteID    int64 `gorm:"uniqueIndex:uk_user_note,priority:2"`
	CreatedAt time.Time
}

func (Like) Tablename() string { return "likes" }

type Collect struct {
	ID        int64 `gorm:"primaryKey"`
	UserID    int64 `gorm:"uniqueIndex:uk_user_note,priority:1"`
	NoteID    int64 `gorm:"uniqueIndex:uk_user_note,priority:2"`
	CreatedAt time.Time
}

func (Collect) Tablename() string { return "collects" }

type Comment struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	NoteID     int64     `gorm:"index:idx_note_created,priority:1" json:"note_id"`
	UserID     int64     `gorm:"" json:"user_id"`
	Content    string    `gorm:"size:512" json:"content"`
	ParentID   int64     `gorm:"index:idx_parent,priority:1" json:"parent_id"`
	ReplyCount int64     `gorm:"" json:"reply_count"`
	ReplyTo    int64     `gorm:"" json:"reply_to"`
	Status     int8      `json:"-"`
	CreatedAt  time.Time `gorm:"index:idx_note_created,priority:2" json:"created_at"`
}

type CommentDTO struct {
	ID            int64        `json:"id"`
	NoteID        int64        `json:"note_id"`
	UserID        int64        `json:"user_id"`
	Content       string       `json:"content"`
	ParentID      int64        `json:"parent_id"`
	ReplyTo       int64        `json:"reply_to"`
	ReplyCount    int64        `json:"reply_count"`
	Author        *user.Author `json:"author,omitempty"`
	ReplyToAuthor *user.Author `json:"reply_to_author,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
}

func (Comment) Tablename() string { return "comments" }

type Follow struct {
	ID         int64 `gorm:"primaryKey"`
	FollowerID int64 `gorm:"uniqueIndex:uk_pair,priority:1"`
	FolloweeID int64 `gorm:"uniqueIndex:uk_pair,priority:2;index:idx_followee"`
	CreatedAt  time.Time
}

func (Follow) Tablename() string { return "follows" }
