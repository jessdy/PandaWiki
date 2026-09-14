package pg

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	storepg "github.com/chaitin/panda-wiki/store/pg"
)

type ExpertProfileRepo struct {
	db     *storepg.DB
	logger *log.Logger
}

func NewExpertProfileRepo(db *storepg.DB, logger *log.Logger) *ExpertProfileRepo {
	return &ExpertProfileRepo{
		db:     db,
		logger: logger.WithModule("repo.pg.expert_profile"),
	}
}

func (r *ExpertProfileRepo) ListByUserIDs(ctx context.Context, userIDs []string) (map[string]domain.ExpertProfile, error) {
	result := make(map[string]domain.ExpertProfile, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	var profiles []domain.ExpertProfile
	if err := r.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&profiles).Error; err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		result[profile.UserID] = profile
	}
	return result, nil
}

func (r *ExpertProfileRepo) Upsert(ctx context.Context, profile *domain.ExpertProfile) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"introduction",
			"updated_at",
		}),
	}).Create(profile).Error
}
