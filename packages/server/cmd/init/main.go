package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/config"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/securevalue"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProviderTemplate represents a predefined OAuth/OIDC provider configuration
type ProviderTemplate struct {
	Name        string
	DisplayName string
	IssuerURL   string
	AuthURL     string
	TokenURL    string
	UserInfoURL string
	Scopes      string
	IconURL     string
}

// Helper function to convert string to pointer
func strPtr(s string) *string {
	return &s
}

// Common provider templates
var providerTemplates = map[string]ProviderTemplate{
	"github": {
		Name:        "github",
		DisplayName: "GitHub",
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		UserInfoURL: "https://api.github.com/user",
		Scopes:      "read:user user:email",
		IconURL:     "https://github.com/fluidicon.png",
	},
	"google": {
		Name:        "google",
		DisplayName: "Google",
		IssuerURL:   "https://accounts.google.com",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
		Scopes:      "openid email profile",
		IconURL:     "https://lh3.googleusercontent.com/COxitqgJr1sJnIDe8-jiKhxDx1FrYbtRHKJ9z_hELisAlapwE9LUPh6fcXIfb5-twpw",
	},
}

func main() {
	_ = config.LoadDotEnv(".env")

	fmt.Println("🍋 CyimeWrite 初始化向导")
	fmt.Println("========================")
	fmt.Println()

	// Initialize database
	database.Connect()
	log.Println("数据库连接成功")

	reader := bufio.NewReader(os.Stdin)

	// Ask if user wants to configure an OAuth provider
	fmt.Println("是否要配置 OAuth/SSO 登录提供商？")
	fmt.Println("1. GitHub")
	fmt.Println("2. Google")
	fmt.Println("3. 自定义 OIDC 提供商")
	fmt.Println("4. 跳过（稍后手动配置）")
	fmt.Print("请选择 (1-4): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "4" {
		fmt.Println("已跳过配置。你可以稍后在数据库中手动添加提供商配置。")
		return
	}

	var provider models.AuthProvider

	switch choice {
	case "1":
		provider = configureGitHubProvider(reader)
	case "2":
		provider = configureGoogleProvider(reader)
	case "3":
		provider = configureCustomProvider(reader)
	default:
		fmt.Println("无效的选择，已跳过配置。")
		return
	}

	encryptedSecret, err := securevalue.EncryptString(provider.ClientSecretEncrypted)
	if err != nil {
		log.Fatalf("加密提供商密钥失败：%v", err)
	}
	provider.ClientSecretEncrypted = encryptedSecret

	// Save to database
	if err := database.DB.Create(&provider).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			fmt.Printf("⚠️  提供商 '%s' 已存在，跳过创建。\n", provider.Name)
		} else {
			log.Fatalf("保存提供商配置失败：%v", err)
		}
	} else {
		fmt.Printf("✅ 成功配置 %s 登录提供商！\n", provider.Name)
		fmt.Println()
		fmt.Println("下一步:")
		fmt.Println("1. 启动服务器：go run cmd/server/main.go")
		fmt.Println("2. 访问 http://localhost:8080 进行测试")
	}
}

func configureGitHubProvider(reader *bufio.Reader) models.AuthProvider {
	fmt.Println()
	fmt.Println("📦 配置 GitHub OAuth")
	fmt.Println("-------------------")
	fmt.Println("请在 GitHub OAuth Apps 页面创建应用：")
	fmt.Println("https://github.com/settings/developers")
	fmt.Println()
	fmt.Println("授权回调 URL 应设置为：http://localhost:8080/api/v1/auth/callback/github")
	fmt.Println()

	fmt.Print("Client ID: ")
	clientID, _ := reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)

	fmt.Print("Client Secret: ")
	clientSecret, _ := reader.ReadString('\n')
	clientSecret = strings.TrimSpace(clientSecret)

	return models.AuthProvider{
		ID:                    uuid.New(),
		Name:                  "github",
		ProtocolType:          "oauth2",
		AuthURL:               strPtr(providerTemplates["github"].AuthURL),
		TokenURL:              strPtr(providerTemplates["github"].TokenURL),
		UserInfoURL:           strPtr(providerTemplates["github"].UserInfoURL),
		ClientID:              clientID,
		ClientSecretEncrypted: clientSecret,
		IconURL:               strPtr(providerTemplates["github"].IconURL),
		Scopes:                providerTemplates["github"].Scopes,
		IsActive:              true,
	}
}

func configureGoogleProvider(reader *bufio.Reader) models.AuthProvider {
	fmt.Println()
	fmt.Println("📦 配置 Google OAuth")
	fmt.Println("-------------------")
	fmt.Println("请在 Google Cloud Console 创建 OAuth 凭据：")
	fmt.Println("https://console.cloud.google.com/apis/credentials")
	fmt.Println()
	fmt.Println("授权重定向 URI 应设置为：http://localhost:8080/api/v1/auth/callback/google")
	fmt.Println()

	fmt.Print("Client ID: ")
	clientID, _ := reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)

	fmt.Print("Client Secret: ")
	clientSecret, _ := reader.ReadString('\n')
	clientSecret = strings.TrimSpace(clientSecret)

	return models.AuthProvider{
		ID:                    uuid.New(),
		Name:                  "google",
		ProtocolType:          "oauth2",
		IssuerURL:             strPtr(providerTemplates["google"].IssuerURL),
		AuthURL:               strPtr(providerTemplates["google"].AuthURL),
		TokenURL:              strPtr(providerTemplates["google"].TokenURL),
		UserInfoURL:           strPtr(providerTemplates["google"].UserInfoURL),
		ClientID:              clientID,
		ClientSecretEncrypted: clientSecret,
		IconURL:               strPtr(providerTemplates["google"].IconURL),
		Scopes:                providerTemplates["google"].Scopes,
		IsActive:              true,
	}
}

func configureCustomProvider(reader *bufio.Reader) models.AuthProvider {
	fmt.Println()
	fmt.Println("📦 配置自定义 OIDC 提供商")
	fmt.Println("-----------------------")

	fmt.Print("提供商名称 (例如：keycloak, auth0): ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print("Issuer URL: ")
	issuerURL, _ := reader.ReadString('\n')
	issuerURL = strings.TrimSpace(issuerURL)

	fmt.Print("Auth URL: ")
	authURL, _ := reader.ReadString('\n')
	authURL = strings.TrimSpace(authURL)

	fmt.Print("Token URL: ")
	tokenURL, _ := reader.ReadString('\n')
	tokenURL = strings.TrimSpace(tokenURL)

	fmt.Print("UserInfo URL: ")
	userInfoURL, _ := reader.ReadString('\n')
	userInfoURL = strings.TrimSpace(userInfoURL)

	fmt.Print("Client ID: ")
	clientID, _ := reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)

	fmt.Print("Client Secret: ")
	clientSecret, _ := reader.ReadString('\n')
	clientSecret = strings.TrimSpace(clientSecret)

	fmt.Print("Scopes (空格分隔，例如：openid email profile): ")
	scopes, _ := reader.ReadString('\n')
	scopes = strings.TrimSpace(scopes)

	fmt.Print("Icon URL (可选): ")
	iconURL, _ := reader.ReadString('\n')
	iconURL = strings.TrimSpace(iconURL)

	return models.AuthProvider{
		ID:                    uuid.New(),
		Name:                  name,
		ProtocolType:          "oidc",
		IssuerURL:             &issuerURL,
		AuthURL:               &authURL,
		TokenURL:              &tokenURL,
		UserInfoURL:           &userInfoURL,
		ClientID:              clientID,
		ClientSecretEncrypted: clientSecret,
		IconURL:               &iconURL,
		Scopes:                scopes,
		IsActive:              true,
	}
}

// Helper function to check if a provider exists
func providerExists(name string) bool {
	var count int64
	database.DB.Model(&models.AuthProvider{}).Where("name = ?", name).Count(&count)
	return count > 0
}

// Helper function to handle provider update or create
func upsertProvider(provider *models.AuthProvider) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Check if provider exists
		var existing models.AuthProvider
		result := tx.Where("name = ?", provider.Name).First(&existing)

		if result.Error == nil {
			// Update existing provider
			return tx.Model(&existing).Updates(provider).Error
		} else if result.Error == gorm.ErrRecordNotFound {
			// Create new provider
			return tx.Create(provider).Error
		}

		return result.Error
	})
}
