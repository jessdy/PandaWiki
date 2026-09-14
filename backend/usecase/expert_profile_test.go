package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"

	v1 "github.com/chaitin/panda-wiki/api/user/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
)

func TestFilterConsultExperts(t *testing.T) {
	users := []v1.UserListItemResp{
		{
			ID:      "doc-only",
			Account: "doc",
			Role:    consts.UserRoleUser,
			KBPerms: []v1.UserKBPermItem{{
				KBName: "知识库 A",
				Perms:  []string{string(consts.UserKBPermissionDocManage)},
			}},
		},
		{
			ID:      "consult",
			Account: "consult",
			Role:    consts.UserRoleUser,
			KBPerms: []v1.UserKBPermItem{
				{
					KBName: "知识库 B",
					Perms:  []string{string(consts.UserKBPermissionConsultManage)},
				},
				{
					KBName: "知识库 A",
					Perms:  []string{string(consts.UserKBPermissionFullControl)},
				},
				{
					KBName: "知识库 B",
					Perms:  []string{string(consts.UserKBPermissionFullControl)},
				},
			},
		},
		{
			ID:      "admin",
			Account: "admin",
			Role:    consts.UserRoleAdmin,
		},
	}

	got := filterConsultExperts(users)
	if len(got) != 2 {
		t.Fatalf("expected 2 eligible users, got %d", len(got))
	}
	if got[0].ID != "admin" || got[1].ID != "consult" {
		t.Fatalf("unexpected eligible order: %s, %s", got[0].ID, got[1].ID)
	}
	if names := consultKBNames(got[1]); !reflect.DeepEqual(names, []string{"知识库 A", "知识库 B"}) {
		t.Fatalf("unexpected deduplicated KB names: %#v", names)
	}
}

func TestExpertProfileUpdateRejectsIneligibleUser(t *testing.T) {
	profiles := &fakeExpertProfileRepo{}
	uc := &ExpertProfileUsecase{
		userRepo: fakeExpertUserRepo{users: []v1.UserListItemResp{{
			ID:      "doc-only",
			Account: "doc",
			Role:    consts.UserRoleUser,
			KBPerms: []v1.UserKBPermItem{{
				KBName: "知识库 A",
				Perms:  []string{string(consts.UserKBPermissionDocManage)},
			}},
		}}},
		expertProfileRepo: profiles,
	}

	_, err := uc.Update(context.Background(), "doc-only", &domain.UpdateExpertProfileReq{
		Name:         "测试专家",
		Introduction: "介绍",
	})
	if !errors.Is(err, ErrExpertNotEligible) {
		t.Fatalf("expected ErrExpertNotEligible, got %v", err)
	}
	if profiles.upsertCalled {
		t.Fatal("ineligible user profile must not be persisted")
	}
}

func TestExpertProfileListSupportsEmptyAndExistingProfiles(t *testing.T) {
	userRepo := fakeExpertUserRepo{users: []v1.UserListItemResp{{
		ID:      "admin",
		Account: "admin",
		Role:    consts.UserRoleAdmin,
	}}}
	for _, test := range []struct {
		name      string
		profiles  map[string]domain.ExpertProfile
		wantName  string
		wantIntro string
	}{
		{name: "empty profile", profiles: map[string]domain.ExpertProfile{}},
		{
			name: "existing profile",
			profiles: map[string]domain.ExpertProfile{
				"admin": {
					UserID:       "admin",
					Name:         "张专家",
					Introduction: "设备维修专家",
				},
			},
			wantName:  "张专家",
			wantIntro: "设备维修专家",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			uc := &ExpertProfileUsecase{
				userRepo: userRepo,
				expertProfileRepo: &fakeExpertProfileRepo{
					profiles: test.profiles,
				},
			}
			items, err := uc.List(context.Background())
			if err != nil {
				t.Fatalf("List returned error: %v", err)
			}
			if len(items) != 1 {
				t.Fatalf("expected one expert, got %d", len(items))
			}
			if items[0].Name != test.wantName || items[0].Introduction != test.wantIntro {
				t.Fatalf("unexpected profile: %#v", items[0])
			}
		})
	}
}

func TestExpertProfileUpdateTrimsAndPersists(t *testing.T) {
	profiles := &fakeExpertProfileRepo{}
	uc := &ExpertProfileUsecase{
		userRepo: fakeExpertUserRepo{users: []v1.UserListItemResp{{
			ID:      "admin",
			Account: "admin",
			Role:    consts.UserRoleAdmin,
		}}},
		expertProfileRepo: profiles,
	}

	got, err := uc.Update(context.Background(), "admin", &domain.UpdateExpertProfileReq{
		Name:         "  张专家  ",
		Introduction: "  设备维修专家  ",
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if got.Name != "张专家" || got.Introduction != "设备维修专家" {
		t.Fatalf("unexpected sanitized profile: %#v", got)
	}
	if profiles.upserted == nil || profiles.upserted.Name != "张专家" {
		t.Fatalf("profile was not persisted: %#v", profiles.upserted)
	}
}

func TestExpertProfileListPublicExcludesEmptyNames(t *testing.T) {
	uc := &ExpertProfileUsecase{
		userRepo: fakeExpertUserRepo{users: []v1.UserListItemResp{
			{ID: "named", Account: "named", Role: consts.UserRoleAdmin},
			{ID: "empty", Account: "empty", Role: consts.UserRoleAdmin},
		}},
		expertProfileRepo: &fakeExpertProfileRepo{
			profiles: map[string]domain.ExpertProfile{
				"named": {
					UserID:       "named",
					Name:         "张专家",
					Introduction: "设备维修专家",
				},
				"empty": {
					UserID:       "empty",
					Name:         "   ",
					Introduction: "不应展示",
				},
			},
		},
	}

	items, err := uc.ListPublic(context.Background())
	if err != nil {
		t.Fatalf("ListPublic returned error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "张专家" {
		t.Fatalf("unexpected public experts: %#v", items)
	}
}

type fakeExpertUserRepo struct {
	users []v1.UserListItemResp
	err   error
}

func (f fakeExpertUserRepo) ListUsers(context.Context) ([]v1.UserListItemResp, error) {
	return f.users, f.err
}

type fakeExpertProfileRepo struct {
	profiles     map[string]domain.ExpertProfile
	err          error
	upsertCalled bool
	upserted     *domain.ExpertProfile
}

func (f *fakeExpertProfileRepo) ListByUserIDs(context.Context, []string) (map[string]domain.ExpertProfile, error) {
	return f.profiles, f.err
}

func (f *fakeExpertProfileRepo) Upsert(_ context.Context, profile *domain.ExpertProfile) error {
	f.upsertCalled = true
	value := *profile
	f.upserted = &value
	return f.err
}
