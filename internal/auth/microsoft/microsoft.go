package microsoft

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// User Provided Client ID
	ClientID           = "928e933f-9802-4e0f-820d-3631e2f1761f"
	Scope              = "XboxLive.signin offline_access"
	// Correct v2.0 endpoints for Personal Accounts (consumers)
	DeviceCodeEndpoint = "https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode"
	TokenEndpoint      = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
)

type DeviceCodeResponse struct {
	UserCode        string `json:"user_code"`
	DeviceCode      string `json:"device_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	UserID       string `json:"user_id"`
}

// StartDeviceFlow initiates the device code flow
func StartDeviceFlow() (*DeviceCodeResponse, error) {
	data := url.Values{}
	data.Set("client_id", ClientID)
	data.Set("scope", Scope)

	req, _ := http.NewRequest("POST", DeviceCodeEndpoint, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Nix-Client-Launcher/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to start device flow: %s - %s", resp.Status, string(body))
	}

	var deviceResp DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return nil, err
	}
	return &deviceResp, nil
}

// PollForToken polls the token endpoint until the user authenticates or the code expires
func PollForToken(deviceCode string, interval int) (*TokenResponse, error) {
	if interval == 0 {
		interval = 5
	}
	
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	timeout := time.After(15 * time.Minute)

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("authentication timed out")
		case <-ticker.C:
			data := url.Values{}
			data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
			data.Set("client_id", ClientID)
			data.Set("device_code", deviceCode)

			req, _ := http.NewRequest("POST", TokenEndpoint, strings.NewReader(data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("User-Agent", "Nix-Client-Launcher/1.0")

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var tokenResp TokenResponse
				if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
					return nil, err
				}
				return &tokenResp, nil
			}

			var errResp map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errResp)
			errCode, _ := errResp["error"].(string)

			if errCode == "authorization_pending" {
				continue
			} else if errCode == "slow_down" {
				ticker.Reset(time.Duration(interval+5) * time.Second)
				continue
			} else {
				return nil, fmt.Errorf("token polling failed: %v", errResp)
			}
		}
	}
}

// RefreshToken refreshes the access token using the refresh token
func RefreshToken(refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", ClientID)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")
	data.Set("scope", Scope) 

	req, _ := http.NewRequest("POST", TokenEndpoint, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Nix-Client-Launcher/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microsoft token refresh failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}
	return &tokenResp, nil
}
