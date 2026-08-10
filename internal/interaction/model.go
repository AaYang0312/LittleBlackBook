package interaction

import "time"

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
	ID        int64     `gorm:"primaryKey" json:"id"`
	NoteID    int64     `gorm:"index:idx_note_created,priority:1" json:"note_id"`
	UserID    int64     `gorm:"" json:"user_id"`
	Content   string    `gorm:"size:512" json:"content"`
	Status    int8      `json:"-"`
	CreatedAt time.Time `gorm:"index:idx_note_created,priority:2" json:"created_at"`
}

func (Comment) Tablename() string { return "comments" }

type Follow struct {
	ID         int64 `gorm:"primaryKey"`
	FollowerID int64 `gorm:"uniqueIndex:uk_pair,priority:1"`
	FolloweeID int64 `gorm:"uniqueIndex:uk_pair,priority:2;index:idx_followee"`
	CreatedAt  time.Time
}

func (Follow) Tablename() string { return "follows" }
