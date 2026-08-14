package user

import "time"

type User struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64" json:"username"`
	PasswordHash string    `gorm:"size:128" json:"-"`
	Nickname     string    `gorm:"size:64" json:"nickname"`
	AvatarURL    string    `gorm:"size:255" json:"avatar_url"`
	Bio          string    `gorm:"size:255" json:"bio"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"-"`
}

func (User) TableName() string {
	return "users"
}

type Author struct {
	ID        int64  `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

func (Author) TableName() string {
	return "authors"
}
