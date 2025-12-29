package api

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CognitoConfig holds Cognito configuration
type CognitoConfig struct {
	Region     string
	UserPoolID string
	ClientID   string
	jwks       *JWKS
}

// JWKS represents JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// NewCognitoConfig creates a new Cognito configuration
func NewCognitoConfig(region, userPoolID, clientID string) (*CognitoConfig, error) {
	cfg := &CognitoConfig{
		Region:     region,
		UserPoolID: userPoolID,
		ClientID:   clientID,
	}

	// Fetch JWKS (public keys for verifying tokens)
	if err := cfg.fetchJWKS(); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	return cfg, nil
}

// fetchJWKS downloads the public keys from Cognito
func (c *CognitoConfig) fetchJWKS() error {
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json",
		c.Region, c.UserPoolID)

	resp, err := http.Get(jwksURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch JWKS: status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	c.jwks = &jwks
	return nil
}

// ValidateToken validates a Cognito JWT token
func (c *CognitoConfig) ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		// Find matching key in JWKS
		key, err := c.getPublicKey(kid)
		if err != nil {
			return nil, err
		}

		return key, nil
	})

	if err != nil {
		return nil, err
	}

	// Validate claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Verify token_use is "access" or "id"
	tokenUse, ok := claims["token_use"].(string)
	if !ok || (tokenUse != "access" && tokenUse != "id") {
		return nil, fmt.Errorf("invalid token_use: %s", tokenUse)
	}

	// Verify client_id matches (for id tokens) or verify issuer
	iss, ok := claims["iss"].(string)
	if !ok {
		return nil, fmt.Errorf("missing issuer claim")
	}

	expectedIssuer := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", c.Region, c.UserPoolID)
	if iss != expectedIssuer {
		return nil, fmt.Errorf("invalid issuer: %s", iss)
	}

	// Verify expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			return nil, fmt.Errorf("token expired")
		}
	}

	return token, nil
}

// getPublicKey retrieves the RSA public key for the given key ID
func (c *CognitoConfig) getPublicKey(kid string) (*rsa.PublicKey, error) {
	for _, key := range c.jwks.Keys {
		if key.Kid == kid {
			return c.convertJWKToPublicKey(key)
		}
	}
	return nil, fmt.Errorf("key with kid %s not found", kid)
}

// convertJWKToPublicKey converts a JWK to an RSA public key
func (c *CognitoConfig) convertJWKToPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode N (modulus)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode E (exponent)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, err
	}
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// extractBearerToken extracts the token from Authorization header
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}
