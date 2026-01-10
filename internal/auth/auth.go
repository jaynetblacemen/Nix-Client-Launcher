package auth

import (
	"context"
	"fmt"
	"time"

	"Nix-Client-Launcher/internal/auth/microsoft"
	"Nix-Client-Launcher/internal/auth/minecraft"
	"Nix-Client-Launcher/internal/auth/xbox"
	"Nix-Client-Launcher/internal/storage"
)

// DeviceLoginFlow encapsulates the state needed for the Device Code flow
type DeviceLoginFlow struct {
	DeviceCode string
	UserCode   string
	AuthURL    string
	Interval   int
}

// StartDeviceLogin initiates the flow and returns the details to show the user
func StartDeviceLogin() (*DeviceLoginFlow, error) {
	resp, err := microsoft.StartDeviceFlow()
	if err != nil {
		return nil, err
	}
	return &DeviceLoginFlow{
		DeviceCode: resp.DeviceCode,
		UserCode:   resp.UserCode,
		AuthURL:    resp.VerificationURI,
		Interval:   resp.Interval,
	}, nil
}

// WaitForLogin polls for the token and completes the chain
func (f *DeviceLoginFlow) WaitForLogin(ctx context.Context) (*storage.AccountData, error) {
	// 1. Poll for Microsoft Token
	msToken, err := microsoft.PollForToken(f.DeviceCode, f.Interval)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %v", err)
	}

	// 2. Xbox Live Auth
	xboxResp, err := xbox.AuthenticateXboxLive(msToken.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("xbox auth failed: %v", err)
	}

	// 3. XSTS Auth
	xstsResp, err := xbox.AuthenticateXSTS(xboxResp.Token)
	if err != nil {
		return nil, fmt.Errorf("xsts auth failed: %v", err)
	}

	// Extract User Hash (uhs)
	if len(xstsResp.DisplayClaims.Xui) == 0 {
		return nil, fmt.Errorf("no user hash found in xsts response")
	}
	userHash := xstsResp.DisplayClaims.Xui[0].Uhs

	// 4. Minecraft Auth
	mcResp, err := minecraft.AuthenticateMinecraft(userHash, xstsResp.Token)
	if err != nil {
		return nil, fmt.Errorf("minecraft auth failed: %v", err)
	}

	// 5. Check Ownership
	ownsGame, err := minecraft.CheckOwnership(mcResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("ownership check failed: %v", err)
	}
	if !ownsGame {
		return nil, fmt.Errorf("user does not own Minecraft Java Edition")
	}

	// 6. Get Profile
	profile, err := minecraft.GetProfile(mcResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %v", err)
	}

	// 7. Prepare Account Data
	account := storage.AccountData{
		Tokens: storage.AuthTokens{
			MicrosoftAccessToken:  msToken.AccessToken,
			MicrosoftRefreshToken: msToken.RefreshToken,
			MicrosoftExpiry:       time.Now().Add(time.Duration(msToken.ExpiresIn) * time.Second),
			MinecraftAccessToken:  mcResp.AccessToken,
			MinecraftExpiry:       time.Now().Add(time.Duration(mcResp.ExpiresIn) * time.Second),
		},
		Profile: storage.MinecraftProfile{
			ID:   profile.ID,
			Name: profile.Name,
		},
	}

	// 8. Save Account
	if err := storage.SaveAccount(account); err != nil {
		return nil, fmt.Errorf("failed to save account: %v", err)
	}

	return &account, nil
}

// RefreshLogin handles token refreshing
func RefreshLogin(account *storage.AccountData) (*storage.AccountData, error) {
	// Refresh Microsoft Token
	msToken, err := microsoft.RefreshToken(account.Tokens.MicrosoftRefreshToken)
	if err != nil {
		return nil, err
	}

	// Re-authenticate Xbox/Minecraft flow
	xboxResp, err := xbox.AuthenticateXboxLive(msToken.AccessToken)
	if err != nil {
		return nil, err
	}

	xstsResp, err := xbox.AuthenticateXSTS(xboxResp.Token)
	if err != nil {
		return nil, err
	}

	userHash := xstsResp.DisplayClaims.Xui[0].Uhs

	mcResp, err := minecraft.AuthenticateMinecraft(userHash, xstsResp.Token)
	if err != nil {
		return nil, err
	}

	// Update Account Data
	account.Tokens.MicrosoftAccessToken = msToken.AccessToken
	account.Tokens.MicrosoftRefreshToken = msToken.RefreshToken
	account.Tokens.MicrosoftExpiry = time.Now().Add(time.Duration(msToken.ExpiresIn) * time.Second)
	account.Tokens.MinecraftAccessToken = mcResp.AccessToken
	account.Tokens.MinecraftExpiry = time.Now().Add(time.Duration(mcResp.ExpiresIn) * time.Second)

	if err := storage.SaveAccount(*account); err != nil {
		return nil, err
	}

	return account, nil
}
