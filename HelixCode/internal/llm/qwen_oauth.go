package llm

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// QwenOAuth2Client handles OAuth2 authentication for Qwen
type QwenOAuth2Client struct {
	httpClient *http.Client
	baseURL    string
	clientID   string
}

// QwenCredentials represents OAuth2 credentials
type QwenCredentials struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiryDate   int64  `json:"expiry_date,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ResourceURL  string `json:"resource_url,omitempty"`
}

// DeviceAuthorizationData represents device authorization response
type DeviceAuthorizationData struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
}

// DeviceTokenData represents device token response
type DeviceTokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	Endpoint     string `json:"endpoint,omitempty"`
	ResourceURL  string `json:"resource_url,omitempty"`
}

// ErrorData represents OAuth2 error response
type ErrorData struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// NewQwenOAuth2Client creates a new Qwen OAuth2 client
func NewQwenOAuth2Client() *QwenOAuth2Client {
	return &QwenOAuth2Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://chat.qwen.ai",
		clientID:   "f0304373b74a44d2b584a3fb70ca9e56", // Qwen's OAuth client ID
	}
}

// GenerateCodeVerifier generates a random code verifier for PKCE
func (c *QwenOAuth2Client) GenerateCodeVerifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// GenerateCodeChallenge generates a code challenge from verifier using SHA-256
func (c *QwenOAuth2Client) GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// RequestDeviceAuthorization requests device authorization
func (c *QwenOAuth2Client) RequestDeviceAuthorization(scope, codeChallenge string) (*DeviceAuthorizationData, error) {
	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("scope", scope)
	data.Set("code_challenge", codeChallenge)
	data.Set("code_challenge_method", "S256")

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v1/oauth2/device/code", c.baseURL),
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Request-ID", uuid.New().String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device authorization failed: %d %s", resp.StatusCode, string(body))
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Check if it's an error response
	if errorResp, ok := result.(map[string]interface{}); ok {
		if errorField, exists := errorResp["error"]; exists {
			errorData := ErrorData{
				Error: fmt.Sprintf("%v", errorField),
			}
			if desc, exists := errorResp["error_description"]; exists {
				errorData.ErrorDescription = fmt.Sprintf("%v", desc)
			}
			return nil, fmt.Errorf("device authorization failed: %s - %s",
				errorData.Error, errorData.ErrorDescription)
		}
	}

	// Parse successful response
	var authData DeviceAuthorizationData
	body, _ := json.Marshal(result)
	if err := json.Unmarshal(body, &authData); err != nil {
		return nil, err
	}

	return &authData, nil
}

// PollDeviceToken polls for device token
func (c *QwenOAuth2Client) PollDeviceToken(deviceCode, codeVerifier string) (*DeviceTokenData, error) {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("client_id", c.clientID)
	data.Set("device_code", deviceCode)
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v1/oauth2/token", c.baseURL),
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusBadRequest {
		var errorData ErrorData
		if err := json.Unmarshal(body, &errorData); err == nil {
			switch errorData.Error {
			case "authorization_pending":
				return nil, fmt.Errorf("authorization_pending")
			case "slow_down":
				return nil, fmt.Errorf("slow_down")
			default:
				return nil, fmt.Errorf("device token failed: %s - %s",
					errorData.Error, errorData.ErrorDescription)
			}
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device token failed: %d %s", resp.StatusCode, string(body))
	}

	var tokenData DeviceTokenData
	if err := json.Unmarshal(body, &tokenData); err != nil {
		return nil, err
	}

	return &tokenData, nil
}

// RefreshAccessToken refreshes an access token
func (c *QwenOAuth2Client) RefreshAccessToken(refreshToken string) (*DeviceTokenData, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", c.clientID)

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v1/oauth2/token", c.baseURL),
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %d %s", resp.StatusCode, string(body))
	}

	var tokenData DeviceTokenData
	if err := json.NewDecoder(resp.Body).Decode(&tokenData); err != nil {
		return nil, err
	}

	return &tokenData, nil
}

// GetCredentialsPath returns the path to cached credentials
func (c *QwenOAuth2Client) GetCredentialsPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".qwen", "oauth_creds.json")
}

// LoadCredentials loads cached credentials
func (c *QwenOAuth2Client) LoadCredentials() (*QwenCredentials, error) {
	path := c.GetCredentialsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var creds QwenCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

// SaveCredentials saves credentials to cache
func (c *QwenOAuth2Client) SaveCredentials(creds *QwenCredentials) error {
	path := c.GetCredentialsPath()
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// ClearCredentials removes cached credentials
func (c *QwenOAuth2Client) ClearCredentials() error {
	path := c.GetCredentialsPath()
	return os.Remove(path)
}

// IsTokenValid checks if the access token is still valid
func (c *QwenOAuth2Client) IsTokenValid(creds *QwenCredentials) bool {
	if creds == nil || creds.AccessToken == "" {
		return false
	}

	// Check if token is expired (with 5 minute buffer)
	if creds.ExpiryDate > 0 {
		return time.Now().Unix() < (creds.ExpiryDate - 300)
	}

	return true
}

// GetValidToken returns a valid access token, refreshing if necessary
func (c *QwenOAuth2Client) GetValidToken() (string, error) {
	creds, err := c.LoadCredentials()
	if err != nil {
		return "", fmt.Errorf("no cached credentials: %v", err)
	}

	if c.IsTokenValid(creds) {
		return creds.AccessToken, nil
	}

	// Token is expired, try to refresh
	if creds.RefreshToken == "" {
		return "", fmt.Errorf("no refresh token available")
	}

	tokenData, err := c.RefreshAccessToken(creds.RefreshToken)
	if err != nil {
		// Refresh failed, clear credentials
		c.ClearCredentials()
		return "", fmt.Errorf("token refresh failed: %v", err)
	}

	// Update credentials
	newCreds := &QwenCredentials{
		AccessToken:  tokenData.AccessToken,
		RefreshToken: tokenData.RefreshToken,
		TokenType:    tokenData.TokenType,
		ResourceURL:  tokenData.ResourceURL,
		ExpiryDate:   time.Now().Unix() + int64(tokenData.ExpiresIn),
	}

	if err := c.SaveCredentials(newCreds); err != nil {
		return "", fmt.Errorf("failed to save refreshed credentials: %v", err)
	}

	return newCreds.AccessToken, nil
}

// AuthenticateWithDeviceFlow performs the complete OAuth2 device flow
func (c *QwenOAuth2Client) AuthenticateWithDeviceFlow(ctx context.Context, openBrowser func(url string) error) (*QwenCredentials, error) {
	// Generate PKCE pair
	codeVerifier, err := c.GenerateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %v", err)
	}

	codeChallenge := c.GenerateCodeChallenge(codeVerifier)

	// Request device authorization
	authData, err := c.RequestDeviceAuthorization("openid profile email model.completion", codeChallenge)
	if err != nil {
		return nil, fmt.Errorf("device authorization failed: %v", err)
	}

	// Open browser for user authorization
	if openBrowser != nil {
		if err := openBrowser(authData.VerificationURIComplete); err != nil {
			fmt.Printf("Please visit: %s\n", authData.VerificationURIComplete)
		}
	} else {
		fmt.Printf("Please visit: %s\n", authData.VerificationURIComplete)
	}

	fmt.Printf("User code: %s\n", authData.UserCode)
	fmt.Println("Waiting for authorization...")

	// Poll for token
	pollInterval := 2 * time.Second
	maxAttempts := authData.ExpiresIn / 2 // Poll for half the expiry time

	for attempt := 0; attempt < maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		tokenData, err := c.PollDeviceToken(authData.DeviceCode, codeVerifier)
		if err != nil {
			if strings.Contains(err.Error(), "authorization_pending") {
				time.Sleep(pollInterval)
				continue
			}
			if strings.Contains(err.Error(), "slow_down") {
				pollInterval = time.Duration(float64(pollInterval) * 1.5)
				if pollInterval > 10*time.Second {
					pollInterval = 10 * time.Second
				}
				time.Sleep(pollInterval)
				continue
			}
			return nil, fmt.Errorf("token polling failed: %v", err)
		}

		// Success! Save credentials
		creds := &QwenCredentials{
			AccessToken:  tokenData.AccessToken,
			RefreshToken: tokenData.RefreshToken,
			TokenType:    tokenData.TokenType,
			ResourceURL:  tokenData.ResourceURL,
			ExpiryDate:   time.Now().Unix() + int64(tokenData.ExpiresIn),
		}

		if err := c.SaveCredentials(creds); err != nil {
			return nil, fmt.Errorf("failed to save credentials: %v", err)
		}

		return creds, nil
	}

	return nil, fmt.Errorf("authentication timed out")
}
