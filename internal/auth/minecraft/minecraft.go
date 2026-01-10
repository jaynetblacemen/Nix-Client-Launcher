package minecraft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	MinecraftAuthURL    = "https://api.minecraftservices.com/authentication/login_with_xbox"
	MinecraftProfileURL = "https://api.minecraftservices.com/minecraft/profile"
	MinecraftEntitlementsURL = "https://api.minecraftservices.com/entitlements/mcstore"
)

type MinecraftAuthRequest struct {
	IdentityToken string `json:"identityToken"`
}

type MinecraftAuthResponse struct {
	Username string `json:"username"`
	Roles    []struct {
		Role string `json:"role"`
	} `json:"roles"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type MinecraftProfile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Skins []struct {
		ID      string `json:"id"`
		State   string `json:"state"`
		URL     string `json:"url"`
		Variant string `json:"variant"`
	} `json:"skins"`
	Capes []struct {
		ID    string `json:"id"`
		State string `json:"state"`
		URL   string `json:"url"`
		Alias string `json:"alias"`
	} `json:"capes"`
}

type EntitlementsResponse struct {
	Items []struct {
		Name string `json:"name"`
	} `json:"items"`
}

// AuthenticateMinecraft exchanges XSTS Token and User Hash for Minecraft Access Token
func AuthenticateMinecraft(userHash, xstsToken string) (*MinecraftAuthResponse, error) {
	// Ensure the identityToken is formatted correctly: "XBL3.0 x=<user_hash>;<xsts_token>"
	reqBody := MinecraftAuthRequest{
		IdentityToken: fmt.Sprintf("XBL3.0 x=%s;%s", userHash, xstsToken),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", MinecraftAuthURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Nix-Client-Launcher/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read error body for debugging
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("minecraft auth failed: %s - Body: %s", resp.Status, string(bodyBytes))
	}

	var authResp MinecraftAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}
	return &authResp, nil
}

// CheckOwnership verifies if the user owns Minecraft Java Edition
func CheckOwnership(accessToken string) (bool, error) {
	req, err := http.NewRequest("GET", MinecraftEntitlementsURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "Nix-Client-Launcher/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("failed to check entitlements: %s - Body: %s", resp.Status, string(bodyBytes))
	}

	var entResp EntitlementsResponse
	if err := json.NewDecoder(resp.Body).Decode(&entResp); err != nil {
		return false, err
	}

	// Check for "product_minecraft" or "game_minecraft"
	for _, item := range entResp.Items {
		if item.Name == "product_minecraft" || item.Name == "game_minecraft" {
			return true, nil
		}
	}

	return false, nil
}

// GetProfile fetches the Minecraft profile (UUID, Username, Skins)
func GetProfile(accessToken string) (*MinecraftProfile, error) {
	req, err := http.NewRequest("GET", MinecraftProfileURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "Nix-Client-Launcher/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get profile: %s - Body: %s", resp.Status, string(bodyBytes))
	}

	var profile MinecraftProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}
	return &profile, nil
}
