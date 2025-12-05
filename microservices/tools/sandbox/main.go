package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

/*
 * Advanced Error Recovery Sandbox
 * Simulates realistic e-commerce defenses with beautiful Tailwind UI
 */

// ==========================================
// Domain Models & State
// ==========================================

type IPStats struct {
	IP           string    `json:"ip"`
	RequestCount int       `json:"request_count"`
	WindowCount  int       `json:"window_count"`
	WindowStart  time.Time `json:"window_start"`
	BotScore     float64   `json:"bot_score"`
	LastRequest  time.Time `json:"last_request"`
	UserAgents   []string  `json:"user_agents"`
	IsBlocked    bool      `json:"is_blocked"`
	BlockExpires time.Time `json:"block_expires"`
}

type RequestLogEntry struct {
	Time       time.Time `json:"time"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	IP         string    `json:"ip"`
	StatusCode int       `json:"status_code"`
	BotScore   float64   `json:"bot_score"`
	Action     string    `json:"action"`
}

type Config struct {
	mu                  sync.RWMutex
	RateLimitThreshold  int     `json:"rate_limit_threshold"`
	BotScoreThreshold   float64 `json:"bot_score_threshold"`
	BlockScoreThreshold float64 `json:"block_score_threshold"`
	NoCookiePenalty     float64 `json:"no_cookie_penalty"`
	BotUAPenalty        float64 `json:"bot_ua_penalty"`
	UARotationPenalty   float64 `json:"ua_rotation_penalty"`
	BlockDurationMins   int     `json:"block_duration_mins"`
	IPStats             map[string]*IPStats
	RequestLog          []RequestLogEntry
}

var config = &Config{
	RateLimitThreshold:  60,
	BotScoreThreshold:   50.0,
	BlockScoreThreshold: 80.0,
	NoCookiePenalty:     10.0,
	BotUAPenalty:        40.0,
	UARotationPenalty:   15.0,
	BlockDurationMins:   10,
	IPStats:             make(map[string]*IPStats),
	RequestLog:          make([]RequestLogEntry, 0),
}

type Product struct {
	ID       string
	Name     string
	Price    float64
	Rating   float64
	Reviews  int
	Category string
	Image    string
}

var products = []Product{
	{ID: "B08N5KWB9H", Name: "Sony WH-1000XM5 Wireless Noise Canceling Headphones", Price: 348.00, Rating: 4.6, Reviews: 12453, Category: "Electronics", Image: "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=300"},
	{ID: "B09G9F5P4J", Name: "Apple MacBook Pro 14-inch M3 Pro Chip", Price: 1999.00, Rating: 4.8, Reviews: 892, Category: "Computers", Image: "https://images.unsplash.com/photo-1517336714731-489689fd1ca8?w=300"},
	{ID: "B07W55DDFB", Name: "Logitech MX Master 3S Wireless Mouse", Price: 99.99, Rating: 4.7, Reviews: 4521, Category: "Accessories", Image: "https://images.unsplash.com/photo-1527864550417-7fd91fc51a46?w=300"},
	{ID: "B09V3HBW8Q", Name: "Samsung Galaxy S24 Ultra Smartphone", Price: 1299.99, Rating: 4.5, Reviews: 3201, Category: "Electronics", Image: "https://images.unsplash.com/photo-1511707171634-5f897ff02aa9?w=300"},
	{ID: "B08J5F3G1L", Name: "Kindle Paperwhite 6.8\" Display E-Reader", Price: 149.99, Rating: 4.8, Reviews: 28901, Category: "Electronics", Image: "https://images.unsplash.com/photo-1544716278-ca5e3f4abd8c?w=300"},
	{ID: "B09JQMJHXY", Name: "Nintendo Switch OLED Model", Price: 349.99, Rating: 4.9, Reviews: 15234, Category: "Gaming", Image: "https://images.unsplash.com/photo-1578303512597-81e6cc155b3e?w=300"},
}

func main() {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(cors.New())
	app.Use(wafMiddleware)

	// Pages
	app.Get("/", dashboard)
	app.Get("/s", searchPage)
	app.Get("/dp/:id", productPage)

	// Error simulation endpoints
	app.Get("/simulate/captcha", func(c *fiber.Ctx) error {
		return c.Type("html").SendString(captchaPage())
	})
	app.Get("/simulate/blocked", func(c *fiber.Ctx) error {
		return c.Status(403).Type("html").SendString(blockedPage(c.IP()))
	})
	app.Get("/simulate/rate-limit", func(c *fiber.Ctx) error {
		return c.Status(429).Type("html").SendString(rateLimitPage())
	})
	app.Get("/simulate/auth", func(c *fiber.Ctx) error {
		return c.Status(401).Type("html").SendString(authPage())
	})
	app.Get("/simulate/error-500", func(c *fiber.Ctx) error {
		return c.Status(500).Type("html").SendString(serverErrorPage())
	})

	// API
	app.Get("/api/stats", getStats)
	app.Get("/api/config", getConfig)
	app.Post("/api/config", updateConfig)
	app.Post("/api/reset", resetStats)

	fmt.Println("🛒 E-Commerce Sandbox Running on http://localhost:9999")
	app.Listen(":9999")
}

// ==========================================
// WAF Middleware
// ==========================================

func wafMiddleware(c *fiber.Ctx) error {
	if c.Path() == "/" || strings.HasPrefix(c.Path(), "/api/") || strings.HasPrefix(c.Path(), "/simulate/") {
		return c.Next()
	}

	config.mu.Lock()
	defer config.mu.Unlock()

	ip := c.IP()
	stats, exists := config.IPStats[ip]
	if !exists {
		stats = &IPStats{IP: ip, WindowStart: time.Now(), UserAgents: make([]string, 0)}
		config.IPStats[ip] = stats
	}

	// Check block
	if stats.IsBlocked && time.Now().Before(stats.BlockExpires) {
		logReq(c, 403, stats.BotScore, "blocked")
		return c.Status(403).Type("html").SendString(blockedPage(ip))
	}
	stats.IsBlocked = false

	// Update stats
	now := time.Now()
	stats.LastRequest = now
	stats.RequestCount++
	if now.Sub(stats.WindowStart) > time.Minute {
		stats.WindowCount = 0
		stats.WindowStart = now
	}
	stats.WindowCount++

	// Track UA
	ua := c.Get("User-Agent")
	found := false
	for _, u := range stats.UserAgents {
		if u == ua {
			found = true
			break
		}
	}
	if !found {
		stats.UserAgents = append(stats.UserAgents, ua)
		if len(stats.UserAgents) > 5 {
			stats.BotScore += config.UARotationPenalty
		}
	}

	// Calculate score
	score := stats.BotScore
	if stats.WindowCount > config.RateLimitThreshold {
		score += 10
	}
	if ua == "" || strings.Contains(strings.ToLower(ua), "bot") {
		score += config.BotUAPenalty
	}
	if c.Cookies("session-id") == "" && stats.RequestCount > 3 {
		score += config.NoCookiePenalty
	}
	if score > 100 {
		score = 100
	}
	stats.BotScore = score

	// Set session cookie
	if c.Cookies("session-id") == "" {
		c.Cookie(&fiber.Cookie{Name: "session-id", Value: fmt.Sprintf("sess-%d", time.Now().Unix()), Expires: time.Now().Add(24 * time.Hour)})
	}

	// Enforce
	if score >= config.BlockScoreThreshold {
		stats.IsBlocked = true
		stats.BlockExpires = time.Now().Add(time.Duration(config.BlockDurationMins) * time.Minute)
		logReq(c, 403, score, "blocked_score")
		return c.Status(403).Type("html").SendString(blockedPage(ip))
	}
	if score >= config.BotScoreThreshold {
		if rand.Float64() > 0.5 {
			logReq(c, 200, score, "captcha")
			return c.Type("html").SendString(captchaPage())
		}
		time.Sleep(time.Duration(rand.Intn(1000)+500) * time.Millisecond)
	}
	if stats.WindowCount > config.RateLimitThreshold {
		logReq(c, 429, score, "rate_limit")
		return c.Status(429).Type("html").SendString(rateLimitPage())
	}

	logReq(c, 200, score, "allowed")
	return c.Next()
}

func logReq(c *fiber.Ctx, status int, score float64, action string) {
	config.RequestLog = append(config.RequestLog, RequestLogEntry{
		Time: time.Now(), Method: c.Method(), Path: c.Path(), IP: c.IP(), StatusCode: status, BotScore: score, Action: action,
	})
	if len(config.RequestLog) > 100 {
		config.RequestLog = config.RequestLog[1:]
	}
}

// ==========================================
// Page Templates with Tailwind CSS
// ==========================================

const tailwindHead = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>body { font-family: 'Inter', sans-serif; }</style>
`

func dashboard(c *fiber.Ctx) error {
	html := tailwindHead + `
    <title>🛒 Sandbox Dashboard</title>
</head>
<body class="bg-gray-900 text-white min-h-screen">
    <div class="container mx-auto px-4 py-8">
        <div class="flex items-center justify-between mb-8">
            <div>
                <h1 class="text-3xl font-bold bg-gradient-to-r from-cyan-400 to-purple-500 bg-clip-text text-transparent">🛒 E-Commerce Sandbox</h1>
                <p class="text-gray-400 mt-1">Realistic anti-bot simulation for testing Crawlify recovery</p>
            </div>
            <button onclick="fetch('/api/reset', {method:'POST'}).then(()=>location.reload())" class="bg-red-600 hover:bg-red-700 px-4 py-2 rounded-lg font-medium transition">Reset All</button>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
            <div class="bg-gray-800 rounded-xl p-6 border border-gray-700">
                <h2 class="text-lg font-semibold text-cyan-400 mb-4">🎯 Test Endpoints</h2>
                <div class="space-y-3">
                    <a href="/s?k=laptop" target="_blank" class="flex items-center justify-between p-3 bg-gray-700/50 rounded-lg hover:bg-gray-700 transition">
                        <span>Search Page</span>
                        <span class="text-xs text-gray-400">/s?k=laptop</span>
                    </a>
                    <a href="/dp/B08N5KWB9H" target="_blank" class="flex items-center justify-between p-3 bg-gray-700/50 rounded-lg hover:bg-gray-700 transition">
                        <span>Product Page</span>
                        <span class="text-xs text-gray-400">/dp/B08N5KWB9H</span>
                    </a>
                </div>
            </div>
            <div class="bg-gray-800 rounded-xl p-6 border border-gray-700">
                <h2 class="text-lg font-semibold text-orange-400 mb-4">⚠️ Error Simulations</h2>
                <div class="space-y-3">
                    <a href="/simulate/captcha" target="_blank" class="flex items-center justify-between p-3 bg-gray-700/50 rounded-lg hover:bg-gray-700 transition">
                        <span>Captcha Page</span>
                        <span class="text-xs bg-yellow-500/20 text-yellow-400 px-2 py-0.5 rounded">200</span>
                    </a>
                    <a href="/simulate/blocked" target="_blank" class="flex items-center justify-between p-3 bg-gray-700/50 rounded-lg hover:bg-gray-700 transition">
                        <span>Blocked (403)</span>
                        <span class="text-xs bg-red-500/20 text-red-400 px-2 py-0.5 rounded">403</span>
                    </a>
                    <a href="/simulate/rate-limit" target="_blank" class="flex items-center justify-between p-3 bg-gray-700/50 rounded-lg hover:bg-gray-700 transition">
                        <span>Rate Limit (429)</span>
                        <span class="text-xs bg-red-500/20 text-red-400 px-2 py-0.5 rounded">429</span>
                    </a>
                    <a href="/simulate/auth" target="_blank" class="flex items-center justify-between p-3 bg-gray-700/50 rounded-lg hover:bg-gray-700 transition">
                        <span>Auth Required</span>
                        <span class="text-xs bg-orange-500/20 text-orange-400 px-2 py-0.5 rounded">401</span>
                    </a>
                    <a href="/simulate/error-500" target="_blank" class="flex items-center justify-between p-3 bg-gray-700/50 rounded-lg hover:bg-gray-700 transition">
                        <span>Server Error</span>
                        <span class="text-xs bg-red-500/20 text-red-400 px-2 py-0.5 rounded">500</span>
                    </a>
                </div>
            </div>
            <div class="bg-gray-800 rounded-xl p-6 border border-gray-700">
                <h2 class="text-lg font-semibold text-green-400 mb-4">⚙️ Bot Detection Rules</h2>
                <div id="config-panel" class="space-y-3 text-sm"></div>
            </div>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div class="bg-gray-800 rounded-xl p-6 border border-gray-700">
                <h2 class="text-lg font-semibold text-purple-400 mb-4">🔍 IP Reputation</h2>
                <div id="ip-stats" class="overflow-x-auto">
                    <table class="w-full text-sm">
                        <thead><tr class="text-gray-400 border-b border-gray-700"><th class="text-left py-2">IP</th><th>Requests</th><th>Score</th><th>Status</th></tr></thead>
                        <tbody id="ip-body"></tbody>
                    </table>
                </div>
            </div>
            <div class="bg-gray-800 rounded-xl p-6 border border-gray-700">
                <h2 class="text-lg font-semibold text-blue-400 mb-4">📜 Request Log</h2>
                <div id="log-container" class="max-h-96 overflow-y-auto space-y-1 text-xs font-mono"></div>
            </div>
        </div>
    </div>

    <script>
        async function load() {
            const res = await fetch('/api/stats');
            const data = await res.json();
            
            // IP Stats
            const ips = Object.values(data.ip_stats || {});
            document.getElementById('ip-body').innerHTML = ips.map(ip => ` + "`" + `
                <tr class="border-b border-gray-700/50">
                    <td class="py-2 font-mono">${ip.ip}</td>
                    <td class="text-center">${ip.request_count}</td>
                    <td class="text-center">
                        <span class="px-2 py-0.5 rounded ${ip.bot_score >= 80 ? 'bg-red-500/20 text-red-400' : ip.bot_score >= 50 ? 'bg-yellow-500/20 text-yellow-400' : 'bg-green-500/20 text-green-400'}">${ip.bot_score.toFixed(0)}</span>
                    </td>
                    <td class="text-center">${ip.is_blocked ? '<span class="text-red-400">BLOCKED</span>' : '<span class="text-green-400">Active</span>'}</td>
                </tr>
            ` + "`" + `).join('') || '<tr><td colspan="4" class="text-center py-4 text-gray-500">No requests yet</td></tr>';

            // Logs
            const logs = (data.request_log || []).reverse();
            document.getElementById('log-container').innerHTML = logs.map(l => ` + "`" + `
                <div class="flex items-center gap-2 p-2 rounded ${l.status_code >= 400 ? 'bg-red-500/10' : 'bg-green-500/10'}">
                    <span class="text-gray-500">${new Date(l.time).toLocaleTimeString()}</span>
                    <span class="font-semibold">${l.method}</span>
                    <span class="text-gray-300 truncate flex-1">${l.path}</span>
                    <span class="${l.status_code >= 400 ? 'text-red-400' : 'text-green-400'}">${l.status_code}</span>
                    <span class="text-gray-500">Score: ${l.bot_score.toFixed(0)}</span>
                    <span class="text-xs px-2 py-0.5 rounded bg-gray-700">${l.action}</span>
                </div>
            ` + "`" + `).join('') || '<div class="text-center py-4 text-gray-500">No requests logged</div>';
        }

        async function loadConfig() {
            const res = await fetch('/api/config');
            const cfg = await res.json();
            document.getElementById('config-panel').innerHTML = ` + "`" + `
                <div class="flex justify-between items-center">
                    <span class="text-gray-300">Rate Limit (req/min)</span>
                    <input type="number" value="${cfg.rate_limit_threshold}" onchange="updateConfig('rate_limit_threshold', parseInt(this.value))"
                        class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right text-white focus:ring-2 focus:ring-green-500">
                </div>
                <div class="flex justify-between items-center">
                    <span class="text-gray-300">Captcha Threshold</span>
                    <input type="number" value="${cfg.bot_score_threshold}" onchange="updateConfig('bot_score_threshold', parseFloat(this.value))"
                        class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right text-yellow-400 focus:ring-2 focus:ring-green-500">
                </div>
                <div class="flex justify-between items-center">
                    <span class="text-gray-300">Block Threshold</span>
                    <input type="number" value="${cfg.block_score_threshold}" onchange="updateConfig('block_score_threshold', parseFloat(this.value))"
                        class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right text-red-400 focus:ring-2 focus:ring-green-500">
                </div>
                <div class="flex justify-between items-center">
                    <span class="text-gray-300">No Cookie Penalty</span>
                    <input type="number" value="${cfg.no_cookie_penalty}" onchange="updateConfig('no_cookie_penalty', parseFloat(this.value))"
                        class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right text-gray-400 focus:ring-2 focus:ring-green-500">
                </div>
                <div class="flex justify-between items-center">
                    <span class="text-gray-300">Bot UA Penalty</span>
                    <input type="number" value="${cfg.bot_ua_penalty}" onchange="updateConfig('bot_ua_penalty', parseFloat(this.value))"
                        class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right text-gray-400 focus:ring-2 focus:ring-green-500">
                </div>
                <div class="flex justify-between items-center">
                    <span class="text-gray-300">UA Rotation Penalty</span>
                    <input type="number" value="${cfg.ua_rotation_penalty}" onchange="updateConfig('ua_rotation_penalty', parseFloat(this.value))"
                        class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right text-gray-400 focus:ring-2 focus:ring-green-500">
                </div>
                <div class="flex justify-between items-center">
                    <span class="text-gray-300">Block Duration (min)</span>
                    <input type="number" value="${cfg.block_duration_mins}" onchange="updateConfig('block_duration_mins', parseInt(this.value))"
                        class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right text-gray-400 focus:ring-2 focus:ring-green-500">
                </div>
            ` + "`" + `;
        }

        async function updateConfig(key, value) {
            const body = {};
            body[key] = value;
            await fetch('/api/config', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body) });
            loadConfig();
        }

        load();
        loadConfig();
        setInterval(load, 1000);
    </script>
</body>
</html>`
	return c.Type("html").SendString(html)
}

func searchPage(c *fiber.Ctx) error {
	q := c.Query("k", "")
	var cards strings.Builder
	for _, p := range products {
		if q == "" || strings.Contains(strings.ToLower(p.Name), strings.ToLower(q)) || strings.Contains(strings.ToLower(p.Category), strings.ToLower(q)) {
			cards.WriteString(fmt.Sprintf(`
                <a href="/dp/%s" class="group bg-white rounded-xl shadow-lg overflow-hidden hover:shadow-xl transition-all duration-300 hover:-translate-y-1">
                    <div class="aspect-square overflow-hidden bg-gray-100">
                        <img src="%s" alt="%s" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300">
                    </div>
                    <div class="p-4">
                        <h3 class="font-medium text-gray-900 line-clamp-2 group-hover:text-orange-600 transition">%s</h3>
                        <div class="flex items-center gap-1 mt-2">
                            <div class="flex text-orange-400">%s</div>
                            <span class="text-sm text-gray-500">(%d)</span>
                        </div>
                        <div class="mt-2">
                            <span class="text-2xl font-bold text-gray-900">$%.2f</span>
                        </div>
                        <div class="mt-2 text-sm text-green-600">✓ In Stock</div>
                    </div>
                </a>
            `, p.ID, p.Image, p.Name, p.Name, renderStars(p.Rating), p.Reviews, p.Price))
		}
	}

	html := tailwindHead + fmt.Sprintf(`
    <title>Search: %s - Amazom</title>
</head>
<body class="bg-gray-100 min-h-screen">
    <header class="bg-gray-900 text-white py-3">
        <div class="container mx-auto px-4 flex items-center gap-4">
            <a href="/" class="text-2xl font-bold text-orange-400">amazom</a>
            <div class="flex-1 max-w-2xl">
                <div class="flex">
                    <input type="text" value="%s" class="flex-1 px-4 py-2 rounded-l-lg text-gray-900" placeholder="Search...">
                    <button class="bg-orange-400 hover:bg-orange-500 px-6 py-2 rounded-r-lg font-medium transition">Search</button>
                </div>
            </div>
            <a href="#" class="hover:text-orange-400 transition">Cart (0)</a>
        </div>
    </header>
    <main class="container mx-auto px-4 py-8">
        <div class="mb-6">
            <h1 class="text-xl font-semibold text-gray-900">Results for "<span class="text-orange-600">%s</span>"</h1>
            <p class="text-sm text-gray-500">%d results</p>
        </div>
        <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">
            %s
        </div>
    </main>
</body>
</html>`, q, q, q, len(products), cards.String())

	return c.Type("html").SendString(html)
}

func productPage(c *fiber.Ctx) error {
	id := c.Params("id")
	var p *Product
	for i := range products {
		if products[i].ID == id {
			p = &products[i]
			break
		}
	}
	if p == nil {
		return c.Status(404).Type("html").SendString(notFoundPage())
	}

	html := tailwindHead + fmt.Sprintf(`
    <title>%s - Amazom</title>
</head>
<body class="bg-gray-50 min-h-screen">
    <header class="bg-gray-900 text-white py-3">
        <div class="container mx-auto px-4 flex items-center gap-4">
            <a href="/" class="text-2xl font-bold text-orange-400">amazom</a>
            <div class="flex-1 max-w-2xl">
                <input type="text" class="w-full px-4 py-2 rounded-lg text-gray-900" placeholder="Search...">
            </div>
        </div>
    </header>
    <main class="container mx-auto px-4 py-8">
        <div class="bg-white rounded-2xl shadow-lg p-8">
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-12">
                <div class="aspect-square bg-gray-100 rounded-xl overflow-hidden">
                    <img src="%s" alt="%s" class="w-full h-full object-cover">
                </div>
                <div>
                    <p class="text-sm text-gray-500 mb-2">%s</p>
                    <h1 id="productTitle" class="text-2xl font-bold text-gray-900 mb-4">%s</h1>
                    <div class="flex items-center gap-2 mb-4">
                        <div class="flex text-orange-400">%s</div>
                        <a href="#reviews" class="text-sm text-blue-600 hover:underline">%d ratings</a>
                    </div>
                    <hr class="my-4">
                    <div class="mb-6">
                        <span class="text-sm text-gray-500">Price:</span>
                        <div class="flex items-baseline gap-1">
                            <span class="text-3xl font-bold text-gray-900" id="priceValue">$%.2f</span>
                        </div>
                    </div>
                    <div class="space-y-3 mb-6">
                        <div class="flex items-center gap-2 text-green-600">
                            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"></path></svg>
                            <span id="stockStatus">In Stock</span>
                        </div>
                    </div>
                    <div class="flex gap-4">
                        <button id="add-to-cart-button" class="flex-1 bg-yellow-400 hover:bg-yellow-500 text-gray-900 font-semibold py-3 px-6 rounded-full transition">Add to Cart</button>
                        <button class="flex-1 bg-orange-500 hover:bg-orange-600 text-white font-semibold py-3 px-6 rounded-full transition">Buy Now</button>
                    </div>
                    <div class="mt-8">
                        <h3 class="font-semibold text-gray-900 mb-2">About this item</h3>
                        <ul id="feature-bullets" class="list-disc list-inside text-gray-600 space-y-1">
                            <li>Premium quality electronics</li>
                            <li>Latest generation technology</li>
                            <li>1 year manufacturer warranty</li>
                            <li>Free shipping eligible</li>
                        </ul>
                    </div>
                </div>
            </div>
        </div>
    </main>
</body>
</html>`, p.Name, p.Image, p.Name, p.Category, p.Name, renderStars(p.Rating), p.Reviews, p.Price)

	return c.Type("html").SendString(html)
}

func renderStars(rating float64) string {
	full := int(rating)
	var stars strings.Builder
	for i := 0; i < full; i++ {
		stars.WriteString(`<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20"><path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"></path></svg>`)
	}
	if rating-float64(full) >= 0.5 {
		stars.WriteString(`<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20"><path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"></path></svg>`)
	}
	return stars.String()
}

// ==========================================
// Error Pages
// ==========================================

func captchaPage() string {
	return tailwindHead + `
    <title>Robot Check</title>
</head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center p-4">
    <div class="bg-white rounded-2xl shadow-xl p-8 max-w-md w-full text-center">
        <div class="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-6">
            <svg class="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
        </div>
        <h1 class="text-2xl font-bold text-gray-900 mb-2">Enter the characters you see below</h1>
        <p class="text-gray-600 mb-6">Sorry, we just need to make sure you're not a robot.</p>
        <div class="bg-gray-100 rounded-lg p-4 mb-6">
            <div class="g-recaptcha" data-sitekey="test">
                <img src="https://via.placeholder.com/300x80/f3f4f6/374151?text=aX9kM2" alt="captcha" class="mx-auto rounded">
            </div>
        </div>
        <input type="text" class="w-full border border-gray-300 rounded-lg px-4 py-3 mb-4 focus:ring-2 focus:ring-orange-500 focus:border-transparent" placeholder="Type the characters">
        <button class="w-full bg-orange-500 hover:bg-orange-600 text-white font-semibold py-3 rounded-lg transition">Continue shopping</button>
    </div>
</body>
</html>`
}

func blockedPage(ip string) string {
	return tailwindHead + fmt.Sprintf(`
    <title>Access Denied</title>
</head>
<body class="bg-gray-900 min-h-screen flex items-center justify-center p-4">
    <div class="bg-gray-800 rounded-2xl shadow-xl p-8 max-w-lg w-full text-center border border-red-500/30">
        <div class="w-20 h-20 bg-red-500/20 rounded-full flex items-center justify-center mx-auto mb-6">
            <svg class="w-10 h-10 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"></path></svg>
        </div>
        <h1 class="text-2xl font-bold text-white mb-2">Access Denied</h1>
        <p class="text-gray-400 mb-6">Your IP address <span class="text-red-400 font-mono">%s</span> has been blocked due to suspicious activity.</p>
        <div class="bg-gray-900 rounded-lg p-4 text-left text-sm font-mono text-gray-400 mb-6">
            <p>Error: BOT_DETECTED</p>
            <p>Status: BLOCKED</p>
            <p>Duration: 10 minutes</p>
            <p>Ray ID: %d</p>
        </div>
        <p class="text-sm text-gray-500">If you believe this is an error, please wait and try again.</p>
    </div>
</body>
</html>`, ip, time.Now().Unix())
}

func rateLimitPage() string {
	return tailwindHead + `
    <title>Too Many Requests</title>
</head>
<body class="bg-orange-50 min-h-screen flex items-center justify-center p-4">
    <div class="bg-white rounded-2xl shadow-xl p-8 max-w-md w-full text-center">
        <div class="w-16 h-16 bg-orange-100 rounded-full flex items-center justify-center mx-auto mb-6">
            <svg class="w-8 h-8 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
        </div>
        <h1 class="text-2xl font-bold text-gray-900 mb-2">Slow Down!</h1>
        <p class="text-gray-600 mb-6">You're making requests too fast. Please wait a moment before trying again.</p>
        <div class="bg-orange-50 rounded-lg p-4 text-sm text-orange-800">
            <p class="font-semibold">Rate limit exceeded</p>
            <p>Please try again in 60 seconds</p>
        </div>
    </div>
</body>
</html>`
}

func authPage() string {
	return tailwindHead + `
    <title>Sign In</title>
</head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center p-4">
    <div class="bg-white rounded-2xl shadow-xl p-8 max-w-md w-full">
        <h1 class="text-2xl font-bold text-gray-900 mb-2 text-center">Sign In</h1>
        <p class="text-gray-600 mb-6 text-center">Please log in to continue</p>
        <form id="login" class="space-y-4">
            <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">Email or username</label>
                <input type="text" name="username" class="w-full border border-gray-300 rounded-lg px-4 py-3 focus:ring-2 focus:ring-orange-500">
            </div>
            <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">Password</label>
                <input type="password" name="password" class="w-full border border-gray-300 rounded-lg px-4 py-3 focus:ring-2 focus:ring-orange-500">
            </div>
            <button type="submit" class="w-full bg-yellow-400 hover:bg-yellow-500 font-semibold py-3 rounded-lg transition">Sign In</button>
        </form>
    </div>
</body>
</html>`
}

func serverErrorPage() string {
	return tailwindHead + `
    <title>Server Error</title>
</head>
<body class="bg-gray-900 min-h-screen flex items-center justify-center p-4">
    <div class="text-center">
        <h1 class="text-9xl font-bold text-gray-800">500</h1>
        <h2 class="text-2xl font-semibold text-white mt-4">Internal Server Error</h2>
        <p class="text-gray-400 mt-2">Something went wrong on our end. Please try again later.</p>
        <a href="/" class="inline-block mt-6 bg-orange-500 hover:bg-orange-600 text-white font-semibold py-2 px-6 rounded-lg transition">Go Home</a>
    </div>
</body>
</html>`
}

func notFoundPage() string {
	return tailwindHead + `
    <title>Page Not Found</title>
</head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center p-4">
    <div class="text-center">
        <h1 class="text-9xl font-bold text-gray-300">404</h1>
        <h2 class="text-2xl font-semibold text-gray-900 mt-4">Page Not Found</h2>
        <p class="text-gray-600 mt-2">The page you're looking for doesn't exist.</p>
        <a href="/" class="inline-block mt-6 bg-orange-500 hover:bg-orange-600 text-white font-semibold py-2 px-6 rounded-lg transition">Go Home</a>
    </div>
</body>
</html>`
}

// ==========================================
// API Endpoints
// ==========================================

func getStats(c *fiber.Ctx) error {
	config.mu.RLock()
	defer config.mu.RUnlock()
	return c.JSON(fiber.Map{"ip_stats": config.IPStats, "request_log": config.RequestLog})
}

func resetStats(c *fiber.Ctx) error {
	config.mu.Lock()
	defer config.mu.Unlock()
	config.IPStats = make(map[string]*IPStats)
	config.RequestLog = make([]RequestLogEntry, 0)
	return c.JSON(fiber.Map{"status": "ok"})
}

func getConfig(c *fiber.Ctx) error {
	config.mu.RLock()
	defer config.mu.RUnlock()
	return c.JSON(fiber.Map{
		"rate_limit_threshold":  config.RateLimitThreshold,
		"bot_score_threshold":   config.BotScoreThreshold,
		"block_score_threshold": config.BlockScoreThreshold,
		"no_cookie_penalty":     config.NoCookiePenalty,
		"bot_ua_penalty":        config.BotUAPenalty,
		"ua_rotation_penalty":   config.UARotationPenalty,
		"block_duration_mins":   config.BlockDurationMins,
	})
}

func updateConfig(c *fiber.Ctx) error {
	config.mu.Lock()
	defer config.mu.Unlock()

	var updates map[string]interface{}
	if err := json.Unmarshal(c.Body(), &updates); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	for key, value := range updates {
		switch key {
		case "rate_limit_threshold":
			config.RateLimitThreshold = int(value.(float64))
		case "bot_score_threshold":
			config.BotScoreThreshold = value.(float64)
		case "block_score_threshold":
			config.BlockScoreThreshold = value.(float64)
		case "no_cookie_penalty":
			config.NoCookiePenalty = value.(float64)
		case "bot_ua_penalty":
			config.BotUAPenalty = value.(float64)
		case "ua_rotation_penalty":
			config.UARotationPenalty = value.(float64)
		case "block_duration_mins":
			config.BlockDurationMins = int(value.(float64))
		}
	}

	return c.JSON(fiber.Map{"status": "ok"})
}
