package ficsitcli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/satisfactorymodding/ficsit-cli/cli"
	resolver "github.com/satisfactorymodding/ficsit-resolver"

	"github.com/satisfactorymodding/SatisfactoryModManager/backend/installfinders/common"
)

func (f *ficsitCLI) initInstallations() error {
	for _, install := range f.ficsitCli.Installations.Installations {
		f.installationMetadata.Store(install.Path, installationMetadata{
			State: InstallStateUnknown,
			Info:  nil,
		})
	}

	err := f.initLocalInstallationsMetadata()
	if err != nil {
		return fmt.Errorf("failed to initialize found installations: %w", err)
	}

	// This may take a while, so we do it in the background
	go f.initRemoteServerInstallationsMetadata()

	// Even if the remote server metadata is not yet available, we can still do this
	f.ensureSelectedInstallationIsValid()

	return nil
}

func (f *ficsitCLI) ensureSelectedInstallationIsValid() {
	if !f.isValidInstall(f.ficsitCli.Installations.SelectedInstallation) {
		filteredInstalls := f.GetInstallations()
		if len(filteredInstalls) > 0 {
			f.ficsitCli.Installations.SelectedInstallation = filteredInstalls[0]
			err := f.ficsitCli.Installations.Save()
			if err != nil {
				slog.Error("failed to save selected installation", slog.Any("error", err))
			}
			f.EmitGlobals()
		}
	}
}

func (f *ficsitCLI) GetInstallations() []string {
	installations := make([]string, 0, len(f.ficsitCli.Installations.Installations))
	for _, installation := range f.ficsitCli.Installations.Installations {
		if !f.isValidInstall(installation.Path) {
			continue
		}
		installations = append(installations, installation.Path)
	}
	return installations
}

func (f *ficsitCLI) GetInstallationsMetadata() map[string]installationMetadata {
	rawMap := make(map[string]installationMetadata, len(f.ficsitCli.Installations.Installations))
	f.installationMetadata.Range(func(key string, value installationMetadata) bool {
		rawMap[key] = value
		return true
	})
	return rawMap
}

func (f *ficsitCLI) GetCurrentInstallationMetadata() installationMetadata {
	meta, _ := f.installationMetadata.Load(f.ficsitCli.Installations.SelectedInstallation)
	return meta
}

func (f *ficsitCLI) GetInvalidInstalls() []string {
	result := []string{}
	for _, err := range f.installFindErrors {
		var installFindErr common.InstallFindError
		if errors.As(err, &installFindErr) {
			result = append(result, installFindErr.Path)
		}
	}
	return result
}

func (f *ficsitCLI) GetInstallation(path string) *cli.Installation {
	return f.ficsitCli.Installations.GetInstallation(path)
}

func (f *ficsitCLI) SelectInstall(path string) error {
	return f.action(ActionSelectInstall, newSimpleItem(path), func(l *slog.Logger, _ chan<- taskUpdate) error {
		if !f.isValidInstall(path) {
			return fmt.Errorf("invalid installation: %s", path)
		}
		if f.ficsitCli.Installations.SelectedInstallation == path {
			return nil
		}
		installation := f.ficsitCli.Installations.GetInstallation(path)
		if installation == nil {
			return fmt.Errorf("installation %s not found", path)
		}

		f.ficsitCli.Installations.SelectedInstallation = path
		err := f.ficsitCli.Installations.Save()
		if err != nil {
			l.Error("failed to save selected installation", slog.Any("error", err))
		}

		f.EmitGlobals()
		f.EmitModsChange()
		return nil
	})
}

func (f *ficsitCLI) GetSelectedInstall() *cli.Installation {
	return f.ficsitCli.Installations.GetInstallation(f.ficsitCli.Installations.SelectedInstallation)
}

func (f *ficsitCLI) SetModsEnabled(enabled bool) error {
	var item ProgressItem
	if enabled {
		item = newSimpleItem("true")
	} else {
		item = newSimpleItem("false")
	}
	return f.action(ActionToggleMods, item, func(l *slog.Logger, taskUpdates chan<- taskUpdate) error {
		selectedInstallation := f.GetSelectedInstall()

		if selectedInstallation == nil {
			return fmt.Errorf("no installation selected")
		}

		l = l.With(slog.String("install", selectedInstallation.Path))

		selectedInstallation.Vanilla = !enabled
		err := f.ficsitCli.Installations.Save()
		if err != nil {
			l.Error("failed to save vanilla state of install", slog.Any("error", err))
		}

		f.EmitGlobals()

		installErr := f.apply(l, taskUpdates)

		if installErr != nil {
			l.Error("failed to validate install", slog.Any("error", installErr))
			return installErr
		}

		return nil
	})
}

func (f *ficsitCLI) GetModsEnabled() bool {
	selectedInstallation := f.GetSelectedInstall()
	return selectedInstallation == nil || !selectedInstallation.Vanilla
}

func (f *ficsitCLI) GetSelectedInstallProfileMods() map[string]cli.ProfileMod {
	selectedInstallation := f.GetSelectedInstall()
	if selectedInstallation == nil {
		return make(map[string]cli.ProfileMod)
	}
	profile := f.GetProfile(selectedInstallation.Profile)
	if profile == nil {
		return make(map[string]cli.ProfileMod)
	}
	return profile.Mods
}

func (f *ficsitCLI) GetSelectedInstallLockfileMods() (map[string]resolver.LockedMod, error) {
	selectedInstallation := f.GetSelectedInstall()
	if selectedInstallation == nil {
		return make(map[string]resolver.LockedMod), nil
	}
	lockfile, err := selectedInstallation.LockFile(f.ficsitCli)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	if lockfile == nil {
		return make(map[string]resolver.LockedMod), nil
	}
	return lockfile.Mods, nil
}

func (f *ficsitCLI) GetSelectedInstallLockfile() (*resolver.LockFile, error) {
	selectedInstallation := f.GetSelectedInstall()
	if selectedInstallation == nil {
		return nil, nil
	}
	lockfile, err := selectedInstallation.LockFile(f.ficsitCli)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return lockfile, nil
}

// LaunchGame launches the currently selected game installation
func (f *ficsitCLI) LaunchGame() {
	selectedInstallation := f.GetSelectedInstall()
	if selectedInstallation == nil {
		slog.Error("no installation selected")
		return
	}
	
	// Check if this is a custom installation by looking at the metadata
	metadata, ok := f.installationMetadata.Load(selectedInstallation.Path)
	if !ok || metadata.Info == nil {
		slog.Error("no metadata for installation")
		return
	}
	
	// For cracked installations (Custom launcher), we need to launch through Steam
	if metadata.Info.Launcher == "Custom" {
		// For cracked installations, we need to launch through Steam
		launchPath := f.findCustomInstallationExecutable(selectedInstallation.Path)
		if launchPath != "" {
			// Launch through Steam with the custom executable path
			out, cmd, err := f.executeLaunchCommand([]string{"cmd", "/C", "start", "steam://launch/1895860", launchPath})
			if err != nil {
				slog.Error("failed to launch game through Steam", slog.Any("error", err), slog.String("cmd", cmd), slog.String("output", string(out)))
				return
			}
			return
		}
		slog.Error("could not find executable for custom installation")
		return
	}
	
	// If we have metadata and a valid launch path, use it (for non-custom installations)
	if len(metadata.Info.LaunchPath) > 0 {
		out, cmd, err := f.executeLaunchCommand(metadata.Info.LaunchPath)
		if err != nil {
			slog.Error("failed to launch game", slog.Any("error", err), slog.String("cmd", cmd), slog.String("output", string(out)))
			return
		}
		return
	}
	
	slog.Error("could not determine launch path for installation")
}

// findCustomInstallationExecutable looks for the game executable in common locations for cracked versions
func (f *ficsitCLI) findCustomInstallationExecutable(installPath string) string {
	// Common executable locations for cracked Satisfactory installations
	possiblePaths := []string{
		filepath.Join(installPath, "FactoryGame", "Binaries", "Win64", "FactoryGame-Win64-Shipping.exe"),
		filepath.Join(installPath, "FactoryGame", "Binaries", "Win64", "FactoryGame.exe"),
		filepath.Join(installPath, "FactoryGame.exe"),
		filepath.Join(installPath, "FactoryGameSteam.exe"),
	}
	
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	return ""
}
