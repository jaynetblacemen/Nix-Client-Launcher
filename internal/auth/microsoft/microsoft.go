package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ClientID     = "da544862-adab-4f3f-b5a2-2ea14cebeb26"
	RedirectURI  = "http://localhost:53682/callback"
	Scope        = "XboxLive.Signin offline_access"
	AuthEndpoint = "https://login.live.com/oauth20_authorize.srf"
	TokenEndpoint = "https://login.live.com/oauth20_token.srf"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	UserID       string `json:"user_id"`
}

func GetLoginURL() string {
	u, _ := url.Parse(AuthEndpoint)
	q := u.Query()
	q.Set("client_id", ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", RedirectURI)
	q.Set("scope", Scope)
	u.RawQuery = q.Encode()
	return u.String()
}

func ExchangeCode(code string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", ClientID)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", RedirectURI)

	req, err := http.NewRequest("POST", TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("microsoft token exchange failed: %s", resp.Status)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}
	return &tokenResp, nil
}

func RefreshToken(refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", ClientID)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")
	data.Set("redirect_uri", RedirectURI)

	req, err := http.NewRequest("POST", TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("microsoft token refresh failed: %s", resp.Status)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}
	return &tokenResp, nil
}

func StartCallbackServer(ctx context.Context) (string, error) {
	codeChan := make(chan string)
	errChan := make(chan error)

	server := &http.Server{Addr: ":53682"}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No code found", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "Login successful! You can close this window.")
		codeChan <- code
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case code := <-codeChan:
		server.Shutdown(ctx)
		return code, nil
	case err := <-errChan:
		return "", err
	case <-ctx.Done():
		server.Shutdown(ctx)
		return "", ctx.Err()
	}
}
