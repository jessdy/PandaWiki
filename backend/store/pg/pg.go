package pg

import (
	"database/sql"
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratePG "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/chaitin/panda-wiki/config"
	applog "github.com/chaitin/panda-wiki/log"
)

// redactWriter 让 GORM 标准库日志也走应用脱敏链。
type redactWriter struct{ w io.Writer }

func (r redactWriter) Write(p []byte) (int, error) {
	s := applog.RedactString(string(p))
	n, err := io.WriteString(r.w, s)
	if n < len(s) && err == nil {
		err = io.ErrShortWrite
	}
	// 对外仍报告「写入了入参长度」，避免 logger 因长度不一致报错
	return len(p), err
}

// 嵌入迁移 SQL，避免依赖 runtime.Caller 解析出的源码路径。
// 雳鉴 IAST 插桩后 Caller 会变成 /tmp/moresec-...，file:// 源会找不到目录。
//
//go:embed migration/*.sql
var migrationFS embed.FS

type DB struct {
	*gorm.DB
}

func NewDB(config *config.Config) (*DB, error) {
	dsn := config.PG.DSN
	// same as gorm logger.Default, but without colorful output and ignore record not found error
	newLogger := logger.New(log.New(redactWriter{w: os.Stdout}, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true,
		Logger:         newLogger,
	})
	if err != nil {
		return nil, err
	}
	// create raglite database if not exists
	var exists bool
	if err := db.Raw("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = 'raglite')").Scan(&exists).Error; err != nil {
		return nil, err
	}
	if !exists {
		if err := db.Exec("CREATE DATABASE raglite").Error; err != nil {
			return nil, err
		}
	}
	if err := doMigrate(dsn); err != nil {
		return nil, err
	}

	return &DB{DB: db}, nil
}

func doMigrate(dsn string) error {
	// lib/pq 不支持 sslmode=prefer，迁移和 GORM 一样走 pgx
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db failed: %w", err)
	}
	driver, err := migratePG.WithInstance(db, &migratePG.Config{})
	if err != nil {
		return fmt.Errorf("with instance failed: %w", err)
	}

	sourceDriver, err := iofs.New(migrationFS, "migration")
	if err != nil {
		return fmt.Errorf("create migration source failed: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("new with database instance failed: %w", err)
	}
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			return nil
		}
		return fmt.Errorf("migrate db failed: %w", err)
	}

	return nil
}
