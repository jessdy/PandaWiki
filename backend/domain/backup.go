package domain

import "time"

const (
	BackupTriggerManual     = "manual"
	BackupTriggerScheduled  = "scheduled"
	BackupTriggerPreRestore = "pre_restore"
	BackupTriggerUploaded   = "uploaded"
)

// BackupSetting 定时备份、保留份数与存储告警配置
type BackupSetting struct {
	Enabled      bool   `json:"enabled"`
	Time         string `json:"time" validate:"max=5"` // 每天执行时间 HH:MM
	Retention    int    `json:"retention" validate:"min=1,max=365"`
	IncludeFiles bool   `json:"include_files"`
	AlertPercent int    `json:"alert_percent" validate:"min=50,max=99"`
	RemindDays   int    `json:"remind_days" validate:"min=1,max=90"`
}

func DefaultBackupSetting() BackupSetting {
	return BackupSetting{
		Enabled:      true,
		Time:         "02:00",
		Retention:    7,
		IncludeFiles: true,
		AlertPercent: 80,
		RemindDays:   7,
	}
}

type BackupTable struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Rows    int64    `json:"rows"`
}

type BackupEntry struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	SM3  string `json:"sm3"`
}

// BackupManifest 备份包清单，位于压缩包末尾，并由 manifest.sig 签名
type BackupManifest struct {
	Format        string        `json:"format"`
	Version       int           `json:"version"`
	Name          string        `json:"name"`
	Trigger       string        `json:"trigger"`
	CreatedAt     time.Time     `json:"created_at"`
	CreatedBy     string        `json:"created_by"`
	SchemaVersion uint          `json:"schema_version"`
	Tables        []BackupTable `json:"tables"`
	Files         int           `json:"files"`
	FileBytes     int64         `json:"file_bytes"`
	Entries       []BackupEntry `json:"entries"`
}

type BackupInfo struct {
	Name          string    `json:"name"`
	Size          int64     `json:"size"`
	CreatedAt     time.Time `json:"created_at"`
	Trigger       string    `json:"trigger"`
	CreatedBy     string    `json:"created_by"`
	SchemaVersion uint      `json:"schema_version"`
	Tables        int       `json:"tables"`
	Rows          int64     `json:"rows"`
	Files         int       `json:"files"`
}

type BackupTaskStatus struct {
	Running    bool       `json:"running"`
	Kind       string     `json:"kind"`
	Name       string     `json:"name"`
	Step       string     `json:"step"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Error      string     `json:"error"`
}

type BackupStorageStatus struct {
	TotalBytes    int64      `json:"total_bytes"`
	FreeBytes     int64      `json:"free_bytes"`
	BackupBytes   int64      `json:"backup_bytes"`
	UsedPercent   float64    `json:"used_percent"`
	AlertPercent  int        `json:"alert_percent"`
	Alert         bool       `json:"alert"`
	LastSuccessAt *time.Time `json:"last_success_at"`
	RemindDays    int        `json:"remind_days"`
	Remind        bool       `json:"remind"`
}

type BackupRestoreReq struct {
	Name    string `json:"name" validate:"required,max=200"`
	Confirm string `json:"confirm" validate:"required"`
}

// ClassificationSetting 密级标识，默认不显示
type ClassificationSetting struct {
	Enabled bool   `json:"enabled"`
	Level   string `json:"level" validate:"max=20"`
}

type SystemAlert struct {
	Type    string `json:"type"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type SystemPublicInfo struct {
	Classification ClassificationSetting `json:"classification"`
}
