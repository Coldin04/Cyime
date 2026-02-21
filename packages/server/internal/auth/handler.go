package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/database"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
)

// Shared struct to store user info from any provider
type UserProfile struct {
	Subject string
	Email   string
	Name    string
	Picture string
}

// ProviderInfo represents the data sent to the frontend for a login provider.
type ProviderInfo struct {
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	SSOUrl string `json:"ssoUrl"`
}

// GetAuthConfig is the handler for GET /api/v1/auth/config
func GetAuthConfig(c *fiber.Ctx) error {
	var dbProviders []models.AuthProvider
	if err := database.DB.Where("is_active = ?", true).Find(&dbProviders).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "数据库查询失败"})
	}

	responseProviders := make([]ProviderInfo, 0, len(dbProviders))
	for _, p := range dbProviders {
		var iconURL string
		if p.IconURL != nil {
			iconURL = *p.IconURL
		}
		responseProviders = append(responseProviders, ProviderInfo{
			Name:   p.Name,
			Icon:   iconURL,
			SSOUrl: "/api/v1/auth/login/" + p.Name,
		})
	}

	return c.JSON(fiber.Map{
		"providers": responseProviders,
	})
}

// AuthLogin initiates the OIDC/OAuth2 login flow.
func AuthLogin(c *fiber.Ctx) error {
	providerName := c.Params("provider")
	ctx := c.Context()

	var dbProvider models.AuthProvider
	if err := database.DB.Where("name = ? AND is_active = ?", providerName, true).First(&dbProvider).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "提供的认证商不存在或未激活"})
	}

	endpoint, err := getEndpointFromProvider(ctx, &dbProvider)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	oauth2Config := oauth2.Config{
		ClientID:     dbProvider.ClientID,
		ClientSecret: dbProvider.ClientSecretEncrypted,
		RedirectURL:  fmt.Sprintf("http://localhost:8080/api/v1/auth/callback/%s", providerName),
		Endpoint:     endpoint,
		Scopes:       strings.Split(dbProvider.Scopes, " "),
	}

	state := generateState(c)
	return c.Redirect(oauth2Config.AuthCodeURL(state), fiber.StatusTemporaryRedirect)
}

// AuthCallback handles the callback from the OIDC/OAuth2 provider.
func AuthCallback(c *fiber.Ctx) error {
	providerName := c.Params("provider")
	ctx := c.Context()

	if err := verifyState(c); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	var dbProvider models.AuthProvider
	if err := database.DB.Where("name = ? AND is_active = ?", providerName, true).First(&dbProvider).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "提供的认证商不存在或未激活"})
	}
	
	endpoint, err := getEndpointFromProvider(ctx, &dbProvider)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	oauth2Config := oauth2.Config{
		ClientID:     dbProvider.ClientID,
		ClientSecret: dbProvider.ClientSecretEncrypted,
		RedirectURL:  fmt.Sprintf("http://localhost:8080/api/v1/auth/callback/%s", providerName),
		Endpoint:     endpoint,
		Scopes:       strings.Split(dbProvider.Scopes, " "),
	}

	code := c.Query("code")
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "无法交换授权码: " + err.Error()})
	}

	userProfile, err := getUserProfile(ctx, &dbProvider, &oauth2Config, oauth2Token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	
	// --- User lookup/creation logic would go here ---
	
	return c.JSON(fiber.Map{
		"message": "登录成功！",
		"email":   userProfile.Email,
		"name":    userProfile.Name,
		"subject": userProfile.Subject,
	})
}

// --- Helper Functions ---

func getEndpointFromProvider(ctx context.Context, provider *models.AuthProvider) (oauth2.Endpoint, error) {
	switch provider.ProtocolType {
	case "oidc":
		if provider.IssuerURL == nil || *provider.IssuerURL == "" {
			return oauth2.Endpoint{}, fmt.Errorf("OIDC提供商 '%s' 缺少issuer_url", provider.Name)
		}
		oidcProvider, err := oidc.NewProvider(ctx, *provider.IssuerURL)
		if err != nil {
			return oauth2.Endpoint{}, fmt.Errorf("无法连接到OIDC提供商 '%s'", provider.Name)
		}
		return oidcProvider.Endpoint(), nil
	case "oauth2":
		if provider.AuthURL == nil || *provider.AuthURL == "" || provider.TokenURL == nil || *provider.TokenURL == "" {
			return oauth2.Endpoint{}, fmt.Errorf("OAuth2提供商 '%s' 缺少auth_url或token_url", provider.Name)
		}
		return oauth2.Endpoint{
			AuthURL:  *provider.AuthURL,
			TokenURL: *provider.TokenURL,
		}, nil
	default:
		return oauth2.Endpoint{}, fmt.Errorf("未知的协议类型: '%s'", provider.ProtocolType)
	}
}

func generateState(c *fiber.Ctx) string {
	stateBytes := make([]byte, 32)
	rand.Read(stateBytes)
	state := base64.URLEncoding.EncodeToString(stateBytes)
	c.Cookie(&fiber.Cookie{
		Name:     "oidc_state",
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HTTPOnly: true,
		SameSite: "Lax",
	})
	return state
}

func verifyState(c *fiber.Ctx) error {
	stateFromCookie := c.Cookies("oidc_state")
	stateFromQuery := c.Query("state")
	c.ClearCookie("oidc_state")
	if stateFromCookie == "" || stateFromQuery == "" || stateFromCookie != stateFromQuery {
		return fmt.Errorf("无效的 state 参数")
	}
	return nil
}

func getUserProfile(ctx context.Context, provider *models.AuthProvider, oauth2Config *oauth2.Config, token *oauth2.Token) (*UserProfile, error) {
	var userProfile UserProfile

	switch provider.ProtocolType {
	case "oidc":
		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok {
			return nil, fmt.Errorf("无法从令牌中获取 id_token")
		}
		oidcProvider, err := oidc.NewProvider(ctx, *provider.IssuerURL)
		if err != nil {
			return nil, fmt.Errorf("无法连接到 OIDC 提供商")
		}
		idToken, err := oidcProvider.Verifier(&oidc.Config{ClientID: provider.ClientID}).Verify(ctx, rawIDToken)
		if err != nil {
			return nil, fmt.Errorf("无效的 id_token")
		}
		var claims struct {
			Email   string `json:"email"`
			Name    string `json:"name"`
			Picture string `json:"picture"`
			Subject string `json:"sub"`
		}
		if err := idToken.Claims(&claims); err != nil {
			return nil, fmt.Errorf("无法解析 id_token 的 claims")
		}
		userProfile = UserProfile{Subject: claims.Subject, Email: claims.Email, Name: claims.Name, Picture: claims.Picture}

	case "oauth2":
		if provider.UserInfoURL == nil || *provider.UserInfoURL == "" {
			return nil, fmt.Errorf("OAuth2提供商缺少user_info_url")
		}
		client := oauth2Config.Client(ctx, token)
		resp, err := client.Get(*provider.UserInfoURL)
		if err != nil {
			return nil, fmt.Errorf("无法获取用户信息")
		}
		defer resp.Body.Close()

		// NOTE: This part is still provider-specific because each provider has a different user info response structure.
		// A more advanced implementation might use a plugin system or field mapping in the DB.
		// For now, a switch on the name is a reasonable compromise.
		if provider.Name == "github" {
			var ghUser struct {
				ID     int64  `json:"id"`
				Login  string `json:"login"`
				Name   string `json:"name"`
				Email  string `json:"email"`
				Avatar string `json:"avatar_url"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
				return nil, fmt.Errorf("无法解析GitHub用户信息")
			}
			// Use login as name if name is empty
			userName := ghUser.Name
			if userName == "" {
				userName = ghUser.Login
			}
			userProfile = UserProfile{Subject: fmt.Sprintf("%d", ghUser.ID), Email: ghUser.Email, Name: userName, Picture: ghUser.Avatar}
		} else {
			return nil, fmt.Errorf("未实现对 '%s' 的用户信息解析", provider.Name)
		}
	}

	if userProfile.Subject == "" {
		return nil, fmt.Errorf("未能获取到任何用户信息")
	}
	return &userProfile, nil
}
