package app

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/browser"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	appCommon "github.com/satisfactorymodding/SatisfactoryModManager/backend/common"
	"github.com/satisfactorymodding/SatisfactoryModManager/backend/ficsitcli"
	installCommon "github.com/satisfactorymodding/SatisfactoryModManager/backend/installfinders/common"
	customLauncher "github.com/satisfactorymodding/SatisfactoryModManager/backend/installfinders/launchers/custom"
	"github.com/satisfactorymodding/SatisfactoryModManager/backend/settings"
	"github.com/satisfactorymodding/SatisfactoryModManager/backend/utils"
)

func (a *app) ExpandMod() bool {
	width, height := wailsRuntime.WindowGetSize(appCommon.AppContext)
	wailsRuntime.WindowSetMinSize(appCommon.AppContext, utils.ExpandedMin.Width, utils.ExpandedMin.Height)
	wailsRuntime.WindowSetMaxSize(appCommon.AppContext, utils.ExpandedMax.Width, utils.ExpandedMax.Height)
	if !wailsRuntime.WindowIsMaximised(appCommon.AppContext) {
		wailsRuntime.WindowSetSize(appCommon.AppContext, max(width, settings.Settings.ExpandedSize.Width), height)
	}
	a.IsExpanded = true
	return true
}

func (a *app) UnexpandMod() bool {
	a.IsExpanded = false
	width, height := wailsRuntime.WindowGetSize(appCommon.AppContext)
	wailsRuntime.WindowSetMinSize(appCommon.AppContext, utils.UnexpandedMin.Width, utils.UnexpandedMin.Height)
	wailsRuntime.WindowSetMaxSize(appCommon.AppContext, utils.UnexpandedMax.Width, utils.UnexpandedMax.Height)
	if !wailsRuntime.WindowIsMaximised(appCommon.AppContext) {
		wailsRuntime.WindowSetSize(appCommon.AppContext, min(width, settings.Settings.UnexpandedSize.Width), height)
	}
	return true
}

type FileFilter struct {
	DisplayName string `json:"displayName"`
	Pattern     string `json:"pattern"`
}

type OpenDialogOptions struct {
	DefaultDirectory           string       `json:"defaultDirectory,omitempty"`
	DefaultFilename            string       `json:"defaultFilename,omitempty"`
	Title                      string       `json:"title,omitempty"`
	Filters                    []FileFilter `json:"filters,omitempty"`
	ShowHiddenFiles            bool         `json:"showHiddenFiles,omitempty"`
	CanCreateDirectories       bool         `json:"canCreateDirectories,omitempty"`
	ResolvesAliases            bool         `json:"resolvesAliases,omitempty"`
	TreatPackagesAsDirectories bool         `json:"treatPackagesAsDirectories,omitempty"`
}

func (a *app) OpenFileDialog(options OpenDialogOptions) (string, error) {
	wailsFilters := make([]wailsRuntime.FileFilter, len(options.Filters))
	for i, filter := range options.Filters {
		wailsFilters[i] = wailsRuntime.FileFilter{
			DisplayName: filter.DisplayName,
			Pattern:     filter.Pattern,
		}
	}
	wailsOptions := wailsRuntime.OpenDialogOptions{
		DefaultDirectory:           options.DefaultDirectory,
		DefaultFilename:            options.DefaultFilename,
		Title:                      options.Title,
		Filters:                    wailsFilters,
		ShowHiddenFiles:            options.ShowHiddenFiles,
		CanCreateDirectories:       options.CanCreateDirectories,
		ResolvesAliases:            options.ResolvesAliases,
		TreatPackagesAsDirectories: options.TreatPackagesAsDirectories,
	}
	file, err := wailsRuntime.OpenFileDialog(appCommon.AppContext, wailsOptions)
	if err != nil {
		return "", fmt.Errorf("failed to open file dialog: %w", err)
	}
	return file, nil
}

func (a *app) OpenDirectoryDialog(options OpenDialogOptions) (string, error) {
	wailsFilters := make([]wailsRuntime.FileFilter, len(options.Filters))
	for i, filter := range options.Filters {
		wailsFilters[i] = wailsRuntime.FileFilter{
			DisplayName: filter.DisplayName,
			Pattern:     filter.Pattern,
		}
	}
	wailsOptions := wailsRuntime.OpenDialogOptions{
		DefaultDirectory:           options.DefaultDirectory,
		DefaultFilename:            options.DefaultFilename,
		Title:                      options.Title,
		Filters:                    wailsFilters,
		ShowHiddenFiles:            options.ShowHiddenFiles,
		CanCreateDirectories:       options.CanCreateDirectories,
		ResolvesAliases:            options.ResolvesAliases,
		TreatPackagesAsDirectories: options.TreatPackagesAsDirectories,
	}
	file, err := wailsRuntime.OpenDirectoryDialog(appCommon.AppContext, wailsOptions)
	if err != nil {
		return "", fmt.Errorf("failed to open directory dialog: %w", err)
	}
	return file, nil
}

// AddCustomInstallation allows the frontend to add a custom installation
func (a *app) AddCustomInstallation(installPath string) (*installCommon.Installation, error) {
	// Attempt to create custom installation
	install, err := customLauncher.AddCustomInstallation(installPath)
	if err != nil {
		return nil, fmt.Errorf("failed to add custom installation at path: %s - error: %w", installPath, err)
	}
	
	// Add installation to ficsit-cli
	err = ficsitcli.FicsitCLI.AddInstallation(
		install.Path,
		install.LaunchPath,
		string(install.Type),
		string(install.Branch),
		install.Version,
		install.Launcher,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register installation in ficsit-cli: %w", err)
	}
	
	// Ensure selected installation is valid
	ficsitcli.FicsitCLI.EnsureSelectedInstallationIsValid()
	
	// Log successful addition
	slog.Info("successfully added custom installation", 
		slog.String("path", installPath),
		slog.String("launcher", install.Launcher),
		slog.String("installation_type", string(install.Type)),
		slog.String("branch", string(install.Branch)),
	)
	
	return install, nil
}

// ClearInstallations removes all registered installations
func (a *app) ClearInstallations() error {
	err := ficsitcli.FicsitCLI.ClearInstallations()
	if err != nil {
		return fmt.Errorf("failed to clear installations: %w", err)
	}
	
	slog.Info("successfully cleared all installations")
	return nil
}

func (a *app) ExternalInstallMod(modID, version string) {
	wailsRuntime.EventsEmit(appCommon.AppContext, "externalInstallMod", modID, version)
}

func (a *app) ExternalImportProfile(path string) {
	wailsRuntime.EventsEmit(appCommon.AppContext, "externalImportProfile", path)
}

func (a *app) Show() {
	wailsRuntime.WindowUnminimise(appCommon.AppContext)
	wailsRuntime.Show(appCommon.AppContext)
}

func (a *app) OpenExternal(input string) {
	err := browser.OpenFile(input)
	if err != nil {
		slog.Error("failed to open external", slog.Any("error", err), slog.String("path", input))
	}
}

func (a *app) ShowInExplorer(path string) error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("explorer", "/select,", path)
		_, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to open explorer: %w", err)
		}
		return nil
	case "linux":
		conn, err := dbus.SessionBus()
		if err != nil {
			return fmt.Errorf("failed to connect to session bus: %w", err)
		}
		fileURL := "file://" + path
		obj := conn.Object("org.freedesktop.FileManager1", "/org/freedesktop/FileManager1")
		call := obj.Call("org.freedesktop.FileManager1.ShowItems", 0, []string{fileURL}, "")
		if call.Err != nil {
			return fmt.Errorf("failed to open file manager: %w", call.Err)
		}
		return nil
	case "darwin":
		cmd := exec.Command("open", "-R", path)
		_, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to open finder: %w", err)
		}
		return nil
	}
	return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
}
