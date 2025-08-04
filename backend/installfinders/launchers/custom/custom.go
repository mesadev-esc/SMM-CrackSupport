package custom

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/satisfactorymodding/SatisfactoryModManager/backend/installfinders/common"
	"github.com/satisfactorymodding/SatisfactoryModManager/backend/installfinders/launchers"
)

const (
	LauncherName = "Custom"
)

func init() {
	launchers.Add("custom", FindInstallationsCustom)
}

func FindInstallationsCustom() ([]*common.Installation, []error) {
	// This finder doesn't automatically find installations
	// It's used for manually added custom installations
	return nil, nil
}

// AddCustomInstallation allows adding a custom installation manually
func AddCustomInstallation(installPath string) (*common.Installation, error) {
	// Verify the path exists
	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("installation path does not exist: %s", installPath)
	}

	// Try to determine if it's a client or server installation
	// For now, we'll assume it's a Windows client installation
	// In the future, this could be made more sophisticated

	var launchPath []string
	
	// Check for the game executable in common locations including cracked versions
	possibleExecutables := []string{
		filepath.Join(installPath, "FactoryGame.exe"),                                    // Root directory
		filepath.Join(installPath, "Binaries", "Win64", "FactoryGame.exe"),              // Standard Epic/Steam install
		filepath.Join(installPath, "Engine", "Binaries", "Win64", "FactoryGame.exe"),    // Some custom installs
		filepath.Join(installPath, "FactoryGameSteam.exe"),                              // Steam variant
		filepath.Join(installPath, "FactoryGameEGS.exe"),                                // Epic variant
		// Add more paths for cracked versions
		filepath.Join(installPath, "Binaries", "Win64", "FactoryGameSteam.exe"),
		filepath.Join(installPath, "Binaries", "Win64", "FactoryGameEGS.exe"),
		// Additional paths for cracked versions
		filepath.Join(installPath, "FactoryGame", "Binaries", "Win64", "FactoryGame.exe"),
		filepath.Join(installPath, "FactoryGame", "Binaries", "Win64", "FactoryGame-Win64-Shipping.exe"),
	}
	
	executableFound := false
	var foundExecutable string
	for _, exePath := range possibleExecutables {
		if _, err := os.Stat(exePath); err == nil {
			foundExecutable = exePath
			launchPath = []string{exePath}
			executableFound = true
			break
		}
	}
	
	// If we still can't find the executable, check if it might be in a subdirectory
	if !executableFound {
		// Walk the directory to find executables
		err := filepath.Walk(installPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Continue walking despite errors
			}
			
			// Skip if not a file
			if info.IsDir() {
				return nil
			}
			
			// Check if it's a game executable
			name := strings.ToLower(info.Name())
			if name == "factorygame.exe" || name == "factorygamesteam.exe" || name == "factorygameegs.exe" ||
				name == "factorygame-win64-shipping.exe" || name == "factorygamesteam-win64-shipping.exe" ||
				name == "factorygameegs-win64-shipping.exe" {
				// Found a potential executable
				foundExecutable = path
				launchPath = []string{path}
				executableFound = true
				return filepath.SkipDir // Stop walking
			}
			
			return nil
		})
		
		if err != nil {
			slog.Debug("error walking directory for executables", slog.String("installPath", installPath), slog.Any("error", err))
		}
	}
	
	if !executableFound {
		// Even if we can't find the executable, we still allow the installation
		// but with a warning. The user can fix the launch path later.
		slog.Warn("could not find FactoryGame.exe in common locations", slog.String("installPath", installPath))
		launchPath = []string{installPath} // Fallback to the install path
	}

	// Determine installation type based on executable name and path
	installType := common.InstallTypeWindowsClient
	if executableFound {
		execName := strings.ToLower(filepath.Base(foundExecutable))
		// Check if this is a server installation
		if strings.Contains(execName, "server") {
			installType = common.InstallTypeWindowsServer
		}
		// Additional check for server in path
		if strings.Contains(strings.ToLower(foundExecutable), "server") {
			installType = common.InstallTypeWindowsServer
		}
	} else {
		// If no executable found, but it's a custom install, default to Windows client
		// This is common with cracked versions where the executable might have a different name
		installType = common.InstallTypeWindowsClient
	}

	// Try to get the game version by checking various version files
	// This is important for cracked installations to properly show mod compatibility
	version := 0
	if executableFound {
		version = getGameVersionFromFile(installPath)
	}

	install := &common.Installation{
		Path:       installPath,
		Version:    version, // Use detected version or 0 if not found
		Type:       installType,
		Location:   common.LocationTypeLocal,
		Branch:     common.BranchStable,
		Launcher:   LauncherName,
		LaunchPath: launchPath,
	}

	return install, nil
}

// getGameVersionFromFile attempts to read the game version from various possible version files
// This is especially important for cracked installations that might not have the standard file locations
func getGameVersionFromFile(installPath string) int {
	// Possible version file locations for different game variants
	versionFiles := []string{
		filepath.Join(installPath, "Engine", "Binaries", "Win64", "FactoryGame-Win64-Shipping.version"),
		filepath.Join(installPath, "Engine", "Binaries", "Win64", "FactoryGameSteam-Win64-Shipping.version"),
		filepath.Join(installPath, "Engine", "Binaries", "Win64", "FactoryGameEGS-Win64-Shipping.version"),
		filepath.Join(installPath, "FactoryGame", "Engine", "Binaries", "Win64", "FactoryGame-Win64-Shipping.version"),
		filepath.Join(installPath, "FactoryGame", "Engine", "Binaries", "Win64", "FactoryGameSteam-Win64-Shipping.version"),
		filepath.Join(installPath, "FactoryGame", "Engine", "Binaries", "Win64", "FactoryGameEGS-Win64-Shipping.version"),
		// Generic version file location
		filepath.Join(installPath, "Engine", "Build", "Build.version"),
	}

	for _, versionFile := range versionFiles {
		if _, err := os.Stat(versionFile); err == nil {
			// File exists, try to read it
			versionData, err := readVersionFile(versionFile)
			if err == nil && versionData.Changelist > 0 {
				return versionData.Changelist
			}
		}
	}

	// If we can't find a version file or read it properly, return 0
	return 0
}

// readVersionFile reads and parses a game version file
func readVersionFile(versionFilePath string) (*common.GameVersionFile, error) {
	versionFile, err := os.ReadFile(versionFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read version file %s: %w", versionFilePath, err)
	}

	var versionData common.GameVersionFile
	if err := json.Unmarshal(versionFile, &versionData); err != nil {
		return nil, fmt.Errorf("failed to parse version file %s: %w", versionFilePath, err)
	}

	return &versionData, nil
}