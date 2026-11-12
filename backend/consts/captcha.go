package consts

type RedeemCaptchaReq struct {
	Token     string  `json:"token" validate:"required"`
	Solutions []int64 `json:"solutions" validate:"required,max=1000"`
}
