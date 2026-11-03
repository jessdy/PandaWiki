package setup

import (
	"crypto/rand"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tjfoc/gmsm/sm2"
	gmx509 "github.com/tjfoc/gmsm/x509"
)

var (
	keyFile  = getSSLKeyFile()  // Key file path
	certFile = getSSLCertFile() // Certificate file path
)

// getSSLKeyFile 获取 SSL 私钥文件路径
// 优先使用 /app/etc/nginx/ssl（生产环境），否则使用项目目录
func getSSLKeyFile() string {
	sslDir := getSSLDir()
	return filepath.Join(sslDir, "panda-wiki.key")
}

// getSSLCertFile 获取 SSL 证书文件路径
// 优先使用 /app/etc/nginx/ssl（生产环境），否则使用项目目录
func getSSLCertFile() string {
	sslDir := getSSLDir()
	return filepath.Join(sslDir, "panda-wiki.crt")
}

// getSSLDir 获取 SSL 证书目录
// 优先使用 /app/etc/nginx/ssl（生产环境），否则使用项目目录 ./ssl
func getSSLDir() string {
	// 在 Mac 开发环境中，直接跳过 /app 目录检查，使用项目目录
	// 检查是否是 Mac 系统（通过检查常见的 Mac 目录）
	if _, err := os.Stat("/Applications"); err == nil {
		// 这是 Mac 系统，直接使用项目目录
		if wd, err := os.Getwd(); err == nil {
			sslDir := filepath.Join(wd, "ssl")
			return sslDir
		}
		return filepath.Join(os.TempDir(), "panda-wiki-ssl")
	}

	// 尝试使用 /app/etc/nginx/ssl（生产环境，Linux Docker 容器）
	// 先检查 /app 目录是否存在且可写
	if info, err := os.Stat("/app"); err == nil && info.IsDir() {
		// 尝试创建测试目录来验证可写性
		testDir := "/app/etc/nginx/ssl"
		if err := os.MkdirAll(testDir, 0o755); err == nil {
			// 进一步验证：尝试创建一个临时文件来确保目录真正可写
			testFile := filepath.Join(testDir, ".test-write")
			if f, err := os.Create(testFile); err == nil {
				f.Close()
				os.Remove(testFile) // 清理测试文件
				return testDir
			}
			// 如果创建测试文件失败，说明目录不可写，回退到项目目录
		}
		// 如果创建目录失败，说明 /app 不可写，回退到项目目录
	}
	// 开发环境使用项目目录下的 ssl 目录
	// 尝试使用当前工作目录
	if wd, err := os.Getwd(); err == nil {
		sslDir := filepath.Join(wd, "ssl")
		return sslDir
	}
	// 如果无法获取工作目录，使用临时目录
	return filepath.Join(os.TempDir(), "panda-wiki-ssl")
}

func sslCertCommonName() string {
	if v := strings.TrimSpace(os.Getenv("SSL_CERT_CN")); v != "" {
		return v
	}
	return "localhost"
}

// check init cert
func CheckInitCert() error {
	// Check both key and cert files
	keyExists := false
	certExists := false

	if _, err := os.Stat(keyFile); err == nil {
		keyExists = true
	}

	if _, err := os.Stat(certFile); err == nil {
		certExists = true
	}

	// If either file is missing, recreate both
	if !keyExists || !certExists {
		return createSelfSignedCerts()
	}

	return nil
}

func createSelfSignedCerts() error {
	// Generate SM2 private key (国家商用密码)
	privateKey, err := sm2.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate sm2 private key: %v", err)
	}

	cn := sslCertCommonName()
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial: %v", err)
	}

	template := &gmx509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              gmx509.KeyUsageCertSign | gmx509.KeyUsageDigitalSignature | gmx509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []gmx509.ExtKeyUsage{gmx509.ExtKeyUsageServerAuth},
		DNSNames:              []string{cn},
	}

	certPEM, err := gmx509.CreateCertificateToPem(template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create sm2 certificate: %v", err)
	}

	sslDir := getSSLDir()
	if err := os.MkdirAll(sslDir, 0o755); err != nil {
		return fmt.Errorf("failed to create ssl dir: %v", err)
	}

	certFilePath := getSSLCertFile()
	if err := os.WriteFile(certFilePath, certPEM, 0o644); err != nil {
		return fmt.Errorf("failed to write cert file: %v", err)
	}

	keyDER, err := gmx509.MarshalSm2PrivateKey(privateKey, nil)
	if err != nil {
		return fmt.Errorf("failed to marshal sm2 private key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	keyFilePath := getSSLKeyFile()
	if err := os.WriteFile(keyFilePath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("failed to write key file: %v", err)
	}

	return nil
}
