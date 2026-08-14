package user

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByID(ctx context.Context, id int64) (*User, error)
	BatchFindByIDs(ctx context.Context, ids []int64) ([]*User, error)
}

type gormRepo struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepo{db: db} }

func (r *gormRepo) Create(c context.Context, u *User) error {
	return r.db.WithContext(c).Create(u).Error
}
func (r *gormRepo) FindByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *gormRepo) FindByID(ctx context.Context, id int64) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *gormRepo) BatchFindByIDs(ctx context.Context, ids []int64) ([]*User, error) {
	var out []*User
	if len(ids) == 0 {
		return out, nil
	}
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&out).Error
	return out, err
}
