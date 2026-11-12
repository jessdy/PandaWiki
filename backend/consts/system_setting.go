package consts

type SystemSettingName string

const (
	SystemSettingModelMode      SystemSettingName = "model_setting_mode"
	SystemSettingUpload         SystemSettingName = "upload"
	SystemSettingStandardTerms  SystemSettingName = "standard_terms"
	SystemSettingTestEnv        SystemSettingName = "test_env"
	SystemSettingCloudAuth      SystemSettingName = "cloud_auth"
	SystemSettingClassification SystemSettingName = "classification"
	SystemSettingBackup         SystemSettingName = "backup"
	SystemSettingAuditStorage   SystemSettingName = "audit_storage"
)
