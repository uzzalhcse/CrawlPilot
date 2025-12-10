package camoufox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// VirtualDisplay manages a Xvfb virtual display for headless mode on Linux
type VirtualDisplay struct {
	display int
	proc    *exec.Cmd
	mu      sync.Mutex
	debug   bool
}

// NewVirtualDisplay creates a new virtual display manager
func NewVirtualDisplay(debug bool) *VirtualDisplay {
	return &VirtualDisplay{
		display: -1,
		debug:   debug,
	}
}

// Start starts the virtual display (Xvfb) and returns the display string
func (vd *VirtualDisplay) Start() (string, error) {
	vd.mu.Lock()
	defer vd.mu.Unlock()

	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("virtual display is only supported on Linux")
	}

	// Find Xvfb path
	xvfbPath, err := exec.LookPath("Xvfb")
	if err != nil {
		return "", fmt.Errorf("Xvfb not found. Please install: apt-get install xvfb")
	}

	// Find a free display number
	vd.display = vd.findFreeDisplay()

	displayStr := fmt.Sprintf(":%d", vd.display)

	// Build Xvfb command
	args := []string{
		displayStr,
		"-screen", "0", "1920x1080x24",
		"-ac",
		"-nolisten", "tcp",
		"+extension", "GLX",
		"-shmem",
		"-nocursor",
	}

	if vd.debug {
		fmt.Printf("[VirtualDisplay] Starting Xvfb: %s %s\n", xvfbPath, strings.Join(args, " "))
	}

	vd.proc = exec.Command(xvfbPath, args...)
	vd.proc.Stdout = nil
	vd.proc.Stderr = nil

	if err := vd.proc.Start(); err != nil {
		return "", fmt.Errorf("failed to start Xvfb: %w", err)
	}

	if vd.debug {
		fmt.Printf("[VirtualDisplay] Started on display %s\n", displayStr)
	}

	return displayStr, nil
}

// Stop terminates the virtual display
func (vd *VirtualDisplay) Stop() {
	vd.mu.Lock()
	defer vd.mu.Unlock()

	if vd.proc != nil && vd.proc.Process != nil {
		if vd.debug {
			fmt.Printf("[VirtualDisplay] Stopping display :%d\n", vd.display)
		}
		vd.proc.Process.Kill()
		vd.proc.Wait()
		vd.proc = nil
	}
}

// Display returns the display string (e.g., ":99")
func (vd *VirtualDisplay) Display() string {
	if vd.display < 0 {
		return ""
	}
	return fmt.Sprintf(":%d", vd.display)
}

// findFreeDisplay finds an available display number
func (vd *VirtualDisplay) findFreeDisplay() int {
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = "/tmp"
	}

	// Look for existing X lock files
	lockFiles, _ := filepath.Glob(filepath.Join(tmpDir, ".X*-lock"))

	usedDisplays := make(map[int]bool)
	for _, lockFile := range lockFiles {
		// Extract display number from filename like ".X99-lock"
		base := filepath.Base(lockFile)
		if strings.HasPrefix(base, ".X") && strings.HasSuffix(base, "-lock") {
			numStr := strings.TrimPrefix(base, ".X")
			numStr = strings.TrimSuffix(numStr, "-lock")
			if num, err := strconv.Atoi(numStr); err == nil {
				usedDisplays[num] = true
			}
		}
	}

	// Find first free display starting from 99
	for display := 99; display < 200; display++ {
		if !usedDisplays[display] {
			return display
		}
	}

	return 99 // Fallback
}

// IsLinux returns true if running on Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// CanUseVirtualDisplay checks if virtual display is available
func CanUseVirtualDisplay() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	_, err := exec.LookPath("Xvfb")
	return err == nil
}
