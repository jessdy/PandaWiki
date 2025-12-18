package setup

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
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
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "pandawiki.docs.baizhi.cloud",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // Certificate valid for 10 year
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{"pandawiki.docs.baizhi.cloud"},
	}

	// Sign certificate with private key
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, privateKey.Public(), privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %v", err)
	}

	// ensure ssl dir exists
	sslDir := getSSLDir()
	if err := os.MkdirAll(sslDir, 0o755); err != nil {
		return fmt.Errorf("failed to create ssl dir: %v", err)
	}

	// Write certificate file with appropriate permissions
	certFilePath := getSSLCertFile()
	certFile, err := os.Create(certFilePath)
	if err != nil {
		return fmt.Errorf("failed to create cert file: %v", err)
	}
	defer certFile.Close()

	// Set certificate file permissions to 644 (readable by all)
	if err := certFile.Chmod(0o644); err != nil {
		return fmt.Errorf("failed to set cert file permissions: %v", err)
	}

	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err != nil {
		return fmt.Errorf("failed to encode certificate: %v", err)
	}

	// Write private key file with appropriate permissions
	keyFilePath := getSSLKeyFile()
	keyFile, err := os.Create(keyFilePath)
	if err != nil {
		return fmt.Errorf("failed to create key file: %v", err)
	}
	defer keyFile.Close()

	// Set private key file permissions to 600 (owner read/write)
	if err := keyFile.Chmod(0o600); err != nil {
		return fmt.Errorf("failed to set key file permissions: %v", err)
	}

	err = pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err != nil {
		return fmt.Errorf("failed to encode private key: %v", err)
	}

	return nil
}
