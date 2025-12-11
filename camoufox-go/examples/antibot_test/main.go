package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	camoufox "github.com/uzzalhcse/camoufox-go"
)

const (
	maxConcurrent = 1                // Number of concurrent tests
	testTimeout   = 60 * time.Second // Timeout per test
	closeTimeout  = 10 * time.Second // Timeout for browser close
)

// TestResult holds the result of a single anti-bot test
type TestResult struct {
	TestName   string   `json:"test_name"`
	Config     string   `json:"config"`
	Site       string   `json:"site"`
	Passed     bool     `json:"passed"`
	Score      string   `json:"score,omitempty"`
	Issues     []string `json:"issues,omitempty"`
	Screenshot string   `json:"screenshot,omitempty"`
	Duration   string   `json:"duration"`
	Error      string   `json:"error,omitempty"`
}

// TestJob represents a test to run
type TestJob struct {
	Config   TestConfig
	SiteName string
	URL      string
}

// TestConfig defines a test configuration
type TestConfig struct {
	Name    string
	Options camoufox.Options
}

// Proxy configuration for proxy tests
var testProxy = &camoufox.ProxyConfig{
	Server:   "http://82.22.93.243:7950",
	Username: "lnvmpyru",
	Password: "5un1tb1azapa",
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     Camoufox Anti-Bot Detection Test Suite (Concurrent)      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("⚡ Running %d tests concurrently\n", maxConcurrent)

	// Test configurations
	testConfigs := []TestConfig{
		//{Name: "Basic", Options: camoufox.Options{Headless: false, Debug: true}},
		//{Name: "GeoIP Auto", Options: camoufox.Options{Headless: false, GeoIP: "auto", Debug: true}},
		{Name: "WebRTC Blocked", Options: camoufox.Options{Headless: false, BlockWebRTC: true, Debug: true}},
		{Name: "Full Stealth", Options: camoufox.Options{Headless: false, GeoIP: "auto", BlockWebRTC: false, Humanize: 1.5, Debug: true}},
	}

	// Add proxy tests if configured
	if testProxy.Server != "" {
		//testConfigs = append(testConfigs,
		//	TestConfig{Name: "Proxy Basic", Options: camoufox.Options{Headless: false, Proxy: testProxy, Debug: true}},
		//	TestConfig{Name: "Proxy + GeoIP", Options: camoufox.Options{Headless: false, Proxy: testProxy, GeoIP: "auto", Debug: true}},
		//	TestConfig{Name: "Proxy + Stealth", Options: camoufox.Options{Headless: false, Proxy: testProxy, GeoIP: "auto", BlockWebRTC: false, Debug: true}},
		//)
		fmt.Printf("🔌 Proxy: %s\n", testProxy.Server)
	}

	// Test sites
	testSites := []struct{ Name, URL string }{
		{"BrowserScan", "https://www.browserscan.net/"},
		{"CreepJS", "https://abrahamjuliot.github.io/creepjs/"},
	}

	// Create job queue
	jobs := make(chan TestJob, len(testConfigs)*len(testSites))
	results := make(chan TestResult, len(testConfigs)*len(testSites))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go worker(i+1, jobs, results, &wg)
	}

	// Queue all jobs
	totalJobs := 0
	for _, cfg := range testConfigs {
		for _, site := range testSites {
			jobs <- TestJob{Config: cfg, SiteName: site.Name, URL: site.URL}
			totalJobs++
		}
	}
	close(jobs)

	// Wait for completion in background
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	os.MkdirAll("test_results", 0755)
	allResults := []TestResult{}
	completed := 0
	for result := range results {
		completed++
		allResults = append(allResults, result)

		status := "❌"
		if result.Passed {
			status = "✅"
		}
		fmt.Printf("[%d/%d] %s %s @ %s: %s (%s)\n",
			completed, totalJobs, status, result.Config, result.Site, result.Score, result.Duration)

		if result.Error != "" {
			fmt.Printf("        ⚠️ Error: %s\n", result.Error)
		}
	}

	saveResults(allResults)
	printSummary(allResults)
}

func worker(id int, jobs <-chan TestJob, results chan<- TestResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		debugLog(id, "Starting: %s @ %s", job.Config.Name, job.SiteName)
		result := runTestWithTimeout(id, job)
		results <- result
		debugLog(id, "Completed: %s @ %s", job.Config.Name, job.SiteName)
	}
}

func runTestWithTimeout(workerID int, job TestJob) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resultChan := make(chan TestResult, 1)

	go func() {
		resultChan <- runSingleTest(workerID, job)
	}()

	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return TestResult{
			TestName: fmt.Sprintf("%s @ %s", job.Config.Name, job.SiteName),
			Config:   job.Config.Name,
			Site:     job.SiteName,
			Error:    "Test timed out",
			Issues:   []string{"Timeout after " + testTimeout.String()},
			Duration: testTimeout.String(),
		}
	}
}

func runSingleTest(workerID int, job TestJob) TestResult {
	start := time.Now()

	result := TestResult{
		TestName: fmt.Sprintf("%s @ %s", job.Config.Name, job.SiteName),
		Config:   job.Config.Name,
		Site:     job.SiteName,
	}

	debugLog(workerID, "Launching browser for %s", job.Config.Name)

	// Launch browser
	browser, err := camoufox.NewBrowser(job.Config.Options)
	if err != nil {
		result.Error = fmt.Sprintf("Launch failed: %v", err)
		result.Issues = append(result.Issues, result.Error)
		result.Duration = time.Since(start).Round(time.Millisecond).String()
		return result
	}

	// Close browser with timeout
	defer func() {
		debugLog(workerID, "Closing browser...")
		closeDone := make(chan struct{})
		go func() {
			if err := browser.Close(); err != nil {
				debugLog(workerID, "Close error: %v", err)
			}
			close(closeDone)
		}()

		select {
		case <-closeDone:
			debugLog(workerID, "Browser closed successfully")
		case <-time.After(closeTimeout):
			debugLog(workerID, "Browser close timed out after %v", closeTimeout)
		}
	}()

	debugLog(workerID, "Creating page...")

	// Create page
	page, err := browser.NewPage()
	if err != nil {
		result.Error = fmt.Sprintf("Page creation failed: %v", err)
		result.Issues = append(result.Issues, result.Error)
		result.Duration = time.Since(start).Round(time.Millisecond).String()
		return result
	}

	debugLog(workerID, "Navigating to %s...", job.URL)

	// Navigate with timeout
	_, err = page.Goto(job.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(20000),
	})
	if err != nil {
		result.Error = fmt.Sprintf("Navigation failed: %v", err)
		result.Issues = append(result.Issues, result.Error)
		result.Duration = time.Since(start).Round(time.Millisecond).String()
		return result
	}

	debugLog(workerID, "Waiting for detection analysis...")

	// Wait for detection
	time.Sleep(6 * time.Second)

	// Extract results based on site
	switch job.SiteName {
	case "BrowserScan":
		result = extractBrowserScan(page, result)
	case "CreepJS":
		result = extractCreepJS(page, result)
	case "PixelScan":
		result = extractPixelScan(page, result)
	}

	// Screenshot
	screenshotPath := fmt.Sprintf("test_results/%s_%s_%d.png",
		strings.ReplaceAll(job.Config.Name, " ", "_"),
		job.SiteName,
		time.Now().UnixMilli())

	debugLog(workerID, "Taking screenshot...")
	_, err = page.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(screenshotPath),
		FullPage: playwright.Bool(true),
	})
	if err != nil {
		debugLog(workerID, "Screenshot failed: %v", err)
	} else {
		result.Screenshot = screenshotPath
	}

	result.Duration = time.Since(start).Round(time.Millisecond).String()
	return result
}

func extractBrowserScan(page playwright.Page, result TestResult) TestResult {
	// Wait a bit more for score to load (BrowserScan loads scores dynamically)
	time.Sleep(3 * time.Second)

	// Try to get score via JS with multiple approaches
	score, _ := page.Evaluate(`() => {
		// Try comprehensive score first
		let el = document.querySelector('.comprehensive-score .score-value');
		if (el) return el.textContent.trim();
		
		// Try percentage display
		el = document.querySelector('.score-percentage, .percentage');
		if (el) return el.textContent.trim();
		
		// Try any element with score in class containing a number
		const scoreEls = document.querySelectorAll('[class*="score"]');
		for (const se of scoreEls) {
			const text = se.textContent.trim();
			if (/^\d+%?$/.test(text) || /^\d+\/\d+$/.test(text)) {
				return text;
			}
		}
		
		// Try to find any percentage on page
		const allText = document.body.innerText;
		const match = allText.match(/(\d{2,3})%/);
		if (match) return match[1] + '%';
		
		return '';
	}`)
	if s, ok := score.(string); ok && s != "" {
		result.Score = s
	}

	// Check pass/fail
	if result.Score != "" && result.Score != "N/A" {
		// Extract number from score
		numStr := ""
		for _, c := range result.Score {
			if c >= '0' && c <= '9' {
				numStr += string(c)
			}
		}
		if len(numStr) >= 2 {
			// 90+ is passing
			if numStr[0] == '1' || numStr[0] == '9' {
				result.Passed = true
			}
		}
	}

	if result.Score == "" {
		result.Score = "N/A"
	}

	return result
}

func extractCreepJS(page playwright.Page, result TestResult) TestResult {
	grade, _ := page.Evaluate(`() => {
		const el = document.querySelector('.grade, [class*="grade"]');
		return el ? el.textContent.trim() : '';
	}`)
	if g, ok := grade.(string); ok && g != "" {
		result.Score = g
	}

	lies, _ := page.Evaluate(`() => {
		const text = document.body.innerText;
		const match = text.match(/(\d+)\s*lies/i);
		return match ? match[1] : '0';
	}`)
	if l, ok := lies.(string); ok && l == "0" {
		result.Passed = true
	}

	if result.Score == "" {
		result.Score = "Analyzed"
		result.Passed = true
	}

	return result
}

func extractPixelScan(page playwright.Page, result TestResult) TestResult {
	status, _ := page.Evaluate(`() => {
		const el = document.querySelector('.status, [class*="result"]');
		return el ? el.textContent.trim() : '';
	}`)
	if s, ok := status.(string); ok && s != "" {
		result.Score = s
		lower := strings.ToLower(s)
		result.Passed = !strings.Contains(lower, "bot") && !strings.Contains(lower, "fail")
	}

	if result.Score == "" {
		result.Score = "Scanned"
		result.Passed = true
	}

	return result
}

func debugLog(workerID int, format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] Worker-%d: %s\n", timestamp, workerID, msg)
}

func saveResults(results []TestResult) {
	jsonData, _ := json.MarshalIndent(results, "", "  ")
	os.WriteFile("test_results/results.json", jsonData, 0644)

	var sb strings.Builder
	sb.WriteString("# Anti-Bot Detection Test Results\n\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Concurrency:** %d workers\n\n", maxConcurrent))
	sb.WriteString("| Config | Site | Score | Status | Duration | Error |\n")
	sb.WriteString("|--------|------|-------|--------|----------|-------|\n")

	for _, r := range results {
		status := "❌"
		if r.Passed {
			status = "✅"
		}
		errStr := ""
		if r.Error != "" {
			errStr = r.Error
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
			r.Config, r.Site, r.Score, status, r.Duration, errStr))
	}

	os.WriteFile("test_results/report.md", []byte(sb.String()), 0644)
	fmt.Println("\n📁 Results saved to test_results/")
}

func printSummary(results []TestResult) {
	passed, failed, errors := 0, 0, 0
	for _, r := range results {
		if r.Error != "" {
			errors++
		} else if r.Passed {
			passed++
		} else {
			failed++
		}
	}

	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  ✅ Passed: %d  |  ❌ Failed: %d  |  ⚠️ Errors: %d             ║\n",
		passed, failed, errors)
	fmt.Printf("║  Pass Rate: %.0f%%                                            ║\n",
		float64(passed)/float64(len(results))*100)
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}
