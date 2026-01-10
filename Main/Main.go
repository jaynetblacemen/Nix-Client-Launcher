package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"

	"Nix-Client-Launcher/internal/auth"
	"Nix-Client-Launcher/internal/storage"
)

func main() {
	// Force Wayland if available. 
	// Note: The user must have qt6-wayland (or qt5-wayland) installed on their system.
	os.Setenv("QT_QPA_PLATFORM", "wayland;xcb")
	
	// Set the desktop file name for Wayland icon association
	gui.QGuiApplication_SetDesktopFileName("nix-client-launcher")

	// Create the application
	app := widgets.NewQApplication(len(os.Args), os.Args)

	// Locate media directory
	mediaDir := findMediaDir()
	
	// Set Application Icon
	appIconPath := filepath.Join(mediaDir, "logo.png")
	if _, err := os.Stat(appIconPath); err == nil {
		appIcon := gui.NewQIcon5(appIconPath)
		app.SetWindowIcon(appIcon)
	} else {
		fmt.Println("Warning: Could not find app icon at", appIconPath)
	}

	// Check for existing login
	account, err := storage.LoadAccount()
	if err == nil && account.Tokens.MinecraftAccessToken != "" {
		// Check expiry
		if time.Now().After(account.Tokens.MinecraftExpiry) {
			// Refresh
			refreshedAccount, err := auth.RefreshLogin(account)
			if err != nil {
				fmt.Println("Failed to refresh token, requiring login:", err)
				showLoginWindow(mediaDir)
			} else {
				showMainWindow(refreshedAccount)
			}
		} else {
			// Valid token
			showMainWindow(account)
		}
	} else {
		showLoginWindow(mediaDir)
	}

	// Execute the application
	widgets.QApplication_Exec()
}

// findMediaDir attempts to locate the media directory relative to the working directory
func findMediaDir() string {
	wd, _ := os.Getwd()
	
	// Check current directory
	path := filepath.Join(wd, "media")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	
	// Check parent directory (useful if running from Main/ subdirectory)
	path = filepath.Join(wd, "..", "media")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	
	// Check project root assumption (hardcoded fallback)
	// Adjust this if your project path is fixed
	return filepath.Join(wd, "media") 
}

func showLoginWindow(mediaDir string) {
	window := widgets.NewQMainWindow(nil, 0)
	window.SetWindowTitle("Nix Client Launcher - Login")
	window.SetFixedSize2(400, 500)

	centralWidget := widgets.NewQWidget(window, 0)
	window.SetCentralWidget(centralWidget)

	layout := widgets.NewQVBoxLayout()
	centralWidget.SetLayout(layout)
	layout.SetSpacing(20)

	// Logo
	logoLabel := widgets.NewQLabel(centralWidget, 0)
	
	logoPath := filepath.Join(mediaDir, "microsoft_logo.png")
	fmt.Println("Loading Microsoft logo from:", logoPath) // Debug print
	
	logoPixmap := gui.NewQPixmap3(logoPath, "", core.Qt__AutoColor)
	if !logoPixmap.IsNull() {
		logoLabel.SetPixmap(logoPixmap.Scaled2(100, 100, core.Qt__KeepAspectRatio, core.Qt__SmoothTransformation))
	} else {
		fmt.Println("Failed to load logo image. File exists?", fileExists(logoPath))
		logoLabel.SetText("Microsoft")
		logoLabel.SetStyleSheet("font-size: 24px; font-weight: bold; color: #555;")
	}
	logoLabel.SetAlignment(core.Qt__AlignCenter)
	layout.AddWidget(logoLabel, 0, core.Qt__AlignCenter)

	// Text
	textLabel := widgets.NewQLabel(centralWidget, 0)
	textLabel.SetText("Login with your Minecraft account")
	textLabel.SetAlignment(core.Qt__AlignCenter)
	layout.AddWidget(textLabel, 0, core.Qt__AlignCenter)

	// Button
	loginButton := widgets.NewQPushButton2("Login", centralWidget)
	loginButton.SetFixedWidth(200)
	loginButton.ConnectClicked(func(checked bool) {
		// Start Device Flow in a goroutine to prevent freezing
		go func() {
			flow, err := auth.StartDeviceLogin()
			if err != nil {
				timer := core.NewQTimer(nil)
				timer.SetSingleShot(true)
				timer.ConnectTimeout(func() {
					widgets.QMessageBox_Critical(window, "Error", fmt.Sprintf("Failed to start login: %v", err), widgets.QMessageBox__Ok, widgets.QMessageBox__Ok)
				})
				timer.Start(0)
				return
			}

			// Show Instructions Dialog on Main Thread
			timer := core.NewQTimer(nil)
			timer.SetSingleShot(true)
			timer.ConnectTimeout(func() {
				dialog := widgets.NewQDialog(window, 0)
				dialog.SetWindowTitle("Microsoft Login")
				dialog.SetFixedSize2(400, 250)

				dLayout := widgets.NewQVBoxLayout()
				dialog.SetLayout(dLayout)

				infoLabel := widgets.NewQLabel(dialog, 0)
				infoLabel.SetText(fmt.Sprintf("1. Click the button below to open the login page.\n2. Enter this code: %s", flow.UserCode))
				infoLabel.SetAlignment(core.Qt__AlignCenter)
				infoLabel.SetWordWrap(true)
				infoLabel.SetStyleSheet("font-size: 14px; font-weight: bold;")
				dLayout.AddWidget(infoLabel, 0, core.Qt__AlignCenter)

				// Copy Code Button
				copyButton := widgets.NewQPushButton2("Copy Code", dialog)
				copyButton.ConnectClicked(func(checked bool) {
					gui.QGuiApplication_Clipboard().SetText(flow.UserCode, gui.QClipboard__Clipboard)
				})
				dLayout.AddWidget(copyButton, 0, core.Qt__AlignCenter)

				// Open Browser Button
				openButton := widgets.NewQPushButton2("Open Login Page", dialog)
				openButton.ConnectClicked(func(checked bool) {
					gui.QDesktopServices_OpenUrl(core.NewQUrl3(flow.AuthURL, core.QUrl__TolerantMode))
				})
				dLayout.AddWidget(openButton, 0, core.Qt__AlignCenter)

				dialog.Show()

				// Start Polling in Background
				go func() {
					ctx := context.Background()
					account, err := flow.WaitForLogin(ctx)

					if err != nil {
						fmt.Println("Login Error:", err)
						timer := core.NewQTimer(nil)
						timer.SetSingleShot(true)
						timer.ConnectTimeout(func() {
							dialog.Close()
							widgets.QMessageBox_Critical(window, "Login Error", fmt.Sprintf("Login failed: %v", err), widgets.QMessageBox__Ok, widgets.QMessageBox__Ok)
						})
						timer.Start(0)
						return
					}

					// Success
					fmt.Println("Login Successful for:", account.Profile.Name)
					timer := core.NewQTimer(nil)
					timer.SetSingleShot(true)
					timer.ConnectTimeout(func() {
						dialog.Close()
						widgets.QMessageBox_Information(window, "Login Successful", "Please restart the launcher.", widgets.QMessageBox__Ok, widgets.QMessageBox__Ok)
						window.Close()
					})
					timer.Start(0)
				}()
			})
			timer.Start(0)
		}()
	})
	layout.AddWidget(loginButton, 0, core.Qt__AlignCenter)

	window.Show()
}

func showMainWindow(account *storage.AccountData) {
	window := widgets.NewQMainWindow(nil, 0)
	window.SetWindowTitle("Nix Client Launcher")
	window.SetFixedSize2(800, 600)

	centralWidget := widgets.NewQWidget(window, 0)
	window.SetCentralWidget(centralWidget)
	
	layout := widgets.NewQVBoxLayout()
	centralWidget.SetLayout(layout)

	welcomeLabel := widgets.NewQLabel(centralWidget, 0)
	welcomeLabel.SetText(fmt.Sprintf("Welcome, %s!", account.Profile.Name))
	welcomeLabel.SetAlignment(core.Qt__AlignCenter)
	layout.AddWidget(welcomeLabel, 0, core.Qt__AlignCenter)

	playButton := widgets.NewQPushButton2("Play", centralWidget)
	layout.AddWidget(playButton, 0, core.Qt__AlignCenter)

	window.Show()
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
