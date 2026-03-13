package media

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const defaultSignTTLSeconds = 120

type AssetReadClaims struct {
	jwt.RegisteredClaims
	AssetID uuid.UUID `json:"assetId"`
	UserID  uuid.UUID `json:"userId"`
}

type TokenService struct {
	secret  []byte
	signTTL time.Duration
}

func NewTokenService() (*TokenService, error) {
	secret := os.Getenv("MEDIA_TOKEN_SECRET")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET_KEY")
	}
	if secret == "" {
		return nil, errors.New("missing MEDIA_TOKEN_SECRET (or JWT_SECRET_KEY fallback)")
	}

	ttlSeconds, err := strconv.Atoi(os.Getenv("MEDIA_SIGN_TTL_SECONDS"))
	if err != nil || ttlSeconds <= 0 {
		ttlSeconds = defaultSignTTLSeconds
	}

	return &TokenService{
		secret:  []byte(secret),
		signTTL: time.Duration(ttlSeconds) * time.Second,
	}, nil
}

func (s *TokenService) IssueAssetReadToken(assetID, userID uuid.UUID) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(s.signTTL)
	claims := AssetReadClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   assetID.String(),
			Issuer:    "CyimeWrite.Media",
		},
		AssetID: assetID,
		UserID:  userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return tokenStr, exp, nil
}

func (s *TokenService) VerifyAssetReadToken(tokenStr string) (*AssetReadClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AssetReadClaims{}, func(_ *jwt.Token) (any, error) {
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AssetReadClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid media token")
	}
	return claims, nil
}
