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

// PerformLogin handles the full login flow
func PerformLogin(ctx context.Context) (*storage.AccountData, error) {
	// 1. Start Callback Server
	code, err := microsoft.StartCallbackServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth code: %v", err)
	}

	// 2. Exchange Code for Microsoft Token
	msToken, err := microsoft.ExchangeCode(code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %v", err)
	}

	// 3. Xbox Live Auth
	xboxResp, err := xbox.AuthenticateXboxLive(msToken.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("xbox auth failed: %v", err)
	}

	// 4. XSTS Auth
	xstsResp, err := xbox.AuthenticateXSTS(xboxResp.Token)
	if err != nil {
		return nil, fmt.Errorf("xsts auth failed: %v", err)
	}

	// Extract User Hash (uhs)
	if len(xstsResp.DisplayClaims.Xui) == 0 {
		return nil, fmt.Errorf("no user hash found in xsts response")
	}
	userHash := xstsResp.DisplayClaims.Xui[0].Uhs

	// 5. Minecraft Auth
	mcResp, err := minecraft.AuthenticateMinecraft(userHash, xstsResp.Token)
	if err != nil {
		return nil, fmt.Errorf("minecraft auth failed: %v", err)
	}

	// 6. Check Ownership
	ownsGame, err := minecraft.CheckOwnership(mcResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("ownership check failed: %v", err)
	}
	if !ownsGame {
		return nil, fmt.Errorf("user does not own Minecraft Java Edition")
	}

	// 7. Get Profile
	profile, err := minecraft.GetProfile(mcResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %v", err)
	}

	// 8. Prepare Account Data
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

	// 9. Save Account
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

	// Re-authenticate Xbox/Minecraft flow (similar to login but using new token)
	// For brevity, we reuse the flow logic or you can extract the common parts.
	// Here we just re-run the chain from Xbox auth down.
	
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
