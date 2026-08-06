package note

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, n *Note) error
	FindByID(ctx context.Context, id int64) (*Note, error)
	BatchFindByIDs(ctx context.Context, ids []int64) ([]*Note, error)
	ListLatest(ctx context.Context, cursor int64, size int) ([]*Note, error)
	SoftDelete(ctx context.Context, id, userID int64) error
	AddCountDelta(ctx context.Context, id int64, field string, delta int) error
	ListAllIDs(ctx context.Context) ([]int64, error)
	SetCounts(ctx context.Context, id int64, like, collect, comment int64) error
}
type gormRepo struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepo{db: db} }
func (r *gormRepo) Create(ctx context.Context, n *Note) error {
	return r.db.WithContext(ctx).Create(n).Error
}
func (r *gormRepo) FindByID(ctx context.Context, id int64) (*Note, error) {
	var n Note
	if err := r.db.WithContext(ctx).Where("status = 0").First(&n, id).Error; err != nil {
		return nil, err
	}
	return &n, nil
}
func (r *gormRepo) BatchFindByIDs(ctx context.Context, ids []int64) ([]*Note, error) {
	var ns []*Note
	if len(ids) == 0 {
		return ns, nil
	}
	err := r.db.WithContext(ctx).Where("id IN ? AND status = 0", ids).Find(&ns).Error
	return ns, err
}
func (r *gormRepo) ListLatest(ctx context.Context, cursor int64, size int) ([]*Note, error) {
	var ns []*Note
	q := r.db.WithContext(ctx).Where("status = 0")
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	err := q.Order("id DESC").Limit(size + 1).Find(&ns).Error // 多查 1 条判断 hasMore
	return ns, err
}
func (r *gormRepo) SoftDelete(ctx context.Context, id, userID int64) error {
	res := r.db.WithContext(ctx).Model(&Note{}).
		Where("id = ? AND user_id = ? AND status = 0", id, userID).
		Update("status", 1)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// AddCountDelta 原子加减，field 为字段名如 like_count
func (r *gormRepo) AddCountDelta(ctx context.Context, id int64, field string, delta int) error {
	return r.db.WithContext(ctx).Model(&Note{}).Where("id = ?", id).
		UpdateColumn(field, gorm.Expr(field+" + ?", delta)).Error
}

func (r *gormRepo) ListAllIDs(ctx context.Context) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&Note{}).Where("status = 0").Pluck("id", &ids).Error
	return ids, err
}
func (r *gormRepo) SetCounts(ctx context.Context, id int64, like, collect, comment int64) error {
	return r.db.WithContext(ctx).Model(&Note{}).Where("id = ?", id).Updates(map[string]any{
		"like_count": like, "collect_count": collect, "comment_count": comment,
	}).Error
}
