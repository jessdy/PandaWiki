package pg

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
	"github.com/chaitin/panda-wiki/utils"
)

type AppRepository struct {
	db     *pg.DB
	logger *log.Logger
	cipher *utils.FieldCipher
}

func NewAppRepository(db *pg.DB, cipher *utils.FieldCipher, logger *log.Logger) *AppRepository {
	return &AppRepository{
		db:     db,
		logger: logger.WithModule("repo.pg.app"),
		cipher: cipher,
	}
}

func (r *AppRepository) GetAppDetail(ctx context.Context, id string) (*domain.App, error) {
	app := &domain.App{}
	if err := r.db.WithContext(ctx).
		Model(&domain.App{}).
		Where("id = ?", id).
		First(app).Error; err != nil {
		return nil, err
	}
	r.openApp(app)
	return app, nil
}

func (r *AppRepository) UpdateApp(ctx context.Context, id, kbId string, appRequest *domain.UpdateAppReq) error {
	updateMap := map[string]any{}
	if appRequest.Name != nil {
		updateMap["name"] = appRequest.Name
	}
	if appRequest.Settings != nil {
		sealed := *appRequest.Settings
		if err := SealAppSettings(r.cipher, &sealed); err != nil {
			return err
		}
		updateMap["settings"] = &sealed
	}
	return r.db.WithContext(ctx).Model(&domain.App{}).Where("id = ? and kb_id = ?", id, kbId).Updates(updateMap).Error
}

func (r *AppRepository) openApp(app *domain.App) {
	if app == nil {
		return
	}
	if err := openAppSettings(r.cipher, &app.Settings); err != nil {
		r.logger.Error("decrypt app settings failed", log.String("app_id", app.ID), log.Error(err))
	}
}

func (r *AppRepository) DeleteApp(ctx context.Context, id, kbId string) error {
	return r.db.WithContext(ctx).Delete(&domain.App{}, "id = ? and kb_id = ?", id, kbId).Error
}

func (r *AppRepository) GetOrCreateAppByKBIDAndType(ctx context.Context, kbID string, appType domain.AppType) (*domain.App, error) {
	app := &domain.App{}
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&domain.App{}).Where("kb_id = ? AND type = ?", kbID, appType).First(app).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// create app if kb is exist
				if err := tx.Model(&domain.KnowledgeBase{}).Where("id = ?", kbID).First(&domain.KnowledgeBase{}).Error; err != nil {
					return err
				}
				app = &domain.App{
					ID:   uuid.New().String(),
					KBID: kbID,
					Type: appType,
				}
				return tx.Create(app).Error
			}
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	r.openApp(app)
	return app, nil
}

// GetAppsByTypes returns all apps of a specific type
func (r *AppRepository) GetAppsByTypes(ctx context.Context, appTypes []domain.AppType) ([]*domain.App, error) {
	var apps []*domain.App
	if err := r.db.WithContext(ctx).
		Model(&domain.App{}).
		Where("type IN (?)", appTypes).
		Find(&apps).Error; err != nil {
		return nil, err
	}
	for _, app := range apps {
		r.openApp(app)
	}
	return apps, nil
}

func (r *AppRepository) GetAppList(ctx context.Context, kbID string) (map[string]*domain.App, error) {
	var apps []*domain.App
	if err := r.db.WithContext(ctx).
		Model(&domain.App{}).
		Where("kb_id = ?", kbID).
		Find(&apps).Error; err != nil {
		return nil, err
	}
	for _, app := range apps {
		r.openApp(app)
	}
	return lo.SliceToMap(apps, func(app *domain.App) (string, *domain.App) {
		return app.ID, app
	}), nil
}
