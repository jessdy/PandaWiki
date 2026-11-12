package usecase

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
	storepg "github.com/chaitin/panda-wiki/store/pg"
	"github.com/chaitin/panda-wiki/store/s3"
	"github.com/chaitin/panda-wiki/utils"
)

type BackupUsecase struct {
	db      *storepg.DB
	setting *pg.SystemSettingRepo
	s3      *s3.MinioClient
	config  *config.Config
	logger  *log.Logger
	signKey []byte

	mu     sync.Mutex
	status domain.BackupTaskStatus
}

func NewBackupUsecase(db *storepg.DB, setting *pg.SystemSettingRepo, s3Client *s3.MinioClient, config *config.Config, logger *log.Logger) (*BackupUsecase, error) {
	secrets := config.FieldEncryptionSecrets()
	if len(secrets) == 0 {
		return nil, fmt.Errorf("backup signing key is empty: set DATA_ENCRYPTION_KEY or JWT_SECRET")
	}
	if err := os.MkdirAll(config.BackupDir, 0o750); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}
	return &BackupUsecase{
		db:      db,
		setting: setting,
		s3:      s3Client,
		config:  config,
		logger:  logger.WithModule("usecase.backup"),
		signKey: utils.DeriveKey(secrets[0], "backup-manifest"),
	}, nil
}

func (u *BackupUsecase) Status() domain.BackupTaskStatus {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.status
}

func (u *BackupUsecase) setStatus(s domain.BackupTaskStatus) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.status = s
}

func (u *BackupUsecase) Setting(ctx context.Context) (domain.BackupSetting, error) {
	setting := domain.DefaultBackupSetting()
	raw, err := u.setting.GetSystemSetting(ctx, consts.SystemSettingBackup)
	if err != nil {
		return setting, nil
	}
	_ = json.Unmarshal(raw.Value, &setting)
	if setting.Retention <= 0 {
		setting.Retention = domain.DefaultBackupSetting().Retention
	}
	if setting.Time == "" {
		setting.Time = domain.DefaultBackupSetting().Time
	}
	if setting.AlertPercent <= 0 {
		setting.AlertPercent = domain.DefaultBackupSetting().AlertPercent
	}
	if setting.RemindDays <= 0 {
		setting.RemindDays = domain.DefaultBackupSetting().RemindDays
	}
	return setting, nil
}

func (u *BackupUsecase) SaveSetting(ctx context.Context, setting domain.BackupSetting) (*domain.BackupSetting, error) {
	if setting.Retention <= 0 {
		setting.Retention = domain.DefaultBackupSetting().Retention
	}
	if setting.Time == "" {
		setting.Time = domain.DefaultBackupSetting().Time
	}
	raw, err := json.Marshal(setting)
	if err != nil {
		return nil, err
	}
	if err := u.setting.UpsertSystemSetting(ctx, consts.SystemSettingBackup, string(raw), "备份策略"); err != nil {
		return nil, err
	}
	return &setting, nil
}

func (u *BackupUsecase) List() ([]domain.BackupInfo, error) {
	entries, err := os.ReadDir(u.config.BackupDir)
	if err != nil {
		return nil, err
	}
	list := make([]domain.BackupInfo, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".zip") {
			continue
		}
		info, err := u.readBackupInfo(filepath.Join(u.config.BackupDir, e.Name()))
		if err != nil {
			u.logger.Warn("skip invalid backup", log.String("name", e.Name()), log.Error(err))
			continue
		}
		list = append(list, *info)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) })
	return list, nil
}

func (u *BackupUsecase) readBackupInfo(path string) (*domain.BackupInfo, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	manifest, err := u.readManifest(path)
	if err != nil {
		return &domain.BackupInfo{
			Name:      filepath.Base(path),
			Size:      st.Size(),
			CreatedAt: st.ModTime(),
			Trigger:   "unknown",
		}, nil
	}
	var rows int64
	for _, t := range manifest.Tables {
		rows += t.Rows
	}
	return &domain.BackupInfo{
		Name:          filepath.Base(path),
		Size:          st.Size(),
		CreatedAt:     manifest.CreatedAt,
		Trigger:       manifest.Trigger,
		CreatedBy:     manifest.CreatedBy,
		SchemaVersion: manifest.SchemaVersion,
		Tables:        len(manifest.Tables),
		Rows:          rows,
		Files:         manifest.Files,
	}, nil
}

func (u *BackupUsecase) Open(name string) (*os.File, error) {
	name = filepath.Base(name)
	if !strings.HasSuffix(name, ".zip") || strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid backup name")
	}
	return os.Open(filepath.Join(u.config.BackupDir, name))
}

func (u *BackupUsecase) Delete(name string) error {
	name = filepath.Base(name)
	return os.Remove(filepath.Join(u.config.BackupDir, name))
}

func (u *BackupUsecase) Create(ctx context.Context, trigger, createdBy string) (*domain.BackupInfo, error) {
	u.mu.Lock()
	if u.status.Running {
		u.mu.Unlock()
		return nil, fmt.Errorf("已有备份或恢复任务在执行")
	}
	now := time.Now()
	u.status = domain.BackupTaskStatus{Running: true, Kind: "backup", Step: "starting", StartedAt: &now}
	u.mu.Unlock()

	info, err := u.doCreate(ctx, trigger, createdBy)
	finished := time.Now()
	status := domain.BackupTaskStatus{Running: false, Kind: "backup", StartedAt: &now, FinishedAt: &finished}
	if err != nil {
		status.Error = err.Error()
		status.Step = "failed"
	} else {
		status.Step = "done"
		status.Name = info.Name
	}
	u.setStatus(status)
	return info, err
}

func (u *BackupUsecase) doCreate(ctx context.Context, trigger, createdBy string) (*domain.BackupInfo, error) {
	setting, _ := u.Setting(ctx)
	name := fmt.Sprintf("backup-%s-%s.zip", time.Now().Format("20060102-150405"), trigger)
	path := filepath.Join(u.config.BackupDir, name)
	tmp := path + ".tmp"
	_ = os.Remove(tmp)

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return nil, err
	}
	zw := zip.NewWriter(f)
	manifest := domain.BackupManifest{
		Format:    "panda-wiki-backup",
		Version:   1,
		Name:      name,
		Trigger:   trigger,
		CreatedAt: time.Now().UTC(),
		CreatedBy: createdBy,
		Tables:    make([]domain.BackupTable, 0),
		Entries:   make([]domain.BackupEntry, 0),
	}
	if v, err := u.schemaVersion(ctx); err == nil {
		manifest.SchemaVersion = v
	}

	u.setStep("dumping database")
	tables, err := u.listTables(ctx)
	if err != nil {
		_ = zw.Close()
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, err
	}
	for _, table := range tables {
		entryName := "tables/" + table + ".jsonl"
		w, err := zw.Create(entryName)
		if err != nil {
			_ = zw.Close()
			_ = f.Close()
			_ = os.Remove(tmp)
			return nil, err
		}
		cols, rows, digest, err := u.dumpTable(ctx, table, w)
		if err != nil {
			_ = zw.Close()
			_ = f.Close()
			_ = os.Remove(tmp)
			return nil, fmt.Errorf("dump table %s: %w", table, err)
		}
		manifest.Tables = append(manifest.Tables, domain.BackupTable{Name: table, Columns: cols, Rows: rows})
		manifest.Entries = append(manifest.Entries, domain.BackupEntry{Name: entryName, Size: 0, SM3: digest})
	}

	if setting.IncludeFiles {
		u.setStep("dumping files")
		files, bytes, err := u.dumpFiles(ctx, zw, &manifest)
		if err != nil {
			u.logger.Warn("dump files failed, continue with database only", log.Error(err))
		} else {
			manifest.Files = files
			manifest.FileBytes = bytes
		}
	}

	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = zw.Close()
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, err
	}
	mw, err := zw.Create("manifest.json")
	if err != nil {
		_ = zw.Close()
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, err
	}
	if _, err := mw.Write(raw); err != nil {
		_ = zw.Close()
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, err
	}
	sw, err := zw.Create("manifest.sig")
	if err != nil {
		_ = zw.Close()
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, err
	}
	if _, err := io.WriteString(sw, utils.HMACSM3Hex(u.signKey, string(raw))); err != nil {
		_ = zw.Close()
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, err
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	_ = u.prune(setting.Retention)
	st, _ := os.Stat(path)
	var rows int64
	for _, t := range manifest.Tables {
		rows += t.Rows
	}
	return &domain.BackupInfo{
		Name: name, Size: st.Size(), CreatedAt: manifest.CreatedAt, Trigger: trigger,
		CreatedBy: createdBy, SchemaVersion: manifest.SchemaVersion,
		Tables: len(manifest.Tables), Rows: rows, Files: manifest.Files,
	}, nil
}

func (u *BackupUsecase) setStep(step string) {
	u.mu.Lock()
	u.status.Step = step
	u.mu.Unlock()
}

func (u *BackupUsecase) schemaVersion(ctx context.Context) (uint, error) {
	var v uint
	err := u.db.WithContext(ctx).Raw("SELECT version FROM schema_migrations LIMIT 1").Scan(&v).Error
	return v, err
}

func (u *BackupUsecase) listTables(ctx context.Context) ([]string, error) {
	var tables []string
	err := u.db.WithContext(ctx).Raw(`
		SELECT tablename FROM pg_tables
		WHERE schemaname = 'public'
		  AND tablename NOT IN ('schema_migrations', 'goose_db_version')
		ORDER BY tablename`).Scan(&tables).Error
	return tables, err
}

func (u *BackupUsecase) dumpTable(ctx context.Context, table string, w io.Writer) (cols []string, rows int64, digest string, err error) {
	if err = u.db.WithContext(ctx).Raw(`
		SELECT column_name FROM information_schema.columns
		WHERE table_schema='public' AND table_name=? ORDER BY ordinal_position`, table).Scan(&cols).Error; err != nil {
		return
	}
	hasher := utils.NewStreamingSM3()
	offset := 0
	for {
		var batch []map[string]any
		q := fmt.Sprintf(`SELECT * FROM %s ORDER BY 1 LIMIT 500 OFFSET %d`, quoteIdent(table), offset)
		if err = u.db.WithContext(ctx).Raw(q).Scan(&batch).Error; err != nil {
			return
		}
		if len(batch) == 0 {
			break
		}
		for _, row := range batch {
			normalizeRow(row)
			b, e := json.Marshal(row)
			if e != nil {
				err = e
				return
			}
			b = append(b, '\n')
			if _, err = w.Write(b); err != nil {
				return
			}
			hasher.Write(b)
			rows++
		}
		offset += len(batch)
	}
	digest = hasher.SumHex()
	return
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func normalizeRow(row map[string]any) {
	for k, v := range row {
		switch t := v.(type) {
		case []byte:
			row[k] = string(t)
		case time.Time:
			row[k] = t.UTC().Format(time.RFC3339Nano)
		}
	}
}

func (u *BackupUsecase) dumpFiles(ctx context.Context, zw *zip.Writer, manifest *domain.BackupManifest) (int, int64, error) {
	objects := u.s3.ListObjects(ctx, domain.Bucket, minio.ListObjectsOptions{Recursive: true})
	count := 0
	var bytes int64
	for obj := range objects {
		if obj.Err != nil {
			return count, bytes, obj.Err
		}
		if obj.Key == "" || strings.HasSuffix(obj.Key, "/") {
			continue
		}
		entryName := "files/" + obj.Key
		w, err := zw.Create(entryName)
		if err != nil {
			return count, bytes, err
		}
		reader, err := u.s3.GetObject(ctx, domain.Bucket, obj.Key, minio.GetObjectOptions{})
		if err != nil {
			return count, bytes, err
		}
		hasher := utils.NewStreamingSM3()
		n, err := io.Copy(io.MultiWriter(w, hasher), reader)
		_ = reader.Close()
		if err != nil {
			return count, bytes, err
		}
		manifest.Entries = append(manifest.Entries, domain.BackupEntry{Name: entryName, Size: n, SM3: hasher.SumHex()})
		count++
		bytes += n
	}
	return count, bytes, nil
}

func (u *BackupUsecase) prune(retention int) error {
	if retention <= 0 {
		return nil
	}
	list, err := u.List()
	if err != nil {
		return err
	}
	for i := retention; i < len(list); i++ {
		_ = u.Delete(list[i].Name)
	}
	return nil
}

func (u *BackupUsecase) readManifest(path string) (*domain.BackupManifest, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	var raw, sig []byte
	for _, f := range zr.File {
		switch f.Name {
		case "manifest.json":
			raw, err = readZipFile(f)
		case "manifest.sig":
			sig, err = readZipFile(f)
		}
		if err != nil {
			return nil, err
		}
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("manifest.json missing")
	}
	if !utils.HexEqual(utils.HMACSM3Hex(u.signKey, string(raw)), strings.TrimSpace(string(sig))) {
		return nil, fmt.Errorf("manifest signature invalid")
	}
	var manifest domain.BackupManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func (u *BackupUsecase) Restore(ctx context.Context, name, actor string) error {
	u.mu.Lock()
	if u.status.Running {
		u.mu.Unlock()
		return fmt.Errorf("已有备份或恢复任务在执行")
	}
	now := time.Now()
	u.status = domain.BackupTaskStatus{Running: true, Kind: "restore", Name: name, Step: "pre-backup", StartedAt: &now}
	u.mu.Unlock()

	err := u.doRestore(ctx, name, actor)
	finished := time.Now()
	status := domain.BackupTaskStatus{Running: false, Kind: "restore", Name: name, StartedAt: &now, FinishedAt: &finished}
	if err != nil {
		status.Error = err.Error()
		status.Step = "failed"
	} else {
		status.Step = "done"
	}
	u.setStatus(status)
	return err
}

func (u *BackupUsecase) doRestore(ctx context.Context, name, actor string) error {
	if _, err := u.doCreate(ctx, domain.BackupTriggerPreRestore, actor); err != nil {
		return fmt.Errorf("restore 前自动备份失败: %w", err)
	}
	path := filepath.Join(u.config.BackupDir, filepath.Base(name))
	manifest, err := u.readManifest(path)
	if err != nil {
		return err
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer zr.Close()

	u.setStep("restoring database")
	filesByName := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		filesByName[f.Name] = f
	}
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SET session_replication_role = 'replica'`).Error; err != nil {
			return err
		}
		for _, table := range manifest.Tables {
			entry := filesByName["tables/"+table.Name+".jsonl"]
			if entry == nil {
				return fmt.Errorf("missing table dump: %s", table.Name)
			}
			if err := tx.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", quoteIdent(table.Name))).Error; err != nil {
				return err
			}
			r, err := entry.Open()
			if err != nil {
				return err
			}
			sc := bufio.NewScanner(r)
			sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
			for sc.Scan() {
				line := sc.Bytes()
				if len(line) == 0 {
					continue
				}
				var row map[string]any
				if err := json.Unmarshal(line, &row); err != nil {
					_ = r.Close()
					return err
				}
				if err := tx.Table(table.Name).Create(row).Error; err != nil {
					_ = r.Close()
					return fmt.Errorf("restore %s: %w", table.Name, err)
				}
			}
			_ = r.Close()
			if err := sc.Err(); err != nil {
				return err
			}
		}
		_ = tx.Exec(`SET session_replication_role = 'origin'`).Error

		u.setStep("restoring files")
		for _, f := range zr.File {
			if !strings.HasPrefix(f.Name, "files/") || f.FileInfo().IsDir() {
				continue
			}
			key := strings.TrimPrefix(f.Name, "files/")
			r, err := f.Open()
			if err != nil {
				return err
			}
			_, err = u.s3.PutObject(ctx, domain.Bucket, key, r, f.FileInfo().Size(), minio.PutObjectOptions{})
			_ = r.Close()
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (u *BackupUsecase) StorageStatus(ctx context.Context) (*domain.BackupStorageStatus, error) {
	setting, _ := u.Setting(ctx)
	var st syscall.Statfs_t
	if err := syscall.Statfs(u.config.BackupDir, &st); err != nil {
		return nil, err
	}
	total := int64(st.Blocks) * int64(st.Bsize)
	free := int64(st.Bavail) * int64(st.Bsize)
	list, _ := u.List()
	var backupBytes int64
	var last *time.Time
	for i := range list {
		backupBytes += list[i].Size
		if last == nil || list[i].CreatedAt.After(*last) {
			t := list[i].CreatedAt
			last = &t
		}
	}
	usedPercent := 0.0
	if total > 0 {
		usedPercent = float64(total-free) * 100 / float64(total)
	}
	remind := last == nil || time.Since(*last) > time.Duration(setting.RemindDays)*24*time.Hour
	return &domain.BackupStorageStatus{
		TotalBytes: total, FreeBytes: free, BackupBytes: backupBytes,
		UsedPercent: usedPercent, AlertPercent: setting.AlertPercent,
		Alert: usedPercent >= float64(setting.AlertPercent),
		LastSuccessAt: last, RemindDays: setting.RemindDays, Remind: remind,
	}, nil
}

// RunScheduled 由 cron 每分钟检查一次，到达配置时间且当天尚未成功备份时执行
func (u *BackupUsecase) RunScheduled(ctx context.Context) {
	setting, err := u.Setting(ctx)
	if err != nil || !setting.Enabled {
		return
	}
	parts := strings.Split(setting.Time, ":")
	if len(parts) != 2 {
		return
	}
	var hour, minute int
	fmt.Sscanf(parts[0], "%d", &hour)
	fmt.Sscanf(parts[1], "%d", &minute)
	now := time.Now()
	if now.Hour() != hour || now.Minute() != minute {
		return
	}
	list, _ := u.List()
	for _, item := range list {
		if item.Trigger == domain.BackupTriggerScheduled && item.CreatedAt.Local().Format("2006-01-02") == now.Format("2006-01-02") {
			return
		}
	}
	if _, err := u.Create(ctx, domain.BackupTriggerScheduled, "system"); err != nil {
		u.logger.Error("scheduled backup failed", log.Error(err))
	}
}
