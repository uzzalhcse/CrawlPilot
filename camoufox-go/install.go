package camoufox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// GetExecutablePath finds the Camoufox executable path.
// It checks common installation locations and the system PATH.
func GetExecutablePath() (string, error) {
	// Check if CAMOUFOX_PATH environment variable is set
	if envPath := os.Getenv("CAMOUFOX_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	// Check common installation paths first (faster than calling Python)
	homeDir, _ := os.UserHomeDir()
	possiblePaths := []string{
		// Standard cache location (where 'python -m camoufox fetch' installs)
		filepath.Join(homeDir, ".cache", "camoufox", "camoufox-bin"),
		filepath.Join(homeDir, ".camoufox", "camoufox", "launch"),
		filepath.Join(homeDir, ".local", "share", "camoufox", "launch"),
		"/usr/local/bin/camoufox",
		"/usr/bin/camoufox",
	}

	// Add platform-specific paths
	switch runtime.GOOS {
	case "darwin":
		possiblePaths = append(possiblePaths,
			filepath.Join(homeDir, "Library", "Caches", "camoufox", "camoufox-bin"),
			filepath.Join(homeDir, "Library", "Application Support", "camoufox", "launch"),
			"/Applications/Camoufox.app/Contents/MacOS/camoufox",
		)
	case "windows":
		appData := os.Getenv("LOCALAPPDATA")
		if appData != "" {
			possiblePaths = append(possiblePaths,
				filepath.Join(appData, "camoufox", "camoufox-bin.exe"),
				filepath.Join(appData, "camoufox", "launch.exe"),
			)
		}
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Try to find via Python package as fallback
	pythonPath, err := findViaPython()
	if err == nil && pythonPath != "" {
		return pythonPath, nil
	}

	return "", fmt.Errorf("camoufox executable not found. Run: camoufox.Install() or 'pip install camoufox && python -m camoufox fetch'")
}

// findViaPython tries to find Camoufox path using the Python package
func findViaPython() (string, error) {
	cmd := exec.Command("python3", "-c", "from camoufox.pkgman import launch_path; print(launch_path())")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", fmt.Errorf("empty path returned")
	}

	if _, err := os.Stat(path); err != nil {
		return "", err
	}

	return path, nil
}

// Install downloads and installs all required dependencies for Camoufox.
// This includes:
// - browserforge Python package (for fingerprint generation)
// - camoufox Python package (from coryking's Firefox 142 fork)
// - Camoufox browser binary
func Install() error {
	// Check if Python is available
	if _, err := exec.LookPath("python3"); err != nil {
		return fmt.Errorf("python3 not found. Please install Python 3.8+")
	}

	// Check if pip is available
	if _, err := exec.LookPath("pip"); err != nil {
		if _, err := exec.LookPath("pip3"); err != nil {
			return fmt.Errorf("pip not found. Please install pip")
		}
	}

	// Install browserforge if not present
	if err := installPythonPackage("browserforge", "browserforge"); err != nil {
		return err
	}

	// Install camoufox Python package from coryking's fork (Firefox 142)
	// This uninstalls any existing version first
	fmt.Println("Installing camoufox (Firefox 142) from coryking's fork...")
	pipCmd := "pip"
	if _, err := exec.LookPath("pip"); err != nil {
		pipCmd = "pip3"
	}

	// Uninstall old version first
	uninstallCmd := exec.Command(pipCmd, "uninstall", "camoufox", "-y")
	uninstallCmd.Run() // Ignore errors if not installed

	// Install from coryking's fork
	installCmd := exec.Command(pipCmd, "install",
		"git+https://github.com/coryking/camoufox.git@v142.0.1-fork.27#subdirectory=pythonlib")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install camoufox from fork: %w", err)
	}

	// Run the Camoufox browser installer (uses 'python -m camoufox fetch')
	fmt.Println("Installing Camoufox browser binary (Firefox 142)...")
	cmd := exec.Command("python3", "-m", "camoufox", "fetch")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install camoufox browser: %w", err)
	}

	fmt.Println("✅ Camoufox (Firefox 142) installed successfully!")
	return nil
}

// installPythonPackage installs a Python package if not already installed
func installPythonPackage(importName, packageName string) error {
	// Check if package is already installed
	checkCmd := exec.Command("python3", "-c", fmt.Sprintf("import %s", importName))
	if err := checkCmd.Run(); err != nil {
		fmt.Printf("Installing %s Python package...\n", packageName)

		// Try pip first, then pip3
		pipCmd := "pip"
		if _, err := exec.LookPath("pip"); err != nil {
			pipCmd = "pip3"
		}

		installCmd := exec.Command(pipCmd, "install", packageName)
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install %s package: %w", packageName, err)
		}
		fmt.Printf("✅ %s installed\n", packageName)
	} else {
		fmt.Printf("✅ %s already installed\n", packageName)
	}
	return nil
}

// InstallDependenciesOnly installs only the Python dependencies without the browser binary.
// Useful for environments where the browser is pre-installed.
func InstallDependenciesOnly() error {
	// Check if Python is available
	if _, err := exec.LookPath("python3"); err != nil {
		return fmt.Errorf("python3 not found. Please install Python 3.8+")
	}

	if err := installPythonPackage("browserforge", "browserforge"); err != nil {
		return err
	}

	if err := installPythonPackage("camoufox", "camoufox"); err != nil {
		return err
	}

	fmt.Println("✅ All Python dependencies installed!")
	return nil
}

// IsInstalled checks if Camoufox is installed
func IsInstalled() bool {
	_, err := GetExecutablePath()
	return err == nil
}
