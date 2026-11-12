package v1

import "github.com/chaitin/panda-wiki/consts"

type AuthLoginSimpleReq struct {
	Password string `json:"password" validate:"required"`
}

type AuthLoginSimpleResp struct {
}

type AuthGetReq struct {
}
type AuthGetResp struct {
	AuthType       consts.AuthType       `json:"auth_type"`
	SourceType     consts.SourceType     `json:"source_type"`
	LicenseEdition consts.LicenseEdition `json:"license_edition"`
}

type AuthGitHubReq struct {
	KbID        string `json:"kb_id" validate:"max=64"`
	RedirectUrl string `json:"redirect_url" validate:"max=2048"`
}

type AuthGitHubResp struct {
	Url string `json:"url"`
}

type GitHubCallbackReq struct {
	Code  string `json:"code" query:"code" validate:"required,max=512"`
	State string `json:"state" query:"state" validate:"max=512"`
}

type GitHubCallbackResp struct {
}

type AuthLoginUserLinkReq struct {
	User string `json:"user" validate:"required"`
}

type AuthLoginUserLinkResp struct {
	Username string `json:"username"`
	Region   string `json:"region"`
	Role     string `json:"role"`
}

type AuthLoginTgtReq struct {
	Tgt string `json:"tgt" validate:"required"`
}

type AuthLoginTgtResp struct {
	Username string `json:"username"`
	Region   string `json:"region"`
	Role     string `json:"role"`
}

type AuthInfoResp struct {
	ID        uint   `json:"id,omitempty"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
}
