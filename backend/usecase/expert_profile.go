package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	v1 "github.com/chaitin/panda-wiki/api/user/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/repo/pg"
)

const (
	maxExpertNameRunes         = 50
	maxExpertIntroductionRunes = 500
)

var ErrExpertNotEligible = errors.New("该管理员已不具备咨询管理权限")

type expertUserRepository interface {
	ListUsers(ctx context.Context) ([]v1.UserListItemResp, error)
}

type expertProfileRepository interface {
	ListByUserIDs(ctx context.Context, userIDs []string) (map[string]domain.ExpertProfile, error)
	Upsert(ctx context.Context, profile *domain.ExpertProfile) error
}

type ExpertProfileUsecase struct {
	userRepo          expertUserRepository
	expertProfileRepo expertProfileRepository
}

func NewExpertProfileUsecase(
	userRepo *pg.UserRepository,
	expertProfileRepo *pg.ExpertProfileRepo,
) *ExpertProfileUsecase {
	return &ExpertProfileUsecase{
		userRepo:          userRepo,
		expertProfileRepo: expertProfileRepo,
	}
}

func (u *ExpertProfileUsecase) List(ctx context.Context) ([]domain.ExpertProfileListItem, error) {
	users, err := u.userRepo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	eligibleUsers := filterConsultExperts(users)
	userIDs := make([]string, 0, len(eligibleUsers))
	for _, user := range eligibleUsers {
		userIDs = append(userIDs, user.ID)
	}
	profiles, err := u.expertProfileRepo.ListByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	result := make([]domain.ExpertProfileListItem, 0, len(eligibleUsers))
	for _, user := range eligibleUsers {
		profile := profiles[user.ID]
		result = append(result, domain.ExpertProfileListItem{
			UserID:         user.ID,
			Account:        user.Account,
			Role:           user.Role,
			ConsultKBNames: consultKBNames(user),
			Name:           profile.Name,
			Introduction:   profile.Introduction,
			UpdatedAt:      profileUpdatedAt(profile),
		})
	}
	return result, nil
}

func (u *ExpertProfileUsecase) ListPublic(ctx context.Context) ([]domain.PublicExpertProfile, error) {
	items, err := u.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.PublicExpertProfile, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		result = append(result, domain.PublicExpertProfile{
			Name:         item.Name,
			Introduction: item.Introduction,
		})
	}
	return result, nil
}

func (u *ExpertProfileUsecase) Update(
	ctx context.Context,
	userID string,
	req *domain.UpdateExpertProfileReq,
) (*domain.ExpertProfile, error) {
	users, err := u.userRepo.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	eligible := false
	for _, user := range filterConsultExperts(users) {
		if user.ID == userID {
			eligible = true
			break
		}
	}
	if !eligible {
		return nil, ErrExpertNotEligible
	}

	name := strings.TrimSpace(req.Name)
	introduction := strings.TrimSpace(req.Introduction)
	if utf8.RuneCountInString(name) > maxExpertNameRunes {
		return nil, fmt.Errorf("姓名不能超过 %d 个字符", maxExpertNameRunes)
	}
	if utf8.RuneCountInString(introduction) > maxExpertIntroductionRunes {
		return nil, fmt.Errorf("介绍不能超过 %d 个字符", maxExpertIntroductionRunes)
	}

	profile := &domain.ExpertProfile{
		UserID:       userID,
		Name:         name,
		Introduction: introduction,
	}
	if err := u.expertProfileRepo.Upsert(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func filterConsultExperts(users []v1.UserListItemResp) []v1.UserListItemResp {
	result := make([]v1.UserListItemResp, 0, len(users))
	for _, user := range users {
		if user.Role == consts.UserRoleAdmin || len(consultKBNames(user)) > 0 {
			result = append(result, user)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Role != result[j].Role {
			return result[i].Role == consts.UserRoleAdmin
		}
		return result[i].Account < result[j].Account
	})
	return result
}

func consultKBNames(user v1.UserListItemResp) []string {
	seen := make(map[string]struct{}, len(user.KBPerms))
	names := make([]string, 0, len(user.KBPerms))
	for _, kbPerm := range user.KBPerms {
		allowed := false
		for _, perm := range kbPerm.Perms {
			if perm == string(consts.UserKBPermissionConsultManage) ||
				perm == string(consts.UserKBPermissionFullControl) {
				allowed = true
				break
			}
		}
		if allowed {
			if _, exists := seen[kbPerm.KBName]; !exists {
				seen[kbPerm.KBName] = struct{}{}
				names = append(names, kbPerm.KBName)
			}
		}
	}
	sort.Strings(names)
	return names
}

func profileUpdatedAt(profile domain.ExpertProfile) *time.Time {
	if profile.UpdatedAt.IsZero() {
		return nil
	}
	value := profile.UpdatedAt
	return &value
}
