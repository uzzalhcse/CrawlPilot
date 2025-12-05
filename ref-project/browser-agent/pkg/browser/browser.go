package browser

import (
	"crawer-agent/exp/v2/internal/utils"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"
)

const (
	CHROME_DEBUG_PORT         = 9222
	CHROME_DEFAULT_USER_AGENT = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"
)

var IN_DOCKER = os.Getenv("IN_DOCKER") == "true"

var CHROME_ARGS = []string{
	"--disable-blink-features=AutomationControlled",
	"--disable-sync",
	"--disable-cookie-encryption",
	"--no-first-run",
	"--no-default-browser-check",
	"--disable-infobars",
	"--disable-default-apps",
	"--remote-debugging-port=" + strconv.Itoa(CHROME_DEBUG_PORT),
	"--enable-experimental-extension-apis",
	"--disable-focus-on-load",
	"--disable-window-activation",
}

var CHROME_HEADLESS_ARGS = []string{
	"--headless=new",
	"--test-type",
}

var CHROME_DOCKER_ARGS = []string{
	"--no-sandbox",
	"--disable-gpu-sandbox",
	"--disable-setuid-sandbox",
	"--disable-dev-shm-usage",
}

type Browser struct {
	Config            BrowserConfig
	Playwright        *playwright.Playwright
	PlaywrightBrowser playwright.Browser
	chromeProcess     *os.Process
}

func NewBrowser(customConfig BrowserConfig) *Browser {
	config := NewBrowserConfig()
	for key, value := range customConfig {
		config[key] = value
	}
	return &Browser{
		Config: config,
	}
}

func (b *Browser) NewContext() *BrowserContext {
	return &BrowserContext{
		ContextID: uuid.New().String(),
		Config:    b.Config,
		Browser:   b,
		State:     &BrowserContextState{},
	}
}

func (b *Browser) GetPlaywrightBrowser() playwright.Browser {
	if b.PlaywrightBrowser == nil {
		return b.init()
	}
	return b.PlaywrightBrowser
}

func (b *Browser) Close(options ...playwright.BrowserCloseOptions) error {
	if b.chromeProcess != nil {
		b.chromeProcess.Kill()
	}
	if b.PlaywrightBrowser == nil {
		return nil
	}
	return b.PlaywrightBrowser.Close(options...)
}

func (b *Browser) init() playwright.Browser {
	pw, err := playwright.Run()
	if err != nil {
		panic(err)
	}
	b.Playwright = pw
	b.PlaywrightBrowser = b.setupBrowser(pw)
	return b.PlaywrightBrowser
}

func (b *Browser) setupBrowser(pw *playwright.Playwright) playwright.Browser {
	if cdpURL := b.Config["cdp_url"]; cdpURL != nil {
		return b.setupRemoteCDPBrowser(pw)
	}

	if headless, ok := b.Config["headless"].(bool); ok && headless {
		log.Warn("Headless mode may be detected by some sites")
	}

	if binaryPath := b.Config["browser_binary_path"]; binaryPath != nil {
		return b.setupUserProvidedBrowser(pw)
	}

	return b.setupBuiltinBrowser(pw)
}

func (b *Browser) setupRemoteCDPBrowser(pw *playwright.Playwright) playwright.Browser {
	cdpURL := b.Config["cdp_url"].(string)
	log.Infof("Connecting to remote browser via CDP %s", cdpURL)

	browser, err := pw.Chromium.ConnectOverCDP(cdpURL)
	if err != nil {
		panic(err)
	}
	return browser
}

func (b *Browser) setupUserProvidedBrowser(pw *playwright.Playwright) playwright.Browser {
	binaryPath := b.Config["browser_binary_path"].(string)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("Chrome binary not found: %s", binaryPath))
	}

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://localhost:9222/json/version")

	if err == nil && response.StatusCode == 200 {
		log.Info("Reusing existing browser on port 9222")
		browser, err := pw.Chromium.ConnectOverCDP("http://localhost:9222")
		if err != nil {
			panic(err)
		}
		return browser
	}

	chromeArgs := b.buildChromeArgs()
	userDataDir := getChromeUserDataDir()
	chromeArgs = append([]string{"--user-data-dir=" + userDataDir}, chromeArgs...)

	cmd := exec.Command(binaryPath, chromeArgs...)
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	b.chromeProcess = cmd.Process

	for i := 0; i < 10; i++ {
		response, err := client.Get("http://localhost:9222/json/version")
		if err == nil && response.StatusCode == 200 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	browser, err := pw.Chromium.ConnectOverCDP("http://localhost:9222")
	if err != nil {
		panic(err)
	}
	return browser
}

func (b *Browser) setupBuiltinBrowser(pw *playwright.Playwright) playwright.Browser {
	chromeArgs := b.buildChromeArgs()

	screenSize := getScreenResolution()
	offsetX, offsetY := getWindowAdjustments()

	if headless, ok := b.Config["headless"].(bool); ok && !headless {
		chromeArgs = append(chromeArgs,
			fmt.Sprintf("--window-position=%d,%d", offsetX, offsetY),
			fmt.Sprintf("--window-size=%d,%d", screenSize["width"], screenSize["height"]),
		)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:9222")
	if err != nil {
		for i, arg := range chromeArgs {
			if arg == "--remote-debugging-port=9222" {
				chromeArgs = append(chromeArgs[:i], chromeArgs[i+1:]...)
				break
			}
		}
	} else {
		ln.Close()
	}

	var proxySetting *playwright.Proxy
	if proxy, ok := b.Config["proxy"].(map[string]interface{}); ok {
		server := proxy["server"].(string)
		proxySetting = &playwright.Proxy{Server: server}

		if bypass, ok := proxy["bypass"].(string); ok {
			proxySetting.Bypass = &bypass
		}
		if username, ok := proxy["username"].(string); ok {
			proxySetting.Username = &username
		}
		if password, ok := proxy["password"].(string); ok {
			proxySetting.Password = &password
		}
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless:      playwright.Bool(utils.GetDefaultValue(b.Config, "headless", false)),
		Args:          chromeArgs,
		Proxy:         proxySetting,
		HandleSIGTERM: playwright.Bool(false),
		HandleSIGINT:  playwright.Bool(false),
	})
	if err != nil {
		panic(err)
	}
	return browser
}

func (b *Browser) buildChromeArgs() []string {
	args := make([]string, len(CHROME_ARGS))
	copy(args, CHROME_ARGS)

	if IN_DOCKER {
		args = append(args, CHROME_DOCKER_ARGS...)
	}

	if headless, ok := b.Config["headless"].(bool); ok && headless {
		args = append(args, CHROME_HEADLESS_ARGS...)
	}

	if extraArgs, ok := b.Config["extra_browser_args"].([]string); ok {
		args = append(args, extraArgs...)
	}

	return args
}

func getChromeUserDataDir() string {
	tempDir := os.TempDir()
	pid := os.Getpid()
	return filepath.Join(tempDir, "chrome-profile-"+strconv.Itoa(pid))
}
