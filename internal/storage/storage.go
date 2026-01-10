package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type MinecraftProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AuthTokens struct {
	MicrosoftAccessToken  string    `json:"ms_access_token"`
	MicrosoftRefreshToken string    `json:"ms_refresh_token"`
	MicrosoftExpiry       time.Time `json:"ms_expiry"`
	MinecraftAccessToken  string    `json:"mc_access_token"`
	MinecraftExpiry       time.Time `json:"mc_expiry"` // Usually 24h
}

type AccountData struct {
	Tokens  AuthTokens       `json:"tokens"`
	Profile MinecraftProfile `json:"profile"`
}

func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(configDir, "NixClientLauncher")
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}
	return path, nil
}

func SaveAccount(data AccountData) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(dir, "accounts.json"))
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func LoadAccount() (*AccountData, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filepath.Join(dir, "accounts.json"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data AccountData
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
