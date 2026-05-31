package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	repoPg "github.com/chaitin/panda-wiki/repo/pg"
)

type ConsultUsecase struct {
	repo   *repoPg.ConsultRepo
	logger *log.Logger
}

func NewConsultUsecase(repo *repoPg.ConsultRepo, logger *log.Logger) *ConsultUsecase {
	return &ConsultUsecase{repo: repo, logger: logger.WithModule("usecase.consult")}
}

const (
	consultMaxContent     = 8000
	consultMaxContact     = 200
	consultMaxAttachments = 13 // 上限 10 图 + 3 视频
	consultTitleMaxRunes  = 30
)

/* ---------------------------------------------------------------- */
/* Share 侧入口                                                       */
/* ---------------------------------------------------------------- */

// CreateInquiryShareInput 前台创建咨询的入参，由 handler 从 session 注入 UserID/SubmitterName。
type CreateInquiryShareInput struct {
	UserID        string
	SubmitterName string
	RemoteIP      string
	Content       string
	Contact       string
	Attachments   domain.ConsultAttachments
}

// Create 前台提交咨询：用户必须已登录（UserID 非空），content 必填。
// 自动从 content 派生 title（首行去 HTML 后前 N 字符）。
func (u *ConsultUsecase) Create(ctx context.Context, in CreateInquiryShareInput) (*domain.ConsultInquiry, error) {
	if strings.TrimSpace(in.UserID) == "" {
		return nil, fmt.Errorf("请先登录后再提交咨询")
	}
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return nil, fmt.Errorf("请填写咨询内容")
	}
	if len(content) > consultMaxContent {
		return nil, fmt.Errorf("内容过长（最多 %d 字符）", consultMaxContent)
	}
	contact := strings.TrimSpace(in.Contact)
	if len(contact) > consultMaxContact {
		return nil, fmt.Errorf("联系方式过长（最多 %d 字符）", consultMaxContact)
	}
	if err := validateAttachments(in.Attachments); err != nil {
		return nil, err
	}

	title := deriveTitle(content, consultTitleMaxRunes)
	now := time.Now()

	info := map[string]any{}
	if ip := strings.TrimSpace(in.RemoteIP); ip != "" {
		info["remote_ip"] = ip
	}
	if name := strings.TrimSpace(in.SubmitterName); name != "" {
		info["submitter_name"] = name
	}
	infoBytes, _ := json.Marshal(info)

	inquiry := &domain.ConsultInquiry{
		UserID:        in.UserID,
		Title:         title,
		Content:       content,
		Contact:       contact,
		Attachments:   in.Attachments,
		Status:        domain.ConsultStatusPending,
		Info:          infoBytes,
		CreatedAt:     now,
		UpdatedAt:     now,
		LastMessageAt: now,
	}
	firstMsg := &domain.ConsultMessage{
		SenderKind:  domain.ConsultSenderUser,
		SenderID:    in.UserID,
		SenderName:  strings.TrimSpace(in.SubmitterName),
		Content:     content,
		Attachments: in.Attachments,
		CreatedAt:   now,
	}
	if err := u.repo.CreateInquiryWithFirstMessage(ctx, inquiry, firstMsg); err != nil {
		return nil, err
	}
	return inquiry, nil
}

// AppendShareInput 前台追问入参。
type AppendShareInput struct {
	UserID        string
	SubmitterName string
	InquiryID     int64
	Content       string
	Attachments   domain.ConsultAttachments
}

// AppendUserMessage 前台同一用户在自己的咨询单上追加消息（权限按 user_id 校验）。
func (u *ConsultUsecase) AppendUserMessage(ctx context.Context, in AppendShareInput) (*domain.ConsultMessage, error) {
	if strings.TrimSpace(in.UserID) == "" {
		return nil, fmt.Errorf("请先登录")
	}
	inq, err := u.repo.GetInquiryByID(ctx, in.InquiryID)
	if err != nil {
		return nil, fmt.Errorf("咨询不存在")
	}
	if inq.UserID != in.UserID {
		return nil, fmt.Errorf("无权访问该咨询")
	}
	if inq.Status == domain.ConsultStatusClosed {
		return nil, fmt.Errorf("该咨询已关闭，无法继续追问")
	}
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return nil, fmt.Errorf("请填写追问内容")
	}
	if len(content) > consultMaxContent {
		return nil, fmt.Errorf("内容过长（最多 %d 字符）", consultMaxContent)
	}
	if err := validateAttachments(in.Attachments); err != nil {
		return nil, err
	}
	msg := &domain.ConsultMessage{
		SenderKind:  domain.ConsultSenderUser,
		SenderID:    in.UserID,
		SenderName:  strings.TrimSpace(in.SubmitterName),
		Content:     content,
		Attachments: in.Attachments,
		CreatedAt:   time.Now(),
	}
	if err := u.repo.AppendMessage(ctx, in.InquiryID, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// CloseForUser 前台用户主动关闭自己的咨询单（关闭后不可再追问）。
func (u *ConsultUsecase) CloseForUser(ctx context.Context, userID string, inquiryID int64) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("请先登录")
	}
	inq, err := u.repo.GetInquiryByID(ctx, inquiryID)
	if err != nil {
		return fmt.Errorf("咨询不存在")
	}
	if inq.UserID != userID {
		return fmt.Errorf("无权访问该咨询")
	}
	if inq.Status == domain.ConsultStatusClosed {
		return nil
	}
	return u.repo.SetStatus(ctx, inquiryID, domain.ConsultStatusClosed)
}

// ListByUser 前台「我的咨询」列表（按 user_id 强约束）。
func (u *ConsultUsecase) ListByUser(ctx context.Context, userID string, status string, page, perPage int) ([]domain.ConsultInquiryListItem, int64, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, 0, fmt.Errorf("请先登录")
	}
	return u.repo.List(ctx, repoPg.ConsultListParams{
		UserID:  userID,
		Status:  status,
		Page:    page,
		PerPage: perPage,
	})
}

// DetailForUser 前台咨询详情（含消息流），按 user_id 校验权限。
func (u *ConsultUsecase) DetailForUser(ctx context.Context, userID string, id int64) (*domain.ConsultInquiryDetail, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("请先登录")
	}
	inq, err := u.repo.GetInquiryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("咨询不存在")
	}
	if inq.UserID != userID {
		return nil, fmt.Errorf("无权访问该咨询")
	}
	msgs, err := u.repo.GetMessages(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.ConsultInquiryDetail{ConsultInquiry: *inq, Messages: msgs}, nil
}

/* ---------------------------------------------------------------- */
/* Admin 侧入口（Phase 2 用，先把基础方法落到 usecase）                  */
/* ---------------------------------------------------------------- */

func (u *ConsultUsecase) AdminList(ctx context.Context, status, keyword string, page, perPage int) ([]domain.ConsultInquiryListItem, int64, error) {
	return u.repo.List(ctx, repoPg.ConsultListParams{
		Status:  status,
		Keyword: keyword,
		Page:    page,
		PerPage: perPage,
	})
}

func (u *ConsultUsecase) AdminOpenCount(ctx context.Context) (int64, error) {
	return u.repo.CountOpen(ctx)
}

func (u *ConsultUsecase) AdminDetail(ctx context.Context, id int64) (*domain.ConsultInquiryDetail, error) {
	inq, err := u.repo.GetInquiryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	msgs, err := u.repo.GetMessages(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.ConsultInquiryDetail{ConsultInquiry: *inq, Messages: msgs}, nil
}

type AdminReplyInput struct {
	AdminID     string
	AdminName   string
	InquiryID   int64
	Content     string
	Attachments domain.ConsultAttachments
	MarkReplied bool // 回复后是否直接置为 replied 状态
}

func (u *ConsultUsecase) AdminReply(ctx context.Context, in AdminReplyInput) (*domain.ConsultMessage, error) {
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return nil, fmt.Errorf("回复内容不能为空")
	}
	if len(content) > consultMaxContent {
		return nil, fmt.Errorf("回复过长（最多 %d 字符）", consultMaxContent)
	}
	if err := validateAttachments(in.Attachments); err != nil {
		return nil, err
	}
	if _, err := u.repo.GetInquiryByID(ctx, in.InquiryID); err != nil {
		return nil, fmt.Errorf("咨询不存在")
	}
	msg := &domain.ConsultMessage{
		SenderKind:  domain.ConsultSenderAdmin,
		SenderID:    strings.TrimSpace(in.AdminID),
		SenderName:  strings.TrimSpace(in.AdminName),
		Content:     content,
		Attachments: in.Attachments,
		CreatedAt:   time.Now(),
	}
	if err := u.repo.AppendMessage(ctx, in.InquiryID, msg); err != nil {
		return nil, err
	}
	if in.MarkReplied {
		_ = u.repo.SetStatus(ctx, in.InquiryID, domain.ConsultStatusReplied)
	}
	return msg, nil
}

func (u *ConsultUsecase) AdminSetStatus(ctx context.Context, id int64, status string) error {
	if !domain.IsValidConsultStatus(status) {
		return fmt.Errorf("invalid status: %s", status)
	}
	return u.repo.SetStatus(ctx, id, status)
}

func (u *ConsultUsecase) AdminDelete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("ids is required")
	}
	return u.repo.DeleteByIDs(ctx, ids)
}

/* ---------------------------------------------------------------- */
/* 工具                                                              */
/* ---------------------------------------------------------------- */

// validateAttachments 限制：图片 ≤ 10、视频 ≤ 3、type 必须是 image/video。
// 大小/MIME 由前端校验；后端只做条数与类型基本闸门，避免 jsonb 列被滥用。
func validateAttachments(items domain.ConsultAttachments) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) > consultMaxAttachments {
		return fmt.Errorf("附件数量超限")
	}
	var imgCnt, vidCnt int
	for _, a := range items {
		switch a.Type {
		case "image":
			imgCnt++
		case "video":
			vidCnt++
		default:
			return fmt.Errorf("附件类型不支持: %s", a.Type)
		}
		if strings.TrimSpace(a.URL) == "" {
			return fmt.Errorf("附件 URL 为空")
		}
	}
	if imgCnt > 10 {
		return fmt.Errorf("图片数量超限（最多 10 张）")
	}
	if vidCnt > 3 {
		return fmt.Errorf("视频数量超限（最多 3 个）")
	}
	return nil
}

// deriveTitle 从内容提取标题（取首行非空，截取前 maxRunes 个字符）。
func deriveTitle(content string, maxRunes int) string {
	first := ""
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if t != "" {
			first = t
			break
		}
	}
	if first == "" {
		first = strings.TrimSpace(content)
	}
	runes := []rune(first)
	if len(runes) <= maxRunes {
		return first
	}
	return string(runes[:maxRunes]) + "…"
}
