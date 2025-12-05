package browser

import (
	"crawer-agent/exp/v2/internal/dom"
	"crawer-agent/exp/v2/internal/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	neturl "net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/playwright-community/playwright-go"
)

type BrowserSession struct {
	ActiveTab   playwright.Page
	Context     playwright.BrowserContext
	CachedState *BrowserState
}

func NewSession(context playwright.BrowserContext, cachedState *BrowserState) *BrowserSession {
	return &BrowserSession{
		Context:     context,
		CachedState: cachedState,
	}
}

type BrowserContextState struct {
	TargetID *string
}

type BrowserContext struct {
	ContextID string
	Config    BrowserConfig
	Browser   *Browser
	Session   *BrowserSession
	State     *BrowserContextState
	ActiveTab playwright.Page
}

func (bc *BrowserContext) GetState(cacheClickableElements bool) *BrowserState {
	bc.waitForPageLoad()
	page := bc.GetCurrentPage()

	session := bc.GetSession()
	updatedState := bc.getUpdatedState(page)

	session.CachedState = updatedState
	return updatedState
}

func (bc *BrowserContext) getUpdatedState(page playwright.Page) *BrowserState {
	domService := dom.NewService(page)
	content, err := domService.GetClickableElements(
		utils.GetDefaultValue(bc.Config, "highlight_elements", true),
		-1,
		utils.GetDefaultValue(bc.Config, "viewport_expansion", 0),
	)
	if err != nil {
		log.Errorf("Failed to get clickable elements: %s", err)
		// Return a minimal state instead of continuing with nil
		title, _ := page.Title()
		return &BrowserState{
			ElementTree: &dom.DOMElementNode{
				TagName:    "body",
				Xpath:      "/html/body",
				Attributes: map[string]string{},
				Children:   []dom.DOMBaseNode{},
				IsVisible:  true,
			},
			SelectorMap:   &dom.SelectorMap{},
			URL:           page.URL(),
			Title:         title,
			Tabs:          bc.GetTabsInfo(),
			Screenshot:    nil,
			PixelAbove:    0,
			PixelBelow:    0,
			BrowserErrors: []string{err.Error()},
		}
	}

	tabsInfo := bc.GetTabsInfo()
	screenshot, err := bc.TakeScreenshot(false)
	if err != nil {
		log.Warnf("Failed to take screenshot: %s", err)
	}

	pixelsAbove, pixelsBelow, err := bc.GetScrollInfo(page)
	if err != nil {
		log.Warnf("Failed to get scroll info: %s", err)
	}

	title, _ := page.Title()
	return &BrowserState{
		ElementTree:   content.ElementTree,
		SelectorMap:   content.SelectorMap,
		URL:           page.URL(),
		Title:         title,
		Tabs:          tabsInfo,
		Screenshot:    screenshot,
		PixelAbove:    pixelsAbove,
		PixelBelow:    pixelsBelow,
		BrowserErrors: []string{},
	}
}

func (bc *BrowserContext) TakeScreenshot(fullPage bool) (*string, error) {
	page := bc.GetCurrentPage()
	page.BringToFront()
	page.WaitForLoadState()

	screenshot, err := page.Screenshot(playwright.PageScreenshotOptions{
		FullPage:   playwright.Bool(fullPage),
		Animations: playwright.ScreenshotAnimationsDisabled,
	})
	if err != nil {
		return nil, err
	}

	screenshotBase64 := base64.StdEncoding.EncodeToString(screenshot)
	return &screenshotBase64, nil
}

func (bc *BrowserContext) GetScrollInfo(page playwright.Page) (int, int, error) {
	scrollY, err := page.Evaluate("() => window.scrollY")
	if err != nil {
		return 0, 0, err
	}
	viewportHeight, err := page.Evaluate("() => window.innerHeight")
	if err != nil {
		return 0, 0, err
	}
	totalHeight, err := page.Evaluate("() => document.documentElement.scrollHeight")
	if err != nil {
		return 0, 0, err
	}

	pixelsAbove, _ := ParseNumberToInt(scrollY)
	totalHeightInt, _ := ParseNumberToInt(totalHeight)
	viewportHeightInt, _ := ParseNumberToInt(viewportHeight)
	pixelsBelow := totalHeightInt - (pixelsAbove + viewportHeightInt)

	return pixelsAbove, pixelsBelow, nil
}

func (bc *BrowserContext) GetSession() *BrowserSession {
	if bc.Session == nil {
		session, err := bc.initializeSession()
		if err != nil {
			panic(err)
		}
		return session
	}
	return bc.Session
}

func (bc *BrowserContext) GetCurrentPage() playwright.Page {
	session := bc.GetSession()
	return bc.getCurrentPage(session)
}

func (bc *BrowserContext) Close() {
	if bc.Session == nil {
		return
	}

	if cookiesFile, ok := bc.Config["cookies_file"].(string); ok && cookiesFile != "" {
		bc.SaveCookies()
	}

	if keepAlive, ok := bc.Config["keep_alive"].(bool); !ok || !keepAlive {
		bc.Session.Context.Close()
	}

	bc.Session = nil
	bc.ActiveTab = nil
}

func (bc *BrowserContext) GetSelectorMap() *dom.SelectorMap {
	session := bc.GetSession()
	if session.CachedState == nil {
		return nil
	}
	return session.CachedState.SelectorMap
}

func (bc *BrowserContext) GetDOMElementByIndex(index int) (*dom.DOMElementNode, error) {
	selectorMap := bc.GetSelectorMap()
	if selectorMap == nil || (*selectorMap)[index] == nil {
		return nil, fmt.Errorf("element with index %d does not exist", index)
	}
	return (*selectorMap)[index], nil
}

func (bc *BrowserContext) GetLocateElement(element *dom.DOMElementNode) playwright.Locator {
	currentPage := bc.GetCurrentPage()
	var currentFrame playwright.FrameLocator

	var parents []*dom.DOMElementNode
	current := element
	for current.Parent != nil {
		parents = append(parents, current.Parent)
		current = current.Parent
	}
	slices.Reverse(parents)

	var iframes []*dom.DOMElementNode
	for _, item := range parents {
		if item.TagName == "iframe" {
			iframes = append(iframes, item)
		}
	}

	includeDynamic := utils.GetDefaultValue(bc.Config, "include_dynamic_attributes", true)
	for _, parent := range iframes {
		cssSelector := dom.EnhancedCSSSelector(parent, includeDynamic)
		if currentFrame != nil {
			currentFrame = currentFrame.FrameLocator(cssSelector)
		} else {
			currentFrame = currentPage.FrameLocator(cssSelector)
		}
	}

	cssSelector := dom.EnhancedCSSSelector(element, includeDynamic)
	if currentFrame != nil {
		return currentFrame.Locator(cssSelector)
	}
	return currentPage.Locator(cssSelector)
}

func (bc *BrowserContext) NavigateTo(url string) error {
	if !bc.isURLAllowed(url) {
		return &BrowserError{Message: "Navigation to non-allowed URL: " + url}
	}

	page := bc.GetCurrentPage()
	page.Goto(url)
	page.WaitForLoadState()
	return nil
}

func (bc *BrowserContext) ClickElementNode(elementNode *dom.DOMElementNode) (*string, error) {
	page := bc.GetCurrentPage()
	elementLocator := bc.GetLocateElement(elementNode)
	if elementLocator == nil {
		return nil, &BrowserError{Message: "Element not found"}
	}

	performClick := func(clickFunc func() error) (*string, error) {
		if saveDownloadPath, ok := bc.Config["save_downloads_path"].(string); ok {
			downloadInfo, err := page.ExpectDownload(clickFunc, playwright.PageExpectDownloadOptions{
				Timeout: playwright.Float(3000),
			})
			if err != nil {
				if strings.HasPrefix(err.Error(), "timeout:") {
					page.WaitForLoadState()
					bc.checkAndHandleNavigation(page)
					return nil, nil
				}
				return nil, err
			}

			filename := downloadInfo.SuggestedFilename()
			uniqueFilename := bc.getUniqueFilename(saveDownloadPath, filename)
			downloadPath := filepath.Join(saveDownloadPath, uniqueFilename)

			if err := downloadInfo.SaveAs(downloadPath); err != nil {
				return nil, err
			}
			return &downloadPath, nil
		}

		newPage, err := bc.GetSession().Context.ExpectPage(func() error {
			return clickFunc()
		}, playwright.BrowserContextExpectPageOptions{Timeout: playwright.Float(1500)})

		if err != nil {
			if strings.HasPrefix(err.Error(), "timeout:") {
				page.WaitForLoadState()
				bc.checkAndHandleNavigation(page)
				return nil, nil
			}
			return nil, err
		}

		newPage.WaitForLoadState()
		bc.checkAndHandleNavigation(newPage)
		return nil, nil
	}

	return performClick(func() error {
		return elementLocator.First().Click(playwright.LocatorClickOptions{
			Timeout: playwright.Float(1500),
		})
	})
}

func (bc *BrowserContext) InputTextElementNode(elementNode *dom.DOMElementNode, text string) error {
	locator := bc.GetLocateElement(elementNode)
	if locator == nil {
		return &BrowserError{Message: "Element not found"}
	}

	selectorState := playwright.WaitForSelectorState("visible")
	locator.WaitFor(playwright.LocatorWaitForOptions{
		State:   &selectorState,
		Timeout: playwright.Float(1000),
	})

	if isHidden, err := locator.IsHidden(); err == nil && !isHidden {
		locator.ScrollIntoViewIfNeeded()
	}

	tagNameAny, _ := locator.Evaluate("el => el.tagName.toLowerCase()", nil)
	tagName := tagNameAny.(string)

	if tagName == "input" || tagName == "textarea" {
		locator.Evaluate("el => { el.textContent = ''; el.value = ''; }", nil)
		return locator.Fill(text)
	}

	return locator.Fill(text)
}

func (bc *BrowserContext) initializeSession() (*BrowserSession, error) {
	log.Debugf("Initializing browser context: %s", bc.ContextID)

	pwBrowser := bc.Browser.GetPlaywrightBrowser()
	context, err := bc.createContext(pwBrowser)
	if err != nil {
		return nil, err
	}

	bc.Session = &BrowserSession{
		Context:     context,
		CachedState: nil,
	}

	pages := context.Pages()
	var activePage playwright.Page

	if len(pages) > 0 && !strings.HasPrefix(pages[0].URL(), "chrome://") {
		activePage = pages[0]
	} else {
		activePage, err = context.NewPage()
		if err != nil {
			return nil, err
		}
		activePage.Goto("about:blank")
	}

	activePage.BringToFront()
	activePage.WaitForLoadState()
	bc.ActiveTab = activePage

	return bc.Session, nil
}

func (bc *BrowserContext) createContext(browser playwright.Browser) (playwright.BrowserContext, error) {
	var context playwright.BrowserContext
	var err error

	if bc.Browser.Config["cdp_url"] != nil && len(browser.Contexts()) > 0 {
		context = browser.Contexts()[0]
	} else {
		context, err = browser.NewContext(playwright.BrowserNewContextOptions{
			NoViewport:        playwright.Bool(true),
			UserAgent:         playwright.String(utils.GetDefaultValue(bc.Browser.Config, "user_agent", "")),
			JavaScriptEnabled: playwright.Bool(true),
			Locale:            playwright.String(utils.GetDefaultValue(bc.Browser.Config, "locale", "")),
		})
		if err != nil {
			return nil, err
		}
	}

	bc.LoadCookies(context)

	initScript := `
		Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
		Object.defineProperty(navigator, 'languages', { get: () => ['en-US'] });
		Object.defineProperty(navigator, 'plugins', { get: () => [1, 2, 3, 4, 5] });
		window.chrome = { runtime: {} };
	`
	context.AddInitScript(playwright.Script{Content: &initScript})

	return context, nil
}

func (bc *BrowserContext) getCurrentPage(session *BrowserSession) playwright.Page {
	pages := session.Context.Pages()

	if bc.ActiveTab != nil && !bc.ActiveTab.IsClosed() && slices.Contains(pages, bc.ActiveTab) {
		return bc.ActiveTab
	}

	var nonExtensionPages []playwright.Page
	for _, page := range pages {
		if !strings.HasPrefix(page.URL(), "chrome-extension://") && !strings.HasPrefix(page.URL(), "chrome://") {
			nonExtensionPages = append(nonExtensionPages, page)
		}
	}

	if len(nonExtensionPages) > 0 {
		return nonExtensionPages[len(nonExtensionPages)-1]
	}

	page, _ := session.Context.NewPage()
	bc.ActiveTab = page
	return page
}

func (bc *BrowserContext) GetTabsInfo() []*TabInfo {
	session := bc.GetSession()
	var tabsInfo []*TabInfo

	for pageID, page := range session.Context.Pages() {
		title, _ := page.Title()
		tabsInfo = append(tabsInfo, &TabInfo{
			PageID:       pageID,
			URL:          page.URL(),
			Title:        title,
			ParentPageID: nil,
		})
	}
	return tabsInfo
}

func (bc *BrowserContext) SwitchToTab(pageID int) error {
	session := bc.GetSession()
	pages := session.Context.Pages()

	if pageID >= len(pages) {
		return &BrowserError{Message: fmt.Sprintf("No tab found with page_id: %d", pageID)}
	}

	for pageID < 0 {
		pageID += len(pages)
	}

	page := pages[pageID]
	if !bc.isURLAllowed(page.URL()) {
		return NewURLNotAllowedError(page.URL())
	}

	bc.ActiveTab = page
	page.BringToFront()
	page.WaitForLoadState()
	return nil
}

func (bc *BrowserContext) GoBack() error {
	page := bc.GetCurrentPage()
	_, err := page.GoBack(playwright.PageGoBackOptions{
		Timeout:   playwright.Float(1000),
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	return err
}

func (bc *BrowserContext) CreateNewTab(url string) error {
	if len(url) > 0 && !bc.isURLAllowed(url) {
		return &BrowserError{Message: "Cannot create tab with non-allowed URL: " + url}
	}

	session := bc.GetSession()
	newPage, err := session.Context.NewPage()
	if err != nil {
		return err
	}

	bc.ActiveTab = newPage
	newPage.WaitForLoadState()

	if len(url) > 0 {
		_, err := newPage.Goto(url)
		if err != nil {
			return err
		}
	}

	return nil
}

func (bc *BrowserContext) LoadCookies(context playwright.BrowserContext) error {
	cookiesFile, ok := bc.Config["cookies_file"].(string)
	if !ok || !utils.FileExists(cookiesFile) {
		return nil
	}

	f, err := os.Open(cookiesFile)
	if err != nil {
		return err
	}
	defer f.Close()

	var cookies []playwright.OptionalCookie
	if err := json.NewDecoder(f).Decode(&cookies); err != nil {
		return err
	}

	log.Infof("Loaded %d cookies from %s", len(cookies), cookiesFile)
	return context.AddCookies(cookies)
}

func (bc *BrowserContext) SaveCookies() error {
	cookiesFile, ok := bc.Config["cookies_file"].(string)
	if !ok || bc.Session == nil || bc.Session.Context == nil {
		return nil
	}

	cookies, err := bc.Session.Context.Cookies()
	if err != nil {
		return err
	}

	dirname := filepath.Dir(cookiesFile)
	if dirname != "" {
		os.MkdirAll(dirname, 0755)
	}

	f, err := os.Create(cookiesFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(cookies)
}

func (bc *BrowserContext) getUniqueFilename(directory, filename string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	newFilename := filename
	counter := 1

	for {
		fullPath := filepath.Join(directory, newFilename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			break
		}
		newFilename = fmt.Sprintf("%s (%d)%s", base, counter, ext)
		counter++
	}
	return newFilename
}

func (bc *BrowserContext) isURLAllowed(url string) bool {
	allowedDomainsText, ok := bc.Config["allowed_domains"].(string)
	if !ok || allowedDomainsText == "" {
		return true
	}

	if url == "about:blank" {
		return true
	}

	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return false
	}

	domain := strings.ToLower(parsedURL.Host)
	if colonIdx := strings.Index(domain, ":"); colonIdx != -1 {
		domain = domain[:colonIdx]
	}

	allowedDomains := strings.Split(allowedDomainsText, ",")
	for _, allowedDomain := range allowedDomains {
		allowedDomain = strings.ToLower(strings.TrimSpace(allowedDomain))
		if domain == allowedDomain || strings.HasSuffix(domain, "."+allowedDomain) {
			return true
		}
	}

	return false
}

func (bc *BrowserContext) checkAndHandleNavigation(page playwright.Page) error {
	if !bc.isURLAllowed(page.URL()) {
		log.Warnf("Navigation to non-allowed URL detected: %s", page.URL())
		return bc.GoBack()
	}
	return nil
}

func (bc *BrowserContext) waitForPageLoad() {
	page := bc.GetCurrentPage()
	bc.checkAndHandleNavigation(page)
}
func (bc *BrowserContext) RemoveHighlights() {
	// Implementation to remove visual highlights
	page := bc.GetCurrentPage()
	if page != nil {
		page.Evaluate(`
			try {
				const container = document.getElementById('playwright-highlight-container');
				if (container) container.remove();
			} catch (e) {}
		`)
	}
}
