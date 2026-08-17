package interaction

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FollowRepository interface {
	InsertIgnore(ctx context.Context, followerID, followeeID int64) error
	Delete(ctx context.Context, followerID, followeeID int64) error
	IsFollowing(ctx context.Context, followerID, followeeID int64) (bool, error)
	FollowerIDs(ctx context.Context, followeeID int64) ([]int64, error)
}

type LikeRepository interface {
	InsertIgnore(ctx context.Context, userID, noteID int64) (created bool, err error)
	Delete(ctx context.Context, userID, noteID int64) (deleted bool, err error)
	CountByNote(ctx context.Context, noteID int64) (int64, error)
}

type CollectRepository interface {
	InsertIgnore(ctx context.Context, userID, noteID int64) (created bool, err error)
	Delete(ctx context.Context, userID, noteID int64) (deleted bool, err error)
	CountByNote(ctx context.Context, noteID int64) (int64, error)
}

type CommentRepository interface {
	Create(ctx context.Context, c *Comment) error
	ListTopLevelByNote(ctx context.Context, noteID, cursor int64, size int) ([]*Comment, error)
	ListReplies(ctx context.Context, parentID, cursor int64, size int) ([]*Comment, error)
	FindByID(ctx context.Context, id int64) (*Comment, error)
	IncrementReplyCount(ctx context.Context, parentID int64, delta int) error
	CountByNote(ctx context.Context, noteID int64) (int64, error)
}

// 四合一
type Repos struct {
	Follow  FollowRepository
	Like    LikeRepository
	Collect CollectRepository
	Comment CommentRepository
}

// Follow
type gormFollowRepo struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) FollowRepository { return &gormFollowRepo{db: db} }
func (r *gormFollowRepo) InsertIgnore(ctx context.Context, a, b int64) error {
	// 用 ON DUPLICATE KEY UPDATE 而不是 INSERT IGNORE:重复关注依然幂等,
	// 但 INSERT IGNORE 会把外键 violation 降级为警告吞掉,导致关注不存在的用户也返回成功
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&Follow{FollowerID: a, FolloweeID: b}).Error
}
func (r *gormFollowRepo) Delete(ctx context.Context, a, b int64) error {
	return r.db.WithContext(ctx).
		Where("follower_id = ? AND followee_id = ?", a, b).Delete(&Follow{}).Error
}
func (r *gormFollowRepo) IsFollowing(ctx context.Context, a, b int64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Follow{}).
		Where("follower_id = ? AND followee_id = ?", a, b).Count(&n).Error
	return n > 0, err
}
func (r *gormFollowRepo) FollowerIDs(ctx context.Context, b int64) ([]int64, error) {
	var n []int64
	err := r.db.WithContext(ctx).Model(&Follow{}).Where("followee_id = ?", b).Pluck("follower_id", &n).Error
	return n, err
}

// Like
type gormLikeRepo struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) LikeRepository { return &gormLikeRepo{db: db} }

func (r *gormLikeRepo) InsertIgnore(ctx context.Context, userID, noteID int64) (bool, error) {
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&Like{UserID: userID, NoteID: noteID})
	return res.RowsAffected > 0, res.Error
}
func (r *gormLikeRepo) Delete(ctx context.Context, userID, noteID int64) (bool, error) {
	res := r.db.WithContext(ctx).Where("user_id = ? AND note_id = ?", userID, noteID).Delete(&Like{})
	return res.RowsAffected > 0, res.Error
}
func (r *gormLikeRepo) CountByNote(ctx context.Context, noteID int64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Like{}).Where("note_id = ?", noteID).Count(&n).Error
	return n, err
}

// Collect
type gormCollectRepo struct{ db *gorm.DB }

func NewCollectRepository(db *gorm.DB) CollectRepository { return &gormCollectRepo{db: db} }

func (r *gormCollectRepo) InsertIgnore(ctx context.Context, userID, noteID int64) (bool, error) {
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&Collect{UserID: userID, NoteID: noteID})
	return res.RowsAffected > 0, res.Error
}
func (r *gormCollectRepo) Delete(ctx context.Context, userID, noteID int64) (bool, error) {
	res := r.db.WithContext(ctx).Where("user_id = ? AND note_id = ?", userID, noteID).Delete(&Collect{})
	return res.RowsAffected > 0, res.Error
}
func (r *gormCollectRepo) CountByNote(ctx context.Context, noteID int64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Collect{}).Where("note_id = ?", noteID).Count(&n).Error
	return n, err
}

// Comment
type gormCommentRepo struct{ db *gorm.DB }

func NewCommentRepository(db *gorm.DB) CommentRepository { return &gormCommentRepo{db: db} }

func (r *gormCommentRepo) Create(ctx context.Context, c *Comment) error {
	return r.db.WithContext(ctx).Create(c).Error
}
func (r *gormCommentRepo) ListTopLevelByNote(ctx context.Context, noteID, cursor int64, size int) ([]*Comment, error) {
	var cs []*Comment
	q := r.db.WithContext(ctx).Where("note_id = ? AND parent_id = 0 AND status = 0", noteID)
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	err := q.Order("id DESC").Limit(size).Find(&cs).Error
	return cs, err
}
func (r *gormCommentRepo) ListReplies(ctx context.Context, parentID, cursor int64, size int) ([]*Comment, error) {
	var cs []*Comment
	q := r.db.WithContext(ctx).Where("parent_id = ? AND status = 0", parentID)
	if cursor > 0 {
		q = q.Where("id > ?", cursor)
	}
	err := q.Order("id ASC").Limit(size).Find(&cs).Error
	return cs, err
}
func (r *gormCommentRepo) FindByID(ctx context.Context, id int64) (*Comment, error) {
	var c Comment
	if err := r.db.WithContext(ctx).Where("status = 0").First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}
func (r *gormCommentRepo) IncrementReplyCount(ctx context.Context, parentID int64, delta int) error {
	return r.db.WithContext(ctx).Model(&Comment{}).Where("id = ?", parentID).
		UpdateColumn("reply_count", gorm.Expr("reply_count + ?", delta)).Error
}
func (r *gormCommentRepo) CountByNote(ctx context.Context, noteID int64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Comment{}).Where("note_id = ? AND status = 0", noteID).Count(&n).Error
	return n, err
}

func NewRepository(db *gorm.DB) *Repos {
	return &Repos{
		Follow:  NewFollowRepository(db),
		Like:    NewLikeRepository(db),
		Collect: NewCollectRepository(db),
		Comment: NewCommentRepository(db),
	}
}
