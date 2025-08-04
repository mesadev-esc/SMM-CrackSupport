import { TargetName } from './generated';

import { common } from '$wailsjs/go/models';

export type ViewType = 'compact' | 'expanded';

export type LaunchButtonType = 'normal' | 'cat' | 'button';

export function installTypeToTargetName(installType: common.InstallType): TargetName {
  switch(installType) {
    case common.InstallType.WINDOWS:
      return TargetName.Windows;
    case common.InstallType.WINDOWS_CLIENT:
      return TargetName.Windows;
    case common.InstallType.WINDOWS_SERVER:
      return TargetName.WindowsServer;
    case common.InstallType.LINUX_SERVER:
      return TargetName.LinuxServer;
    default:
      throw new Error('Invalid install type');
  }
}

// Helper function to determine target name for an installation
// This is especially useful for custom installations where the target might not be immediately obvious
export function getInstallationTargetName(install: common.Installation): TargetName | null {
  // For custom installations, determine the target based on the installation type
  if (install.launcher === 'Custom') {
    // For cracked installations, we assume they are Windows client installations
    // unless explicitly identified as server installations
    switch (install.type) {
      case common.InstallType.WINDOWS:
      case common.InstallType.WINDOWS_CLIENT:
        return TargetName.Windows;
      case common.InstallType.WINDOWS_SERVER:
        return TargetName.WindowsServer;
      case common.InstallType.LINUX_SERVER:
        return TargetName.LinuxServer;
      default:
        // Default to Windows for custom installations
        // This handles cases where the type might not be properly set
        return TargetName.Windows;
    }
  }
  
  // For non-custom installations, use the standard mapping
  try {
    return installTypeToTargetName(install.type);
  } catch (e) {
    // If we can't determine the target, return null
    return null;
  }
}