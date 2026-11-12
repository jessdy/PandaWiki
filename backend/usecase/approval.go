package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	v1 "github.com/chaitin/panda-wiki/api/user/v1"
	kbv1 "github.com/chaitin/panda-wiki/api/kb/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
)

type ApprovalUsecase struct {
	repo       *pg.ApprovalRepo
	userRepo   *pg.UserRepository
	kbRepo     *pg.KnowledgeBaseRepository
	authRepo   *pg.AuthRepo
	userAccess *pg.UserAccessRepository
	logger     *log.Logger
}

func NewApprovalUsecase(
	repo *pg.ApprovalRepo,
	userRepo *pg.UserRepository,
	kbRepo *pg.KnowledgeBaseRepository,
	authRepo *pg.AuthRepo,
	userAccess *pg.UserAccessRepository,
	logger *log.Logger,
) *ApprovalUsecase {
	return &ApprovalUsecase{
		repo:       repo,
		userRepo:   userRepo,
		kbRepo:     kbRepo,
		authRepo:   authRepo,
		userAccess: userAccess,
		logger:     logger.WithModule("usecase.approval"),
	}
}

type ApprovalActor struct {
	ID      string
	Account string
	Role    consts.UserRole
}

func (u *ApprovalUsecase) Submit(ctx context.Context, actor ApprovalActor, typ domain.ApprovalType, summary, targetUserID, targetAccount, kbID string, payload any) (*domain.ApprovalSubmitResp, error) {
	if consts.IsSeparatedAdminRole(actor.Role) {
		return nil, fmt.Errorf("当前账号不能发起此类申请")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	dupTypes := []domain.ApprovalType{typ}
	switch typ {
	case domain.ApprovalUserCreate, domain.ApprovalGuestCreate:
		if targetAccount != "" {
			exists, err := u.repo.ExistsPending(ctx, dupTypes, targetAccount, "")
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, fmt.Errorf("该账号已有待审批的创建申请")
			}
		}
	case domain.ApprovalUserDelete, domain.ApprovalGuestDelete:
		if targetUserID != "" {
			exists, err := u.repo.ExistsPending(ctx, dupTypes, "", targetUserID)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, fmt.Errorf("该用户已有待审批的删除申请")
			}
		}
	}
	a := &domain.UserApproval{
		Type:             typ,
		Summary:          summary,
		TargetUserID:     targetUserID,
		TargetAccount:    targetAccount,
		KbID:             kbID,
		Payload:          raw,
		RequesterID:      actor.ID,
		RequesterAccount: actor.Account,
	}
	if err := u.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return &domain.ApprovalSubmitResp{
		Pending:    true,
		ApprovalID: a.ID,
		Message:    "已提交安全管理员审批，通过后生效",
	}, nil
}

func (u *ApprovalUsecase) List(ctx context.Context, actor ApprovalActor, req domain.ApprovalListReq) (*domain.ApprovalListResp, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PerPage <= 0 {
		req.PerPage = 20
	}
	requesterID := ""
	if actor.Role == consts.UserRoleAdmin || actor.Role == consts.UserRoleUser {
		requesterID = actor.ID
	}
	list, total, err := u.repo.List(ctx, req.Status, requesterID, (req.Page-1)*req.PerPage, req.PerPage)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Payload = nil
	}
	return &domain.ApprovalListResp{Total: total, List: list}, nil
}

func (u *ApprovalUsecase) CountPending(ctx context.Context, actor ApprovalActor) (int64, error) {
	requesterID := ""
	if actor.Role != consts.UserRoleSecurityAdmin {
		requesterID = actor.ID
	}
	return u.repo.CountPending(ctx, requesterID)
}

func (u *ApprovalUsecase) Reject(ctx context.Context, actor ApprovalActor, id int64, comment string) error {
	if actor.Role != consts.UserRoleSecurityAdmin {
		return fmt.Errorf("只有安全管理员可以审批")
	}
	ok, err := u.repo.Claim(ctx, id, domain.ApprovalStatusRejected, actor.ID, actor.Account, comment)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("申请不存在或已被处理")
	}
	return nil
}

func (u *ApprovalUsecase) Approve(ctx context.Context, actor ApprovalActor, id int64, comment string) error {
	if actor.Role != consts.UserRoleSecurityAdmin {
		return fmt.Errorf("只有安全管理员可以审批")
	}
	a, err := u.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("申请不存在")
		}
		return err
	}
	if a.Status != domain.ApprovalStatusPending {
		return fmt.Errorf("申请已被处理")
	}
	ok, err := u.repo.Claim(ctx, id, domain.ApprovalStatusApproved, actor.ID, actor.Account, comment)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("申请不存在或已被处理")
	}
	if err := u.apply(ctx, a); err != nil {
		_ = u.repo.MarkFailed(ctx, id, err.Error())
		return err
	}
	return nil
}

func (u *ApprovalUsecase) apply(ctx context.Context, a *domain.UserApproval) error {
	switch a.Type {
	case domain.ApprovalUserCreate, domain.ApprovalGuestCreate:
		var payload struct {
			ID       string          `json:"id"`
			Account  string          `json:"account"`
			Password string          `json:"password"`
			Role     consts.UserRole `json:"role"`
			Region   string          `json:"region"`
		}
		if err := json.Unmarshal(a.Payload, &payload); err != nil {
			return err
		}
		return u.userRepo.CreateUserWithHash(ctx, &domain.User{
			ID:       payload.ID,
			Account:  payload.Account,
			Password: payload.Password,
			Role:     payload.Role,
			Region:   payload.Region,
		}, consts.LicenseEditionBusiness)

	case domain.ApprovalUserDelete, domain.ApprovalGuestDelete:
		var payload struct {
			UserID string `json:"user_id"`
		}
		if err := json.Unmarshal(a.Payload, &payload); err != nil {
			return err
		}
		if err := u.userRepo.DeleteUser(ctx, payload.UserID); err != nil {
			return err
		}
		u.userAccess.ForgetAuthUser(payload.UserID)
		return nil

	case domain.ApprovalKBUserInvite:
		var req kbv1.KBUserInviteReq
		if err := json.Unmarshal(a.Payload, &req); err != nil {
			return err
		}
		user, err := u.userRepo.GetUser(ctx, req.UserId)
		if err != nil {
			return err
		}
		if user.Role == consts.UserRoleAdmin || consts.IsSeparatedAdminRole(user.Role) {
			return fmt.Errorf("不能邀请该角色用户到知识库")
		}
		return u.kbRepo.CreateKBUser(ctx, &domain.KBUsers{
			KBId:   req.KBId,
			UserId: req.UserId,
			Perms:  req.Perms,
		})

	case domain.ApprovalKBUserUpdate:
		var req kbv1.KBUserUpdateReq
		if err := json.Unmarshal(a.Payload, &req); err != nil {
			return err
		}
		return u.kbRepo.UpdateKBUserPerms(ctx, req.KBId, req.UserId, req.Perms)

	case domain.ApprovalKBUserDelete:
		var req kbv1.KBUserDeleteReq
		if err := json.Unmarshal(a.Payload, &req); err != nil {
			return err
		}
		return u.kbRepo.DeleteKBUser(ctx, req.KBId, req.UserId)

	case domain.ApprovalAuthGroupCreate:
		var group domain.AuthGroup
		if err := json.Unmarshal(a.Payload, &group); err != nil {
			return err
		}
		return u.authRepo.CreateAuthGroup(ctx, &group)

	case domain.ApprovalAuthGroupUpdate:
		var payload struct {
			ID    uint             `json:"id"`
			Group domain.AuthGroup `json:"group"`
		}
		if err := json.Unmarshal(a.Payload, &payload); err != nil {
			return err
		}
		return u.authRepo.UpdateAuthGroup(ctx, payload.ID, &payload.Group)

	case domain.ApprovalAuthGroupDelete:
		var payload struct {
			ID uint `json:"id"`
		}
		if err := json.Unmarshal(a.Payload, &payload); err != nil {
			return err
		}
		return u.authRepo.DeleteAuthGroup(ctx, payload.ID)

	case domain.ApprovalUserGroupsUpdate:
		var req v1.UpdateUserGroupsReq
		if err := json.Unmarshal(a.Payload, &req); err != nil {
			return err
		}
		return u.authRepo.UpdateUserGroups(ctx, req.UserID, req.GroupIDs)
	default:
		return fmt.Errorf("unknown approval type: %s", a.Type)
	}
}

func HashPasswordForApproval(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}
