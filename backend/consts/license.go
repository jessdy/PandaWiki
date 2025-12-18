package consts

import (
	"github.com/labstack/echo/v4"
)

type contextKey string

const ContextKeyEdition contextKey = "edition"

type LicenseEdition int32

const (
	LicenseEditionFree       LicenseEdition = 0 // 开源版
	LicenseEditionProfession LicenseEdition = 1 // 专业版
	LicenseEditionEnterprise LicenseEdition = 2 // 企业版
	LicenseEditionBusiness   LicenseEdition = 3 // 商业版
)

func GetLicenseEdition(c echo.Context) LicenseEdition {
	edition, _ := c.Get("edition").(LicenseEdition)
	if edition == 0 {
		// 如果没有设置版本，默认返回商业版
		return LicenseEditionBusiness
	}
	return edition
}
