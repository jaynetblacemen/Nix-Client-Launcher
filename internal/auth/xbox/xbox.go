package xbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	XboxLiveAuthURL = "https://user.auth.xboxlive.com/user/authenticate"
	XSTSAuthURL     = "https://xsts.auth.xboxlive.com/xsts/authorize"
)

type XboxAuthRequest struct {
	Properties XboxAuthProperties `json:"Properties"`
	RelyingParty string           `json:"RelyingParty"`
	TokenType    string           `json:"TokenType"`
}

type XboxAuthProperties struct {
	AuthMethod string `json:"AuthMethod"`
	SiteName   string `json:"SiteName"`
	RpsTicket  string `json:"RpsTicket"`
}

type XboxAuthResponse struct {
	IssueInstant  string `json:"IssueInstant"`
	NotAfter      string `json:"NotAfter"`
	Token         string `json:"Token"`
	DisplayClaims struct {
		Xui []struct {
			Uhs string `json:"uhs"`
		} `json:"xui"`
	} `json:"DisplayClaims"`
}

type XSTSAuthRequest struct {
	Properties XSTSAuthProperties `json:"Properties"`
	RelyingParty string           `json:"RelyingParty"`
	TokenType    string           `json:"TokenType"`
}

type XSTSAuthProperties struct {
	SandboxId  string   `json:"SandboxId"`
	UserTokens []string `json:"UserTokens"`
}

// AuthenticateXboxLive exchanges Microsoft Access Token for Xbox Live Token
func AuthenticateXboxLive(msAccessToken string) (*XboxAuthResponse, error) {
	reqBody := XboxAuthRequest{
		Properties: XboxAuthProperties{
			AuthMethod: "RPS",
			SiteName:   "user.auth.xboxlive.com",
			RpsTicket:  "d=" + msAccessToken,
		},
		RelyingParty: "http://auth.xboxlive.com",
		TokenType:    "JWT",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", XboxLiveAuthURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xbox live auth failed: %s", resp.Status)
	}

	var authResp XboxAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}
	return &authResp, nil
}

// AuthenticateXSTS exchanges Xbox Live Token for XSTS Token
func AuthenticateXSTS(xboxToken string) (*XboxAuthResponse, error) {
	reqBody := XSTSAuthRequest{
		Properties: XSTSAuthProperties{
			SandboxId:  "RETAIL",
			UserTokens: []string{xboxToken},
		},
		RelyingParty: "rp://api.minecraftservices.com/",
		TokenType:    "JWT",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", XSTSAuthURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xsts auth failed: %s", resp.Status)
	}

	var authResp XboxAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}
	return &authResp, nil
}
