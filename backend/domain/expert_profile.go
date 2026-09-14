package domain

import (
	"time"

	"github.com/chaitin/panda-wiki/consts"
)

type ExpertProfile struct {
	UserID       string    `json:"user_id" gorm:"column:user_id;primaryKey"`
	Name         string    `json:"name" gorm:"column:name"`
	Introduction string    `json:"introduction" gorm:"column:introduction"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (ExpertProfile) TableName() string { return "expert_profiles" }

type ExpertProfileListItem struct {
	UserID         string          `json:"user_id" gorm:"column:user_id"`
	Account        string          `json:"account" gorm:"column:account"`
	Role           consts.UserRole `json:"role" gorm:"column:role"`
	ConsultKBNames []string        `json:"consult_kb_names"`
	Name           string          `json:"name" gorm:"column:name"`
	Introduction   string          `json:"introduction" gorm:"column:introduction"`
	UpdatedAt      *time.Time      `json:"updated_at,omitempty" gorm:"column:updated_at"`
}

type UpdateExpertProfileReq struct {
	Name         string `json:"name"`
	Introduction string `json:"introduction"`
}

type PublicExpertProfile struct {
	Name         string `json:"name"`
	Introduction string `json:"introduction"`
}
