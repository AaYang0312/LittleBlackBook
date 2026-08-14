package note

import (
	"time"
	"xbs/internal/user"
)

type Note struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	UserID       int64     `gorm:"index:idx_user_created,priority:1" json:"user_id"`
	Title        string    `gorm:"size:128" json:"title"`
	Content      string    `gorm:"type:text" json:"content"`
	CoverURL     string    `gorm:"size:255" json:"cover_url"`
	Images       string    `gorm:"type:json" json:"-"`
	LikeCount    int64     `json:"like_count"`
	CollectCount int64     `json:"collect_count"`
	CommentCount int64     `json:"comment_count"`
	Status       int8      `json:"-"`
	CreatedAt    time.Time `gorm:"index:idx_user_created,priority:2" json:"created_at"`
	UpdatedAt    time.Time `json:"-"`
}

func (Note) TableName() string { return "notes" }

type NoteDTO struct {
	ID           int64        `json:"id"`
	UserID       int64        `json:"user_id"`
	Title        string       `json:"title"`
	Content      string       `json:"content"`
	CoverURL     string       `json:"cover_url"`
	Images       []string     `json:"images"`
	LikeCount    int64        `json:"like_count"`
	CollectCount int64        `json:"collect_count"`
	CommentCount int64        `json:"comment_count"`
	CreatedAt    time.Time    `json:"created_at"`
	Author       *user.Author `json:"author,omitempty"`
}

type Page struct {
	List       []*NoteDTO `json:"list"`
	NextCursor int64      `json:"next_cursor"`
	HasMore    bool       `json:"has_more"`
}
