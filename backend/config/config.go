package config

import (
	"fmt"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Log           LogConfig       `mapstructure:"log"`
	HTTP          HTTPConfig      `mapstructure:"http"`
	AdminPassword string          `mapstructure:"admin_password"`
	PG            PGConfig        `mapstructure:"pg"`
	MQ            MQConfig        `mapstructure:"mq"`
	RAG           RAGConfig       `mapstructure:"rag"`
	Redis         RedisConfig     `mapstructure:"redis"`
	Auth          AuthConfig      `mapstructure:"auth"`
	S3            S3Config        `mapstructure:"s3"`
	Sentry        SentryConfig    `mapstructure:"sentry"`
	CaddyAPI      string          `mapstructure:"caddy_api"`
	SubnetPrefix  string          `mapstructure:"subnet_prefix"`
	CloudAuth     CloudAuthConfig `mapstructure:"cloud_auth"`
	// 敏感字段落库 SM4 加密密钥，未设置时回退 JWT 密钥
	DataEncryptionKey string `mapstructure:"data_encryption_key"`
	// 更换 JWT_SECRET 后填旧值（逗号分隔可多个），只用于解密/校验旧数据，不再签发令牌
	PreviousJWTSecrets []string `mapstructure:"previous_jwt_secrets"`
	// 三员分立：安全管理员 secadmin、审计管理员 auditadmin 的口令，未设置则不创建对应账号
	SecurityAdminPassword string `mapstructure:"security_admin_password"`
	AuditAdminPassword    string `mapstructure:"audit_admin_password"`
	BackupDir             string `mapstructure:"backup_dir"`
}

// FieldEncryptionSecrets 第一个用于加密，其余只用于解密改用 DATA_ENCRYPTION_KEY 或更换 JWT_SECRET 之前写入的数据。
func (c *Config) FieldEncryptionSecrets() []string {
	candidates := append([]string{c.DataEncryptionKey, c.Auth.JWT.Secret}, c.PreviousJWTSecrets...)
	secrets := make([]string, 0, len(candidates))
	for _, s := range candidates {
		s = strings.TrimSpace(s)
		if s != "" && !slices.Contains(secrets, s) {
			secrets = append(secrets, s)
		}
	}
	return secrets
}

type CloudAuthConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

type LogConfig struct {
	Level int `mapstructure:"level"`
}

type HTTPConfig struct {
	Port           int      `mapstructure:"port"`
	TrustedProxies []string `mapstructure:"trusted_proxies"`
	RateLimitRPS   float64  `mapstructure:"rate_limit_rps"`
	RateLimitBurst int      `mapstructure:"rate_limit_burst"`
}

type PGConfig struct {
	DSN string `mapstructure:"dsn"`
}

type MQConfig struct {
	Type string     `mapstructure:"type"`
	NATS NATSConfig `mapstructure:"nats"`
}

type NATSConfig struct {
	Server   string `mapstructure:"server"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type RAGConfig struct {
	Provider string      `mapstructure:"provider"`
	CTRAG    CTRAGConfig `mapstructure:"ct_rag"`
}

type CTRAGConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
}

type AuthConfig struct {
	Type string    `mapstructure:"type"`
	JWT  JWTConfig `mapstructure:"jwt"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}

type S3Config struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
}

type SentryConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	DSN     string `mapstructure:"dsn"`
}

func NewConfig() (*Config, error) {
	// set default config
	SUBNET_PREFIX := os.Getenv("SUBNET_PREFIX")
	if SUBNET_PREFIX == "" {
		SUBNET_PREFIX = "169.254.15"
	}
	defaultConfig := &Config{
		Log: LogConfig{
			Level: 0,
		},
		// AdminPassword / JWT / Redis / S3 secrets must come from config file or env.
		HTTP: HTTPConfig{
			Port:           8000,
			RateLimitRPS:   20,
			RateLimitBurst: 60,
		},
		PG: PGConfig{},
		MQ: MQConfig{
			Type: "nats",
			NATS: NATSConfig{
				Server: "nats://panda-wiki-nats:4222",
				User:   "panda-wiki",
			},
		},
		RAG: RAGConfig{
			Provider: "ct",
			CTRAG: CTRAGConfig{
				BaseURL: (&url.URL{
					Scheme: "http",
					Host:   fmt.Sprintf("%s.18:5050", SUBNET_PREFIX),
				}).String(),
			},
		},
		Redis: RedisConfig{
			Addr: "panda-wiki-redis:6379",
		},
		Auth: AuthConfig{
			Type: "jwt",
		},
		S3: S3Config{
			Endpoint: "panda-wiki-minio:9000",
		},
		Sentry: SentryConfig{
			Enabled: false,
		},
		CaddyAPI:     "/app/run/caddy-admin.sock",
		SubnetPrefix: "169.254.15",
	}

	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.SetConfigName("config")
	viper.SetConfigType("yml")

	// try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// if config file not found, return default config
			return nil, err
		}
	}

	// merge config file values to default config
	if err := viper.Unmarshal(defaultConfig); err != nil {
		return nil, err
	}

	// finally, override sensitive info with env variables
	overrideWithEnv(defaultConfig)

	if err := validateRequiredSecrets(defaultConfig); err != nil {
		return nil, err
	}

	return defaultConfig, nil
}

func validateRequiredSecrets(c *Config) error {
	missing := make([]string, 0, 4)
	if strings.TrimSpace(c.Auth.JWT.Secret) == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if strings.TrimSpace(c.Redis.Password) == "" {
		missing = append(missing, "REDIS_PASSWORD")
	}
	if strings.TrimSpace(c.S3.AccessKey) == "" {
		missing = append(missing, "S3_ACCESS_KEY")
	}
	if strings.TrimSpace(c.S3.SecretKey) == "" {
		missing = append(missing, "S3_SECRET_KEY")
	}
	if c.MQ.Type == "nats" && strings.TrimSpace(c.MQ.NATS.Password) == "" {
		missing = append(missing, "NATS_PASSWORD")
	}
	if len(missing) == 0 {
		if len(c.Auth.JWT.Secret) < 32 {
			fmt.Fprintln(os.Stderr, "WARNING: JWT_SECRET is shorter than 32 characters; use a random value such as `openssl rand -hex 32`")
		}
		return nil
	}
	return fmt.Errorf("missing required secrets: %s (set via environment or config file)", strings.Join(missing, ", "))
}

// overrideWithEnv override sensitive info with env variables
func overrideWithEnv(c *Config) {
	if env := os.Getenv("POSTGRES_PASSWORD"); env != "" {
		c.PG.DSN = fmt.Sprintf("host=panda-wiki-postgres user=panda-wiki password=%s dbname=panda-wiki port=5432 %s TimeZone=Asia/Shanghai", env, pgSSLParams())
	}
	if env := os.Getenv("NATS_PASSWORD"); env != "" {
		c.MQ.NATS.Password = env
	}
	if env := os.Getenv("REDIS_PASSWORD"); env != "" {
		c.Redis.Password = env
	}
	if env := os.Getenv("JWT_SECRET"); env != "" {
		c.Auth.JWT.Secret = env
	}
	if env := os.Getenv("S3_ACCESS_KEY"); env != "" {
		c.S3.AccessKey = env
	}
	if env := os.Getenv("S3_SECRET_KEY"); env != "" {
		c.S3.SecretKey = env
	}
	if env := os.Getenv("ADMIN_PASSWORD"); env != "" {
		c.AdminPassword = env
	}
	if env := os.Getenv("DATA_ENCRYPTION_KEY"); env != "" {
		c.DataEncryptionKey = env
	}
	if env := os.Getenv("PREVIOUS_JWT_SECRET"); env != "" {
		c.PreviousJWTSecrets = strings.Split(env, ",")
	}
	if env := os.Getenv("SECURITY_ADMIN_PASSWORD"); env != "" {
		c.SecurityAdminPassword = env
	}
	if env := os.Getenv("AUDIT_ADMIN_PASSWORD"); env != "" {
		c.AuditAdminPassword = env
	}
	if env := os.Getenv("BACKUP_DIR"); env != "" {
		c.BackupDir = env
	}
	if c.BackupDir == "" {
		c.BackupDir = "/app/data/backups"
	}
	if env := os.Getenv("RAG_API_KEY"); env != "" {
		c.RAG.CTRAG.APIKey = env
	}
	if env := os.Getenv("SUBNET_PREFIX"); env != "" {
		c.SubnetPrefix = env
	}
	// pg
	if env := os.Getenv("PG_DSN"); env != "" {
		c.PG.DSN = env
	}
	// nats
	if env := os.Getenv("MQ_NATS_SERVER"); env != "" {
		c.MQ.NATS.Server = env
	}
	// rag
	if env := os.Getenv("RAG_CT_RAG_BASE_URL"); env != "" {
		c.RAG.CTRAG.BaseURL = env
	}
	// redis
	if env := os.Getenv("REDIS_ADDR"); env != "" {
		c.Redis.Addr = env
	}
	// s3
	if env := os.Getenv("S3_ENDPOINT"); env != "" {
		c.S3.Endpoint = env
	}
	// sentry
	if env := os.Getenv("SENTRY_ENABLED"); env != "" {
		c.Sentry.Enabled = env == "true"
	}
	if env := os.Getenv("SENTRY_DSN"); env != "" {
		c.Sentry.DSN = env
	}
	// caddy api
	if env := os.Getenv("CADDY_API"); env != "" {
		c.CaddyAPI = env
	}
	if env := os.Getenv("CLOUD_AUTH_BASE_URL"); env != "" {
		c.CloudAuth.BaseURL = env
	}
	if env := os.Getenv("TRUSTED_PROXIES"); env != "" {
		c.HTTP.TrustedProxies = strings.Split(env, ",")
	}
	if env := os.Getenv("RATE_LIMIT_RPS"); env != "" {
		if v, err := strconv.ParseFloat(env, 64); err == nil {
			c.HTTP.RateLimitRPS = v
		}
	}
	if env := os.Getenv("RATE_LIMIT_BURST"); env != "" {
		if v, err := strconv.Atoi(env); err == nil {
			c.HTTP.RateLimitBurst = v
		}
	}
	// log level
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		if i, err := strconv.Atoi(env); err == nil {
			// -4: debug
			// 0: info
			// 4: warn
			// 8: error
			c.Log.Level = i
		} else {
			fmt.Fprintln(os.Stderr, "Invalid LOG_LEVEL; using default")
		}
	}
}

// pgSSLParams: prefer 在数据库开启 TLS 时加密、未开启时回退明文；
// 数据库配好证书后用 PG_SSLMODE=verify-full + PG_SSLROOTCERT 强制校验。
func pgSSLParams() string {
	mode := strings.TrimSpace(os.Getenv("PG_SSLMODE"))
	if mode == "" {
		mode = "prefer"
	}
	params := "sslmode=" + mode
	if rootCert := strings.TrimSpace(os.Getenv("PG_SSLROOTCERT")); rootCert != "" {
		params += " sslrootcert=" + rootCert
	}
	return params
}

func (*Config) GetString(key string) string {
	return viper.GetString(key)
}

func (*Config) GetInt(key string) int {
	return viper.GetInt(key)
}

func (*Config) GetUint64(key string) uint64 {
	return viper.GetUint64(key)
}

func (*Config) GetBool(key string) bool {
	return viper.GetBool(key)
}

func (*Config) GetStringSlice(key string) []string {
	return viper.GetStringSlice(key)
}

func (*Config) GetFloat64(key string) float64 {
	return viper.GetFloat64(key)
}
