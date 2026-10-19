package consts

type SystemSettingKey string

const (
	SystemSettingModelMode     SystemSettingKey = "model_setting_mode"
	SystemSettingUpload        SystemSettingKey = "upload"
	SystemSettingStandardTerms SystemSettingKey = "standard_terms"
	SystemSettingTestEnv       SystemSettingKey = "test_env"
	SystemSettingCloudAuth     SystemSettingKey = "cloud_auth"
)
