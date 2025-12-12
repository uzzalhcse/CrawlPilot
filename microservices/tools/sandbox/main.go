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

// ==========================================
// Domain Models & State
// ==========================================

type Category struct {
	Slug string
	Name string
	Icon string
	Subs []SubCategory
}

type SubCategory struct {
	Slug string
	Name string
}

type Product struct {
	ID          string
	Name        string
	Price       float64
	OldPrice    float64
	Rating      float64
	Reviews     int
	Category    string
	SubCategory string
	Image       string
	Description string
	Features    []string
	InStock     bool
}

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
	AntiBotEnabled      bool    `json:"anti_bot_enabled"`
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
	AntiBotEnabled:      true, // Enabled by default
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

var categories = []Category{
	{Slug: "electronics", Name: "Electronics", Icon: "📱", Subs: []SubCategory{
		{Slug: "phones", Name: "Cell Phones"},
		{Slug: "laptops", Name: "Laptops"},
		{Slug: "tablets", Name: "Tablets"},
		{Slug: "headphones", Name: "Headphones"},
		{Slug: "cameras", Name: "Cameras"},
	}},
	{Slug: "computers", Name: "Computers", Icon: "💻", Subs: []SubCategory{
		{Slug: "desktops", Name: "Desktop Computers"},
		{Slug: "monitors", Name: "Monitors"},
		{Slug: "keyboards", Name: "Keyboards"},
		{Slug: "mice", Name: "Mice"},
		{Slug: "storage", Name: "Storage"},
	}},
	{Slug: "gaming", Name: "Gaming", Icon: "🎮", Subs: []SubCategory{
		{Slug: "consoles", Name: "Consoles"},
		{Slug: "games", Name: "Video Games"},
		{Slug: "accessories", Name: "Gaming Accessories"},
		{Slug: "chairs", Name: "Gaming Chairs"},
	}},
	{Slug: "fashion", Name: "Fashion", Icon: "👕", Subs: []SubCategory{
		{Slug: "mens", Name: "Men's Clothing"},
		{Slug: "womens", Name: "Women's Clothing"},
		{Slug: "shoes", Name: "Shoes"},
		{Slug: "watches", Name: "Watches"},
	}},
	{Slug: "home", Name: "Home & Kitchen", Icon: "🏠", Subs: []SubCategory{
		{Slug: "furniture", Name: "Furniture"},
		{Slug: "appliances", Name: "Appliances"},
		{Slug: "decor", Name: "Home Decor"},
		{Slug: "kitchen", Name: "Kitchen & Dining"},
	}},
	{Slug: "books", Name: "Books", Icon: "📚", Subs: []SubCategory{
		{Slug: "fiction", Name: "Fiction"},
		{Slug: "nonfiction", Name: "Non-Fiction"},
		{Slug: "textbooks", Name: "Textbooks"},
		{Slug: "ebooks", Name: "eBooks"},
	}},
}

var products = []Product{
	// Electronics - Phones (4)
	{ID: "ELEC001", Name: "iPhone 15 Pro Max 256GB", Price: 1199.00, OldPrice: 1299.00, Rating: 4.8, Reviews: 15234, Category: "electronics", SubCategory: "phones", Image: "https://images.unsplash.com/photo-1592750475338-74b7b21085ab?w=400", Description: "The most advanced iPhone ever with A17 Pro chip", Features: []string{"A17 Pro chip", "48MP camera", "Titanium design", "Action button"}, InStock: true},
	{ID: "ELEC002", Name: "Samsung Galaxy S24 Ultra", Price: 1299.99, OldPrice: 1399.99, Rating: 4.7, Reviews: 8921, Category: "electronics", SubCategory: "phones", Image: "https://images.unsplash.com/photo-1610945415295-d9bbf067e59c?w=400", Description: "Galaxy AI is here with the most powerful Galaxy yet", Features: []string{"Galaxy AI", "200MP camera", "S Pen included", "Titanium frame"}, InStock: true},
	{ID: "ELEC003", Name: "Google Pixel 8 Pro", Price: 999.00, OldPrice: 1099.00, Rating: 4.6, Reviews: 5432, Category: "electronics", SubCategory: "phones", Image: "https://images.unsplash.com/photo-1598327105666-5b89351aff97?w=400", Description: "The best of Google AI in a phone", Features: []string{"Tensor G3", "Magic Eraser", "7 years updates", "50MP camera"}, InStock: true},
	{ID: "ELEC004", Name: "OnePlus 12 5G", Price: 799.00, OldPrice: 899.00, Rating: 4.5, Reviews: 3211, Category: "electronics", SubCategory: "phones", Image: "https://images.unsplash.com/photo-1511707171634-5f897ff02aa9?w=400", Description: "Flagship killer with Snapdragon 8 Gen 3", Features: []string{"100W charging", "Hasselblad camera", "120Hz display", "5400mAh"}, InStock: true},
	// Electronics - Laptops (3)
	{ID: "ELEC005", Name: "MacBook Pro 14\" M3 Pro", Price: 1999.00, OldPrice: 2199.00, Rating: 4.9, Reviews: 5621, Category: "electronics", SubCategory: "laptops", Image: "https://images.unsplash.com/photo-1517336714731-489689fd1ca8?w=400", Description: "Supercharged by M3 Pro for demanding workflows", Features: []string{"M3 Pro chip", "18GB memory", "512GB SSD", "18hr battery"}, InStock: true},
	{ID: "ELEC006", Name: "Dell XPS 15 OLED", Price: 1799.00, OldPrice: 1999.00, Rating: 4.7, Reviews: 3421, Category: "electronics", SubCategory: "laptops", Image: "https://images.unsplash.com/photo-1593642702821-c8da6771f0c6?w=400", Description: "Stunning 3.5K OLED display in a compact design", Features: []string{"Intel Core i7", "32GB RAM", "1TB SSD", "OLED display"}, InStock: true},
	{ID: "ELEC007", Name: "ThinkPad X1 Carbon Gen 11", Price: 1649.00, OldPrice: 1849.00, Rating: 4.8, Reviews: 2156, Category: "electronics", SubCategory: "laptops", Image: "https://images.unsplash.com/photo-1496181133206-80ce9b88a853?w=400", Description: "Legendary business laptop with carbon fiber chassis", Features: []string{"Intel Core Ultra", "16GB RAM", "512GB SSD", "2.8K display"}, InStock: true},
	// Electronics - Tablets (2)
	{ID: "ELEC008", Name: "iPad Pro 12.9\" M2", Price: 1099.00, OldPrice: 1199.00, Rating: 4.9, Reviews: 8765, Category: "electronics", SubCategory: "tablets", Image: "https://images.unsplash.com/photo-1544244015-0df4b3ffc6b0?w=400", Description: "The ultimate iPad experience with M2 chip", Features: []string{"M2 chip", "Liquid Retina XDR", "Face ID", "USB-C"}, InStock: true},
	{ID: "ELEC009", Name: "Samsung Galaxy Tab S9 Ultra", Price: 1199.00, OldPrice: 1299.00, Rating: 4.7, Reviews: 4532, Category: "electronics", SubCategory: "tablets", Image: "https://images.unsplash.com/photo-1561154464-82e9adf32764?w=400", Description: "PC-level performance in a tablet", Features: []string{"14.6\" AMOLED", "Snapdragon 8 Gen 2", "S Pen included", "IP68"}, InStock: true},
	// Electronics - Headphones (2)
	{ID: "ELEC010", Name: "Sony WH-1000XM5 Headphones", Price: 348.00, OldPrice: 399.99, Rating: 4.6, Reviews: 12453, Category: "electronics", SubCategory: "headphones", Image: "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=400", Description: "Industry-leading noise cancellation", Features: []string{"30hr battery", "Multipoint connection", "Speak-to-chat", "Wearing detection"}, InStock: true},
	{ID: "ELEC011", Name: "Apple AirPods Pro 2", Price: 249.00, OldPrice: 279.00, Rating: 4.8, Reviews: 18234, Category: "electronics", SubCategory: "headphones", Image: "https://images.unsplash.com/photo-1588423771073-b8903fbb85b5?w=400", Description: "Adaptive Audio, now with USB-C", Features: []string{"H2 chip", "Active Noise Cancelling", "Spatial Audio", "USB-C"}, InStock: true},
	// Electronics - Cameras (2)
	{ID: "ELEC012", Name: "Canon EOS R6 Mark II", Price: 2499.00, OldPrice: 2699.00, Rating: 4.9, Reviews: 2341, Category: "electronics", SubCategory: "cameras", Image: "https://images.unsplash.com/photo-1516035069371-29a1b244cc32?w=400", Description: "Full-frame mirrorless for photo and video", Features: []string{"24.2MP sensor", "40fps shooting", "4K 60p video", "In-body stabilization"}, InStock: true},
	{ID: "ELEC013", Name: "Sony Alpha A7 IV", Price: 2498.00, OldPrice: 2798.00, Rating: 4.8, Reviews: 3456, Category: "electronics", SubCategory: "cameras", Image: "https://images.unsplash.com/photo-1502920917128-1aa500764cbd?w=400", Description: "The new basic beyond basic", Features: []string{"33MP full-frame", "Real-time Eye AF", "4K 60p", "10fps burst"}, InStock: true},
	// Computers - Desktops (2)
	{ID: "COMP001", Name: "Mac Studio M2 Ultra", Price: 3999.00, OldPrice: 4399.00, Rating: 4.9, Reviews: 1234, Category: "computers", SubCategory: "desktops", Image: "https://images.unsplash.com/photo-1527443224154-c4a3942d3acf?w=400", Description: "Ultimate power for pro workflows", Features: []string{"M2 Ultra", "64GB RAM", "1TB SSD", "6 Thunderbolt"}, InStock: true},
	{ID: "COMP002", Name: "HP Omen 45L Gaming Desktop", Price: 2299.00, OldPrice: 2599.00, Rating: 4.6, Reviews: 876, Category: "computers", SubCategory: "desktops", Image: "https://images.unsplash.com/photo-1587202372775-e229f172b9d7?w=400", Description: "Dominate with RTX 4080", Features: []string{"RTX 4080", "Intel i9", "64GB DDR5", "Cryo Chamber"}, InStock: true},
	// Computers - Monitors (2)
	{ID: "COMP003", Name: "LG UltraGear 27\" 4K Monitor", Price: 699.99, OldPrice: 799.99, Rating: 4.5, Reviews: 4532, Category: "computers", SubCategory: "monitors", Image: "https://images.unsplash.com/photo-1527443224154-c4a3942d3acf?w=400", Description: "4K gaming monitor with 144Hz refresh rate", Features: []string{"4K UHD", "144Hz", "1ms response", "G-Sync Compatible"}, InStock: true},
	{ID: "COMP004", Name: "Samsung Odyssey G9 49\"", Price: 1299.99, OldPrice: 1499.99, Rating: 4.7, Reviews: 2345, Category: "computers", SubCategory: "monitors", Image: "https://images.unsplash.com/photo-1547082299-de196ea013d6?w=400", Description: "Super ultrawide gaming immersion", Features: []string{"DQHD 5120x1440", "240Hz", "1ms", "1000R curve"}, InStock: true},
	// Computers - Keyboards (2)
	{ID: "COMP005", Name: "Logitech MX Keys Keyboard", Price: 119.99, OldPrice: 149.99, Rating: 4.6, Reviews: 8765, Category: "computers", SubCategory: "keyboards", Image: "https://images.unsplash.com/photo-1587829741301-dc798b83add3?w=400", Description: "Advanced wireless illuminated keyboard", Features: []string{"Backlit keys", "Multi-device", "USB-C charging", "Smart illumination"}, InStock: true},
	{ID: "COMP006", Name: "Keychron Q1 Pro Mechanical", Price: 199.00, OldPrice: 229.00, Rating: 4.8, Reviews: 3456, Category: "computers", SubCategory: "keyboards", Image: "https://images.unsplash.com/photo-1595225476474-87563907a212?w=400", Description: "Premium wireless mechanical keyboard", Features: []string{"Gateron switches", "QMK/VIA", "CNC aluminum", "Hot-swappable"}, InStock: true},
	// Computers - Mice (2)
	{ID: "COMP007", Name: "Logitech MX Master 3S", Price: 99.99, OldPrice: 119.99, Rating: 4.7, Reviews: 12543, Category: "computers", SubCategory: "mice", Image: "https://images.unsplash.com/photo-1527864550417-7fd91fc51a46?w=400", Description: "Advanced wireless mouse for creators", Features: []string{"8000 DPI", "Quiet clicks", "USB-C", "Multi-device"}, InStock: true},
	{ID: "COMP008", Name: "Razer DeathAdder V3 Pro", Price: 149.99, OldPrice: 179.99, Rating: 4.8, Reviews: 5678, Category: "computers", SubCategory: "mice", Image: "https://images.unsplash.com/photo-1615663245857-ac93bb7c39e7?w=400", Description: "Pro-level esports mouse", Features: []string{"30K DPI", "63g weight", "90hr battery", "Focus Pro sensor"}, InStock: true},
	// Computers - Storage (2)
	{ID: "COMP009", Name: "Samsung 990 Pro 2TB NVMe", Price: 179.99, OldPrice: 229.99, Rating: 4.9, Reviews: 6789, Category: "computers", SubCategory: "storage", Image: "https://images.unsplash.com/photo-1597872200969-2b65d56bd16b?w=400", Description: "PCIe 4.0 NVMe with 7,450 MB/s read", Features: []string{"7450MB/s read", "6900MB/s write", "600 TBW", "Heat spreader"}, InStock: true},
	{ID: "COMP010", Name: "WD Black SN850X 4TB", Price: 349.99, OldPrice: 399.99, Rating: 4.7, Reviews: 2345, Category: "computers", SubCategory: "storage", Image: "https://images.unsplash.com/photo-1601737487795-dab272f52420?w=400", Description: "Gaming SSD optimized for PS5", Features: []string{"7300MB/s", "Game Mode 2.0", "RGB heatsink", "PCIe Gen4"}, InStock: true},
	// Gaming - Consoles (3)
	{ID: "GAME001", Name: "PlayStation 5 Console", Price: 499.99, OldPrice: 549.99, Rating: 4.8, Reviews: 25678, Category: "gaming", SubCategory: "consoles", Image: "https://images.unsplash.com/photo-1606813907291-d86efa9b94db?w=400", Description: "Experience lightning-fast loading with an ultra-high speed SSD", Features: []string{"4K gaming", "Ray tracing", "3D Audio", "DualSense controller"}, InStock: false},
	{ID: "GAME002", Name: "Nintendo Switch OLED", Price: 349.99, OldPrice: 379.99, Rating: 4.9, Reviews: 18234, Category: "gaming", SubCategory: "consoles", Image: "https://images.unsplash.com/photo-1578303512597-81e6cc155b3e?w=400", Description: "7-inch OLED screen with vibrant colors", Features: []string{"OLED screen", "64GB storage", "Enhanced audio", "Wide adjustable stand"}, InStock: true},
	{ID: "GAME003", Name: "Xbox Series X", Price: 499.99, OldPrice: 549.99, Rating: 4.7, Reviews: 15432, Category: "gaming", SubCategory: "consoles", Image: "https://images.unsplash.com/photo-1621259182978-fbf93132d53d?w=400", Description: "The fastest, most powerful Xbox ever", Features: []string{"12 teraflops", "4K at 120fps", "1TB SSD", "Quick Resume"}, InStock: true},
	// Gaming - Games (2)
	{ID: "GAME004", Name: "The Legend of Zelda: TOTK", Price: 59.99, OldPrice: 69.99, Rating: 4.9, Reviews: 45678, Category: "gaming", SubCategory: "games", Image: "https://images.unsplash.com/photo-1550745165-9bc0b252726f?w=400", Description: "An epic adventure awaits in the vast land and skies of Hyrule", Features: []string{"Open world", "100+ hours", "Crafting system", "Sky islands"}, InStock: true},
	{ID: "GAME005", Name: "Elden Ring GOTY Edition", Price: 49.99, OldPrice: 69.99, Rating: 4.8, Reviews: 32145, Category: "gaming", SubCategory: "games", Image: "https://images.unsplash.com/photo-1493711662062-fa541f7f3d24?w=400", Description: "Rise, Tarnished. Includes Shadow of the Erdtree expansion", Features: []string{"FromSoftware", "Open world RPG", "Co-op multiplayer", "DLC included"}, InStock: true},
	// Gaming - Accessories (2)
	{ID: "GAME006", Name: "DualSense Edge Controller", Price: 199.99, OldPrice: 229.99, Rating: 4.6, Reviews: 3456, Category: "gaming", SubCategory: "accessories", Image: "https://images.unsplash.com/photo-1592840496694-26d035b52b48?w=400", Description: "Ultra-customizable pro controller for PS5", Features: []string{"Swappable sticks", "Back buttons", "Adjustable triggers", "Carrying case"}, InStock: true},
	{ID: "GAME007", Name: "Xbox Elite Controller Series 2", Price: 179.99, OldPrice: 199.99, Rating: 4.7, Reviews: 8765, Category: "gaming", SubCategory: "accessories", Image: "https://images.unsplash.com/photo-1600861194942-f883de0dfe96?w=400", Description: "Over 30 ways to play like a pro", Features: []string{"40hr battery", "4 paddles", "3 thumbstick sets", "Adjustable tension"}, InStock: true},
	// Gaming - Chairs (2)
	{ID: "GAME008", Name: "Secretlab Titan Evo 2022", Price: 449.00, OldPrice: 499.00, Rating: 4.8, Reviews: 9876, Category: "gaming", SubCategory: "chairs", Image: "https://images.unsplash.com/photo-1616626625495-c6b0e91c2f5c?w=400", Description: "Award-winning gaming chair with 4-way lumbar", Features: []string{"4D armrests", "Magnetic headrest", "Pebble seat", "Multi-tilt"}, InStock: true},
	{ID: "GAME009", Name: "Herman Miller x Logitech Embody", Price: 1695.00, OldPrice: 1895.00, Rating: 4.9, Reviews: 2143, Category: "gaming", SubCategory: "chairs", Image: "https://images.unsplash.com/photo-1541558869434-2840d308329a?w=400", Description: "Premium ergonomic gaming chair", Features: []string{"Flexible spine", "12yr warranty", "Breathable fabric", "PostureFit"}, InStock: true},
	// Fashion - Mens (2)
	{ID: "FASH001", Name: "Patagonia Better Sweater Jacket", Price: 139.00, OldPrice: 159.00, Rating: 4.7, Reviews: 5678, Category: "fashion", SubCategory: "mens", Image: "https://images.unsplash.com/photo-1591047139829-d91aecb6caea?w=400", Description: "Classic fleece jacket made from recycled materials", Features: []string{"Recycled polyester", "Zip closure", "Raglan sleeves", "Stand-up collar"}, InStock: true},
	{ID: "FASH002", Name: "Levi's 501 Original Jeans", Price: 69.50, OldPrice: 89.50, Rating: 4.6, Reviews: 12345, Category: "fashion", SubCategory: "mens", Image: "https://images.unsplash.com/photo-1542272604-787c3835535d?w=400", Description: "The original blue jean since 1873", Features: []string{"Straight leg", "Button fly", "100% cotton", "Signature arcuate stitch"}, InStock: true},
	// Fashion - Womens (2)
	{ID: "FASH003", Name: "Lululemon Align Leggings", Price: 98.00, OldPrice: 118.00, Rating: 4.9, Reviews: 23456, Category: "fashion", SubCategory: "womens", Image: "https://images.unsplash.com/photo-1506629082955-511b1aa562c8?w=400", Description: "Buttery soft leggings for yoga and beyond", Features: []string{"Nulu fabric", "High rise", "Naked sensation", "Hidden pocket"}, InStock: true},
	{ID: "FASH004", Name: "Everlane Cashmere Sweater", Price: 145.00, OldPrice: 175.00, Rating: 4.7, Reviews: 4567, Category: "fashion", SubCategory: "womens", Image: "https://images.unsplash.com/photo-1576566588028-4147f3842f27?w=400", Description: "Grade-A Mongolian cashmere crew neck", Features: []string{"100% cashmere", "Relaxed fit", "Ribbed trim", "Sustainable"}, InStock: true},
	// Fashion - Shoes (2)
	{ID: "FASH005", Name: "Nike Air Max 270", Price: 150.00, OldPrice: 180.00, Rating: 4.5, Reviews: 12345, Category: "fashion", SubCategory: "shoes", Image: "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=400", Description: "Lifestyle shoe with Max Air unit for all-day comfort", Features: []string{"Max Air unit", "Mesh upper", "Foam midsole", "Rubber outsole"}, InStock: true},
	{ID: "FASH006", Name: "Adidas Ultraboost 23", Price: 190.00, OldPrice: 220.00, Rating: 4.7, Reviews: 8765, Category: "fashion", SubCategory: "shoes", Image: "https://images.unsplash.com/photo-1560769629-975ec94e6a86?w=400", Description: "Incredible energy return with BOOST midsole", Features: []string{"Primeknit upper", "BOOST midsole", "Continental rubber", "Linear Energy Push"}, InStock: true},
	// Fashion - Watches (2)
	{ID: "FASH007", Name: "Apple Watch Series 9", Price: 399.00, OldPrice: 449.00, Rating: 4.8, Reviews: 9876, Category: "fashion", SubCategory: "watches", Image: "https://images.unsplash.com/photo-1434493789847-2f02dc6ca35d?w=400", Description: "Smarter. Brighter. Mightier.", Features: []string{"S9 chip", "Double tap", "Brighter display", "Carbon neutral"}, InStock: true},
	{ID: "FASH008", Name: "Garmin Fenix 7 Pro", Price: 799.99, OldPrice: 899.99, Rating: 4.8, Reviews: 3456, Category: "fashion", SubCategory: "watches", Image: "https://images.unsplash.com/photo-1508685096489-7aacd43bd3b1?w=400", Description: "Multisport GPS watch with solar charging", Features: []string{"Solar charging", "Topo maps", "32GB storage", "Titanium bezel"}, InStock: true},
	// Home - Furniture (2)
	{ID: "HOME001", Name: "West Elm Mid-Century Sofa", Price: 1499.00, OldPrice: 1799.00, Rating: 4.6, Reviews: 2345, Category: "home", SubCategory: "furniture", Image: "https://images.unsplash.com/photo-1555041469-a586c61ea9bc?w=400", Description: "Iconic mid-century modern design", Features: []string{"Solid wood legs", "Down-filled cushions", "Deco weave fabric", "86\" wide"}, InStock: true},
	{ID: "HOME002", Name: "IKEA MALM Bed Frame", Price: 249.00, OldPrice: 299.00, Rating: 4.4, Reviews: 15678, Category: "home", SubCategory: "furniture", Image: "https://images.unsplash.com/photo-1505693416388-ac5ce068fe85?w=400", Description: "Clean design that looks great on all sides", Features: []string{"Queen size", "4 storage drawers", "High headboard", "Adjustable sides"}, InStock: true},
	// Home - Appliances (2)
	{ID: "HOME003", Name: "Dyson V15 Detect Vacuum", Price: 749.99, OldPrice: 849.99, Rating: 4.7, Reviews: 5678, Category: "home", SubCategory: "appliances", Image: "https://images.unsplash.com/photo-1558317374-067fb5f30001?w=400", Description: "Reveals invisible dust with a laser", Features: []string{"Laser dust detection", "LCD screen", "60min runtime", "HEPA filtration"}, InStock: true},
	{ID: "HOME004", Name: "Vitamix A3500 Blender", Price: 649.95, OldPrice: 749.95, Rating: 4.8, Reviews: 4321, Category: "home", SubCategory: "appliances", Image: "https://images.unsplash.com/photo-1570222094114-d054a817e56b?w=400", Description: "Smart blender with touchscreen controls", Features: []string{"Self-cleaning", "5 presets", "Variable speed", "Stainless steel blades"}, InStock: true},
	// Home - Decor (2)
	{ID: "HOME005", Name: "Philips Hue Starter Kit", Price: 179.99, OldPrice: 199.99, Rating: 4.6, Reviews: 8765, Category: "home", SubCategory: "decor", Image: "https://images.unsplash.com/photo-1507473885765-e6ed057f782c?w=400", Description: "Smart lighting that transforms your home", Features: []string{"16M colors", "Voice control", "Bridge included", "4 bulbs"}, InStock: true},
	{ID: "HOME006", Name: "Anthropologie Moroccan Rug", Price: 498.00, OldPrice: 598.00, Rating: 4.5, Reviews: 1234, Category: "home", SubCategory: "decor", Image: "https://images.unsplash.com/photo-1531835551805-16d864c8d311?w=400", Description: "Handwoven bohemian area rug", Features: []string{"8x10 feet", "Wool blend", "Hand-tufted", "Fade resistant"}, InStock: true},
	// Home - Kitchen (2)
	{ID: "HOME007", Name: "Instant Pot Duo 7-in-1", Price: 89.99, OldPrice: 119.99, Rating: 4.6, Reviews: 45678, Category: "home", SubCategory: "kitchen", Image: "https://images.unsplash.com/photo-1585515320310-259814833e62?w=400", Description: "7-in-1 Electric Pressure Cooker", Features: []string{"7 appliances in 1", "6 quart capacity", "13 programs", "Dishwasher safe"}, InStock: true},
	{ID: "HOME008", Name: "KitchenAid Stand Mixer", Price: 379.99, OldPrice: 449.99, Rating: 4.9, Reviews: 23456, Category: "home", SubCategory: "kitchen", Image: "https://images.unsplash.com/photo-1594385208974-2e4c0f7c4e7b?w=400", Description: "Iconic stand mixer with 10-speed control", Features: []string{"5-quart bowl", "10 speeds", "59-point mixing", "Tilt-head design"}, InStock: true},
	// Books - Fiction (2)
	{ID: "BOOK001", Name: "Fourth Wing by Rebecca Yarros", Price: 24.99, OldPrice: 29.99, Rating: 4.7, Reviews: 34567, Category: "books", SubCategory: "fiction", Image: "https://images.unsplash.com/photo-1544716278-ca5e3f4abd8c?w=400", Description: "Enter the Basgiath War College in this dragon fantasy", Features: []string{"Hardcover", "512 pages", "#1 Bestseller", "Fantasy romance"}, InStock: true},
	{ID: "BOOK002", Name: "A Court of Thorns and Roses", Price: 16.99, OldPrice: 19.99, Rating: 4.8, Reviews: 56789, Category: "books", SubCategory: "fiction", Image: "https://images.unsplash.com/photo-1512820790803-83ca734da794?w=400", Description: "A sensual Beauty and the Beast retelling", Features: []string{"Paperback", "432 pages", "Series starter", "Adult fantasy"}, InStock: true},
	// Books - Non-Fiction (2)
	{ID: "BOOK003", Name: "Atomic Habits by James Clear", Price: 18.99, OldPrice: 24.99, Rating: 4.9, Reviews: 89012, Category: "books", SubCategory: "nonfiction", Image: "https://images.unsplash.com/photo-1589829085413-56de8ae18c73?w=400", Description: "Tiny changes, remarkable results", Features: []string{"Hardcover", "320 pages", "10M+ sold", "Self-improvement"}, InStock: true},
	{ID: "BOOK004", Name: "The Psychology of Money", Price: 16.99, OldPrice: 21.99, Rating: 4.8, Reviews: 45678, Category: "books", SubCategory: "nonfiction", Image: "https://images.unsplash.com/photo-1554774853-aae0a22c8aa4?w=400", Description: "Timeless lessons on wealth, greed, and happiness", Features: []string{"Paperback", "256 pages", "Finance", "Morgan Housel"}, InStock: true},
	// Books - Textbooks (2)
	{ID: "BOOK005", Name: "Introduction to Algorithms", Price: 89.99, OldPrice: 129.99, Rating: 4.7, Reviews: 12345, Category: "books", SubCategory: "textbooks", Image: "https://images.unsplash.com/photo-1456513080510-7bf3a84b82f8?w=400", Description: "The bible of computer science algorithms", Features: []string{"Hardcover", "1312 pages", "4th Edition", "CLRS authors"}, InStock: true},
	{ID: "BOOK006", Name: "Campbell Biology 12th Edition", Price: 139.99, OldPrice: 179.99, Rating: 4.6, Reviews: 6789, Category: "books", SubCategory: "textbooks", Image: "https://images.unsplash.com/photo-1532012197267-da84d127e765?w=400", Description: "Gold standard biology textbook", Features: []string{"Hardcover", "1488 pages", "Mastering Bio access", "Full color"}, InStock: true},
	// Books - eBooks (2)
	{ID: "BOOK007", Name: "Kindle Paperwhite 16GB", Price: 149.99, OldPrice: 179.99, Rating: 4.8, Reviews: 28901, Category: "books", SubCategory: "ebooks", Image: "https://images.unsplash.com/photo-1544716278-ca5e3f4abd8c?w=400", Description: "The best Kindle for reading, anywhere", Features: []string{"6.8\" display", "Adjustable warm light", "Waterproof", "Weeks of battery"}, InStock: true},
	{ID: "BOOK008", Name: "Kindle Scribe 64GB", Price: 339.99, OldPrice: 399.99, Rating: 4.7, Reviews: 5678, Category: "books", SubCategory: "ebooks", Image: "https://images.unsplash.com/photo-1507842217343-583bb7270b66?w=400", Description: "Read and write on a 10.2\" display", Features: []string{"10.2\" display", "Premium Pen included", "PDF annotation", "Weeks of battery"}, InStock: true},
}

func main() {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(cors.New())
	app.Use(wafMiddleware)

	// Pages
	app.Get("/", homePage)
	app.Get("/category/:cat", categoryPage)
	app.Get("/category/:cat/:sub", subCategoryPage)
	app.Get("/dp/:id", productPage)
	app.Get("/s", searchPage)

	// Error simulation
	app.Get("/simulate/captcha", func(c *fiber.Ctx) error { return c.Type("html").SendString(captchaPage()) })
	app.Get("/simulate/blocked", func(c *fiber.Ctx) error { return c.Status(403).Type("html").SendString(blockedPage(c.IP())) })
	app.Get("/simulate/rate-limit", func(c *fiber.Ctx) error { return c.Status(429).Type("html").SendString(rateLimitPage()) })
	app.Get("/simulate/auth", func(c *fiber.Ctx) error { return c.Status(401).Type("html").SendString(authPage()) })
	app.Get("/simulate/error-500", func(c *fiber.Ctx) error { return c.Status(500).Type("html").SendString(serverErrorPage()) })

	// API
	app.Get("/api/stats", getStats)
	app.Get("/api/config", getConfig)
	app.Post("/api/config", updateConfig)
	app.Post("/api/reset", resetStats)

	// Dashboard
	app.Get("/admin", dashboard)

	fmt.Println("🛒 Amazom E-Commerce Sandbox Running on http://localhost:8585")
	app.Listen(":8585")
}

// ==========================================
// WAF Middleware
// ==========================================

func wafMiddleware(c *fiber.Ctx) error {
	if strings.HasPrefix(c.Path(), "/api/") || strings.HasPrefix(c.Path(), "/simulate/") || c.Path() == "/admin" {
		return c.Next()
	}

	config.mu.Lock()
	defer config.mu.Unlock()

	// If anti-bot is disabled, allow all requests through
	if !config.AntiBotEnabled {
		logReq(c, 200, 0, "allowed_bypass")
		return c.Next()
	}

	ip := c.IP()
	stats, exists := config.IPStats[ip]
	if !exists {
		stats = &IPStats{IP: ip, WindowStart: time.Now(), UserAgents: make([]string, 0)}
		config.IPStats[ip] = stats
	}

	if stats.IsBlocked && time.Now().Before(stats.BlockExpires) {
		logReq(c, 403, stats.BotScore, "blocked")
		return c.Status(403).Type("html").SendString(blockedPage(ip))
	}
	stats.IsBlocked = false

	now := time.Now()
	stats.LastRequest = now
	stats.RequestCount++
	if now.Sub(stats.WindowStart) > time.Minute {
		stats.WindowCount = 0
		stats.WindowStart = now
	}
	stats.WindowCount++

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

	if c.Cookies("session-id") == "" {
		c.Cookie(&fiber.Cookie{Name: "session-id", Value: fmt.Sprintf("sess-%d", time.Now().Unix()), Expires: time.Now().Add(24 * time.Hour)})
	}

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
	config.RequestLog = append(config.RequestLog, RequestLogEntry{Time: time.Now(), Method: c.Method(), Path: c.Path(), IP: c.IP(), StatusCode: status, BotScore: score, Action: action})
	if len(config.RequestLog) > 100 {
		config.RequestLog = config.RequestLog[1:]
	}
}

// ==========================================
// Shared Components
// ==========================================

const tailwindHead = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body { font-family: 'Inter', sans-serif; }
        .dropdown:hover .dropdown-menu { display: block; }
        .dropdown-menu { display: none; }
    </style>
`

func navBar() string {
	var catItems strings.Builder
	for _, cat := range categories {
		var subItems strings.Builder
		for _, sub := range cat.Subs {
			subItems.WriteString(fmt.Sprintf(`<a href="/category/%s/%s" class="subcategory-link block px-4 py-2 text-gray-700 hover:bg-orange-50 hover:text-orange-600" data-category="%s" data-subcategory="%s">%s</a>`, cat.Slug, sub.Slug, cat.Slug, sub.Slug, sub.Name))
		}
		catItems.WriteString(fmt.Sprintf(`
            <div class="dropdown relative">
                <a href="/category/%s" class="category-link px-3 py-2 hover:text-orange-400 transition flex items-center gap-1" data-category="%s">
                    <span class="category-icon">%s</span>
                    <span class="category-name">%s</span>
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                </a>
                <div class="dropdown-menu absolute left-0 top-full bg-white shadow-xl rounded-lg py-2 min-w-[200px] z-50 border">
                    %s
                </div>
            </div>`, cat.Slug, cat.Slug, cat.Icon, cat.Name, subItems.String()))
	}

	return fmt.Sprintf(`
    <header class="bg-gray-900 text-white sticky top-0 z-40">
        <div class="container mx-auto px-4">
            <div class="flex items-center h-16 gap-6">
                <a href="/" class="text-2xl font-bold text-orange-400 flex-shrink-0">amazom</a>
                <div class="flex-1 max-w-2xl">
                    <form action="/s" method="GET" class="flex">
                        <input type="text" name="k" class="flex-1 px-4 py-2 rounded-l-lg text-gray-900" placeholder="Search products...">
                        <button class="bg-orange-400 hover:bg-orange-500 px-6 py-2 rounded-r-lg font-medium transition">🔍</button>
                    </form>
                </div>
                <div class="flex items-center gap-4 text-sm">
                    <a href="/simulate/auth" class="hover:text-orange-400">Sign In</a>
                    <a href="#" class="hover:text-orange-400">🛒 Cart</a>
                </div>
            </div>
        </div>
        <nav class="bg-gray-800 border-t border-gray-700">
            <div class="container mx-auto px-4">
                <div class="flex items-center gap-1 text-sm py-1">
                    %s
                </div>
            </div>
        </nav>
    </header>`, catItems.String())
}

func footer() string {
	return `
    <footer class="bg-gray-900 text-gray-400 mt-16">
        <div class="container mx-auto px-4 py-12">
            <div class="grid grid-cols-2 md:grid-cols-4 gap-8">
                <div>
                    <h4 class="font-semibold text-white mb-4">Get to Know Us</h4>
                    <ul class="space-y-2 text-sm"><li><a href="#" class="hover:text-white">About Us</a></li><li><a href="#" class="hover:text-white">Careers</a></li></ul>
                </div>
                <div>
                    <h4 class="font-semibold text-white mb-4">Make Money</h4>
                    <ul class="space-y-2 text-sm"><li><a href="#" class="hover:text-white">Sell products</a></li><li><a href="#" class="hover:text-white">Affiliate</a></li></ul>
                </div>
                <div>
                    <h4 class="font-semibold text-white mb-4">Payment</h4>
                    <ul class="space-y-2 text-sm"><li><a href="#" class="hover:text-white">Gift Cards</a></li><li><a href="#" class="hover:text-white">Reload Balance</a></li></ul>
                </div>
                <div>
                    <h4 class="font-semibold text-white mb-4">Help</h4>
                    <ul class="space-y-2 text-sm"><li><a href="#" class="hover:text-white">Your Account</a></li><li><a href="#" class="hover:text-white">Returns</a></li></ul>
                </div>
            </div>
            <div class="border-t border-gray-800 mt-8 pt-8 text-center text-sm">
                <p>© 2024 Amazom Sandbox - For Testing Purposes Only</p>
            </div>
        </div>
    </footer>`
}

func productCard(p Product) string {
	stockBadge := `<span class="product-stock text-green-600 text-sm">✓ In Stock</span>`
	if !p.InStock {
		stockBadge = `<span class="product-stock text-red-500 text-sm">Out of Stock</span>`
	}
	discount := ""
	if p.OldPrice > p.Price {
		pct := int((1 - p.Price/p.OldPrice) * 100)
		discount = fmt.Sprintf(`<span class="product-discount bg-red-600 text-white text-xs px-2 py-0.5 rounded">-%d%%</span>`, pct)
	}
	return fmt.Sprintf(`
        <a href="/dp/%s" class="product-card group bg-white rounded-xl shadow hover:shadow-xl transition-all duration-300 overflow-hidden flex flex-col" data-product-id="%s">
            <div class="relative aspect-square overflow-hidden bg-gray-100">
                <img src="%s" alt="%s" class="product-image w-full h-full object-cover group-hover:scale-105 transition-transform duration-300">
                <div class="absolute top-2 left-2">%s</div>
            </div>
            <div class="p-4 flex-1 flex flex-col">
                <h3 class="product-title font-medium text-gray-900 line-clamp-2 group-hover:text-orange-600 transition mb-2">%s</h3>
                <div class="flex items-center gap-1 mb-2">
                    <div class="product-rating flex text-orange-400 text-sm">%s</div>
                    <span class="product-reviews text-sm text-gray-500">(%d)</span>
                </div>
                <div class="mt-auto">
                    <div class="flex items-baseline gap-2">
                        <span class="product-price text-xl font-bold text-gray-900">$%.2f</span>
                        <span class="product-old-price text-sm text-gray-400 line-through">$%.2f</span>
                    </div>
                    <div class="mt-1">%s</div>
                </div>
            </div>
        </a>
    `, p.ID, p.ID, p.Image, p.Name, discount, p.Name, renderStars(p.Rating), p.Reviews, p.Price, p.OldPrice, stockBadge)
}

func renderStars(rating float64) string {
	full := int(rating)
	var stars strings.Builder
	for i := 0; i < full; i++ {
		stars.WriteString(`★`)
	}
	if rating-float64(full) >= 0.5 {
		stars.WriteString(`★`)
	}
	return stars.String()
}

// ==========================================
// Page Handlers
// ==========================================

func homePage(c *fiber.Ctx) error {
	// Featured products
	var featuredCards strings.Builder
	for _, p := range products[:6] {
		featuredCards.WriteString(productCard(p))
	}

	// Category boxes
	var catBoxes strings.Builder
	for _, cat := range categories {
		catBoxes.WriteString(fmt.Sprintf(`
            <a href="/category/%s" class="category-box bg-white rounded-xl p-6 shadow hover:shadow-lg transition group" data-category="%s">
                <div class="category-icon text-4xl mb-3">%s</div>
                <h3 class="category-name font-semibold text-gray-900 group-hover:text-orange-600">%s</h3>
                <p class="text-sm text-gray-500 mt-1">%d subcategories</p>
            </a>
        `, cat.Slug, cat.Slug, cat.Icon, cat.Name, len(cat.Subs)))
	}

	html := tailwindHead + `<title>Amazom - Shop Everything</title></head>
<body class="bg-gray-100 min-h-screen">` + navBar() + `
    <main>
        <!-- Hero -->
        <div class="bg-gradient-to-r from-gray-900 to-gray-800 text-white py-16">
            <div class="container mx-auto px-4 text-center">
                <h1 class="text-4xl md:text-5xl font-bold mb-4">Welcome to Amazom</h1>
                <p class="text-xl text-gray-300 mb-8">Your one-stop shop for everything you need</p>
                <a href="/s?k=" class="bg-orange-500 hover:bg-orange-600 px-8 py-3 rounded-full font-semibold transition inline-block">Shop Now</a>
            </div>
        </div>

        <div class="container mx-auto px-4 py-12">
            <!-- Categories -->
            <section class="mb-12">
                <h2 class="text-2xl font-bold text-gray-900 mb-6">Shop by Category</h2>
                <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">` + catBoxes.String() + `</div>
            </section>

            <!-- Featured Products -->
            <section>
                <h2 class="text-2xl font-bold text-gray-900 mb-6">Featured Products</h2>
                <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-6">` + featuredCards.String() + `</div>
            </section>
        </div>
    </main>` + footer() + `</body></html>`
	return c.Type("html").SendString(html)
}

func categoryPage(c *fiber.Ctx) error {
	catSlug := c.Params("cat")
	var cat *Category
	for i := range categories {
		if categories[i].Slug == catSlug {
			cat = &categories[i]
			break
		}
	}
	if cat == nil {
		return c.Status(404).Type("html").SendString(notFoundPage())
	}

	// Find products in this category
	var productCards strings.Builder
	count := 0
	for _, p := range products {
		if p.Category == catSlug {
			productCards.WriteString(productCard(p))
			count++
		}
	}

	// Subcategory links
	var subLinks strings.Builder
	for _, sub := range cat.Subs {
		subLinks.WriteString(fmt.Sprintf(`<a href="/category/%s/%s" class="bg-white px-4 py-2 rounded-full shadow hover:shadow-md transition text-sm">%s</a>`, catSlug, sub.Slug, sub.Name))
	}

	html := tailwindHead + fmt.Sprintf(`<title>%s - Amazom</title></head>
<body class="bg-gray-100 min-h-screen">`, cat.Name) + navBar() + fmt.Sprintf(`
    <main class="container mx-auto px-4 py-8">
        <nav class="text-sm text-gray-500 mb-4">
            <a href="/" class="hover:text-orange-600">Home</a> / <span class="text-gray-900">%s</span>
        </nav>
        <h1 class="text-3xl font-bold text-gray-900 mb-2">%s %s</h1>
        <p class="text-gray-500 mb-6">%d products</p>

        <div class="flex flex-wrap gap-2 mb-8">%s</div>

        <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">%s</div>
    </main>`, cat.Name, cat.Icon, cat.Name, count, subLinks.String(), productCards.String()) + footer() + `</body></html>`
	return c.Type("html").SendString(html)
}

func subCategoryPage(c *fiber.Ctx) error {
	catSlug := c.Params("cat")
	subSlug := c.Params("sub")

	var cat *Category
	var sub *SubCategory
	for i := range categories {
		if categories[i].Slug == catSlug {
			cat = &categories[i]
			for j := range cat.Subs {
				if cat.Subs[j].Slug == subSlug {
					sub = &cat.Subs[j]
					break
				}
			}
			break
		}
	}
	if cat == nil || sub == nil {
		return c.Status(404).Type("html").SendString(notFoundPage())
	}

	var productCards strings.Builder
	count := 0
	for _, p := range products {
		if p.Category == catSlug && p.SubCategory == subSlug {
			productCards.WriteString(productCard(p))
			count++
		}
	}

	html := tailwindHead + fmt.Sprintf(`<title>%s - %s - Amazom</title></head>
<body class="bg-gray-100 min-h-screen">`, sub.Name, cat.Name) + navBar() + fmt.Sprintf(`
    <main class="container mx-auto px-4 py-8">
        <nav class="text-sm text-gray-500 mb-4">
            <a href="/" class="hover:text-orange-600">Home</a> / 
            <a href="/category/%s" class="hover:text-orange-600">%s</a> / 
            <span class="text-gray-900">%s</span>
        </nav>
        <h1 class="text-3xl font-bold text-gray-900 mb-2">%s</h1>
        <p class="text-gray-500 mb-6">%d products in %s</p>

        <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">%s</div>
    </main>`, cat.Slug, cat.Name, sub.Name, sub.Name, count, cat.Name, productCards.String()) + footer() + `</body></html>`
	return c.Type("html").SendString(html)
}

func searchPage(c *fiber.Ctx) error {
	q := c.Query("k", "")
	var cards strings.Builder
	count := 0
	for _, p := range products {
		if q == "" || strings.Contains(strings.ToLower(p.Name), strings.ToLower(q)) || strings.Contains(strings.ToLower(p.Category), strings.ToLower(q)) {
			cards.WriteString(productCard(p))
			count++
		}
	}

	html := tailwindHead + fmt.Sprintf(`<title>Results for "%s" - Amazom</title></head>
<body class="bg-gray-100 min-h-screen">`, q) + navBar() + fmt.Sprintf(`
    <main class="container mx-auto px-4 py-8">
        <h1 class="text-xl font-semibold text-gray-900 mb-2">Results for "<span class="text-orange-600">%s</span>"</h1>
        <p class="text-sm text-gray-500 mb-6">%d results</p>
        <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">%s</div>
    </main>`, q, count, cards.String()) + footer() + `</body></html>`
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

	// Features list
	var features strings.Builder
	for _, f := range p.Features {
		features.WriteString(fmt.Sprintf(`<li class="flex items-start gap-2"><span class="text-green-500">✓</span>%s</li>`, f))
	}

	stockStatus := `<span class="text-green-600 font-semibold">In Stock</span>`
	buyBtn := `<button id="add-to-cart-button" class="w-full bg-yellow-400 hover:bg-yellow-500 text-gray-900 font-semibold py-3 rounded-full transition mb-2">Add to Cart</button>
               <button class="w-full bg-orange-500 hover:bg-orange-600 text-white font-semibold py-3 rounded-full transition">Buy Now</button>`
	if !p.InStock {
		stockStatus = `<span class="text-red-500 font-semibold">Currently Unavailable</span>`
		buyBtn = `<button disabled class="w-full bg-gray-300 text-gray-500 font-semibold py-3 rounded-full cursor-not-allowed">Out of Stock</button>`
	}

	html := tailwindHead + fmt.Sprintf(`<title>%s - Amazom</title></head>
<body class="bg-gray-50 min-h-screen">`, p.Name) + navBar() + fmt.Sprintf(`
    <main class="container mx-auto px-4 py-8">
        <nav class="text-sm text-gray-500 mb-6">
            <a href="/" class="hover:text-orange-600">Home</a> / 
            <a href="/category/%s" class="hover:text-orange-600">%s</a> / 
            <span class="text-gray-900 truncate">%s</span>
        </nav>

        <div class="bg-white rounded-2xl shadow-lg p-8">
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
                <!-- Image -->
                <div class="lg:col-span-1">
                    <div class="aspect-square bg-gray-100 rounded-xl overflow-hidden sticky top-24">
                        <img id="main-image" src="%s" alt="%s" class="w-full h-full object-cover">
                    </div>
                </div>

                <!-- Details -->
                <div class="lg:col-span-1">
                    <h1 id="productTitle" class="text-2xl font-bold text-gray-900 mb-4">%s</h1>
                    <div class="flex items-center gap-2 mb-4">
                        <span class="text-orange-400">%s</span>
                        <a href="#reviews" class="text-sm text-blue-600 hover:underline">%d ratings</a>
                    </div>
                    <hr class="my-4">
                    <div class="mb-4">
                        <span class="text-sm text-gray-500">Price:</span>
                        <div class="flex items-baseline gap-2">
                            <span id="priceValue" class="text-3xl font-bold text-red-600">$%.2f</span>
                            <span class="text-lg text-gray-400 line-through">$%.2f</span>
                            <span class="text-sm text-green-600">Save $%.2f</span>
                        </div>
                    </div>
                    <p id="product-description" class="text-gray-600 mb-6">%s</p>
                    <div class="mb-6">
                        <h3 class="font-semibold text-gray-900 mb-3">About this item</h3>
                        <ul id="feature-bullets" class="space-y-2 text-gray-600">%s</ul>
                    </div>
                </div>

                <!-- Buy Box -->
                <div class="lg:col-span-1">
                    <div class="bg-gray-50 rounded-xl p-6 border sticky top-24">
                        <div class="text-2xl font-bold text-gray-900 mb-2">$%.2f</div>
                        <div id="stockStatus" class="mb-4">%s</div>
                        <div class="space-y-2">%s</div>
                        <div class="mt-4 pt-4 border-t text-sm text-gray-500">
                            <div class="flex justify-between mb-1"><span>Ships from</span><span class="text-gray-900">Amazom</span></div>
                            <div class="flex justify-between"><span>Sold by</span><span class="text-gray-900">Amazom</span></div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </main>`, p.Category, p.Category, p.Name, p.Image, p.Name, p.Name, renderStars(p.Rating), p.Reviews, p.Price, p.OldPrice, p.OldPrice-p.Price, p.Description, features.String(), p.Price, stockStatus, buyBtn) + footer() + `</body></html>`
	return c.Type("html").SendString(html)
}

// ==========================================
// Error Pages
// ==========================================

func captchaPage() string {
	return tailwindHead + `<title>Robot Check</title></head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center p-4">
    <div class="bg-white rounded-2xl shadow-xl p-8 max-w-md w-full text-center">
        <div class="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-6">
            <span class="text-3xl">🤖</span>
        </div>
        <h1 class="text-2xl font-bold text-gray-900 mb-2">Enter the characters you see below</h1>
        <p class="text-gray-600 mb-6">Sorry, we just need to make sure you're not a robot.</p>
        <div class="bg-gray-100 rounded-lg p-4 mb-6">
            <div class="g-recaptcha" data-sitekey="test">
                <img src="https://via.placeholder.com/300x80/f3f4f6/374151?text=aX9kM2" alt="captcha" class="mx-auto rounded">
            </div>
        </div>
        <input type="text" class="w-full border border-gray-300 rounded-lg px-4 py-3 mb-4" placeholder="Type the characters">
        <button class="w-full bg-orange-500 hover:bg-orange-600 text-white font-semibold py-3 rounded-lg transition">Continue</button>
    </div>
</body></html>`
}

func blockedPage(ip string) string {
	return tailwindHead + fmt.Sprintf(`<title>Access Denied</title></head>
<body class="bg-gray-900 min-h-screen flex items-center justify-center p-4">
    <div class="bg-gray-800 rounded-2xl shadow-xl p-8 max-w-lg w-full text-center border border-red-500/30">
        <div class="w-20 h-20 bg-red-500/20 rounded-full flex items-center justify-center mx-auto mb-6">
            <span class="text-4xl">🚫</span>
        </div>
        <h1 class="text-2xl font-bold text-white mb-2">Access Denied</h1>
        <p class="text-gray-400 mb-6">Your IP address <span class="text-red-400 font-mono">%s</span> has been blocked.</p>
        <div class="bg-gray-900 rounded-lg p-4 text-left text-sm font-mono text-gray-400 mb-6">
            <p>Error: BOT_DETECTED</p>
            <p>Duration: %d minutes</p>
            <p>Ray ID: %d</p>
        </div>
    </div>
</body></html>`, ip, config.BlockDurationMins, time.Now().Unix())
}

func rateLimitPage() string {
	return tailwindHead + `<title>Too Many Requests</title></head>
<body class="bg-orange-50 min-h-screen flex items-center justify-center p-4">
    <div class="bg-white rounded-2xl shadow-xl p-8 max-w-md w-full text-center">
        <div class="text-6xl mb-4">⏰</div>
        <h1 class="text-2xl font-bold text-gray-900 mb-2">Slow Down!</h1>
        <p class="text-gray-600 mb-6">You're making requests too fast.</p>
        <div class="bg-orange-100 rounded-lg p-4 text-orange-800">Rate limit exceeded. Try again in 60 seconds.</div>
    </div>
</body></html>`
}

func authPage() string {
	return tailwindHead + `<title>Sign In</title></head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center p-4">
    <div class="bg-white rounded-2xl shadow-xl p-8 max-w-md w-full">
        <h1 class="text-2xl font-bold text-gray-900 mb-2 text-center">Sign In</h1>
        <p class="text-gray-600 mb-6 text-center">Please log in to continue</p>
        <form id="login" class="space-y-4">
            <input type="text" name="username" class="w-full border rounded-lg px-4 py-3" placeholder="Email or username">
            <input type="password" name="password" class="w-full border rounded-lg px-4 py-3" placeholder="Password">
            <button class="w-full bg-yellow-400 hover:bg-yellow-500 font-semibold py-3 rounded-lg">Sign In</button>
        </form>
    </div>
</body></html>`
}

func serverErrorPage() string {
	return tailwindHead + `<title>Server Error</title></head>
<body class="bg-gray-900 min-h-screen flex items-center justify-center p-4">
    <div class="text-center text-white">
        <h1 class="text-9xl font-bold text-gray-700">500</h1>
        <h2 class="text-2xl font-semibold mt-4">Internal Server Error</h2>
        <p class="text-gray-400 mt-2">Something went wrong. Please try again later.</p>
        <a href="/" class="inline-block mt-6 bg-orange-500 px-6 py-2 rounded-lg">Go Home</a>
    </div>
</body></html>`
}

func notFoundPage() string {
	return tailwindHead + `<title>Not Found</title></head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center p-4">
    <div class="text-center">
        <h1 class="text-9xl font-bold text-gray-300">404</h1>
        <h2 class="text-2xl font-semibold text-gray-900 mt-4">Page Not Found</h2>
        <a href="/" class="inline-block mt-6 bg-orange-500 text-white px-6 py-2 rounded-lg">Go Home</a>
    </div>
</body></html>`
}

// ==========================================
// Dashboard & API
// ==========================================

func dashboard(c *fiber.Ctx) error {
	html := tailwindHead + `<title>Admin Dashboard</title></head>
<body class="bg-gray-900 text-white min-h-screen">
    <div class="container mx-auto px-4 py-8">
        <div class="flex items-center justify-between mb-8">
            <div>
                <h1 class="text-3xl font-bold">🛡️ Anti-Bot Sandbox Dashboard</h1>
                <p class="text-gray-400 mt-1">Test Crawlify's error recovery system against realistic bot detection</p>
            </div>
            <a href="/" class="bg-orange-500 hover:bg-orange-600 px-4 py-2 rounded-lg text-sm font-medium">← Back to Store</a>
        </div>

        <!-- How It Works Section -->
        <div class="bg-gradient-to-r from-blue-900/50 to-purple-900/50 rounded-2xl p-6 border border-blue-500/30 mb-8">
            <h2 class="text-xl font-bold text-blue-400 mb-4">📖 How Anti-Bot Detection Works</h2>
            
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <!-- Flow Diagram -->
                <div class="bg-gray-800/50 rounded-xl p-4">
                    <h3 class="font-semibold text-white mb-3">Detection Flow</h3>
                    <div class="space-y-2 text-sm font-mono">
                        <div class="flex items-center gap-2">
                            <span class="bg-blue-500 text-white px-2 py-1 rounded">1</span>
                            <span class="text-gray-300">Request arrives → IP tracked</span>
                        </div>
                        <div class="flex items-center gap-2 pl-8">
                            <span class="text-gray-500">↓</span>
                        </div>
                        <div class="flex items-center gap-2">
                            <span class="bg-blue-500 text-white px-2 py-1 rounded">2</span>
                            <span class="text-gray-300">Check headers (UA, cookies, etc.)</span>
                        </div>
                        <div class="flex items-center gap-2 pl-8">
                            <span class="text-gray-500">↓</span>
                        </div>
                        <div class="flex items-center gap-2">
                            <span class="bg-yellow-500 text-black px-2 py-1 rounded">3</span>
                            <span class="text-gray-300">Calculate Bot Score (0-100)</span>
                        </div>
                        <div class="flex items-center gap-2 pl-8">
                            <span class="text-gray-500">↓</span>
                        </div>
                        <div class="flex items-center gap-2">
                            <span class="bg-green-500 text-black px-2 py-1 rounded">4a</span>
                            <span class="text-green-400">Score < 50 → Allow ✓</span>
                        </div>
                        <div class="flex items-center gap-2">
                            <span class="bg-yellow-500 text-black px-2 py-1 rounded">4b</span>
                            <span class="text-yellow-400">Score 50-79 → Captcha/Delay</span>
                        </div>
                        <div class="flex items-center gap-2">
                            <span class="bg-red-500 text-white px-2 py-1 rounded">4c</span>
                            <span class="text-red-400">Score ≥ 80 → Block 403 🚫</span>
                        </div>
                    </div>
                </div>

                <!-- Scoring Rules -->
                <div class="bg-gray-800/50 rounded-xl p-4">
                    <h3 class="font-semibold text-white mb-3">Scoring Rules</h3>
                    <table class="w-full text-sm">
                        <thead>
                            <tr class="text-gray-400 border-b border-gray-700">
                                <th class="text-left py-2">Behavior</th>
                                <th class="text-right">Penalty</th>
                            </tr>
                        </thead>
                        <tbody class="text-gray-300">
                            <tr class="border-b border-gray-700/50">
                                <td class="py-2">Missing/Bot User-Agent</td>
                                <td class="text-right text-red-400">+40</td>
                            </tr>
                            <tr class="border-b border-gray-700/50">
                                <td class="py-2">UA rotation (>5 different)</td>
                                <td class="text-right text-orange-400">+15</td>
                            </tr>
                            <tr class="border-b border-gray-700/50">
                                <td class="py-2">No session cookie (after 3 reqs)</td>
                                <td class="text-right text-yellow-400">+10</td>
                            </tr>
                            <tr class="border-b border-gray-700/50">
                                <td class="py-2">Rate limit exceeded</td>
                                <td class="text-right text-yellow-400">+10</td>
                            </tr>
                            <tr>
                                <td class="py-2 text-green-400">Good behavior decay</td>
                                <td class="text-right text-green-400">-1/min</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <!-- Examples -->
            <div class="mt-6 bg-gray-800/50 rounded-xl p-4">
                <h3 class="font-semibold text-white mb-3">🧪 Test Examples</h3>
                <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                    <div class="bg-green-500/10 border border-green-500/30 rounded-lg p-3">
                        <div class="font-semibold text-green-400 mb-2">✓ Good Scraper</div>
                        <ul class="text-gray-300 space-y-1">
                            <li>• Real browser UA</li>
                            <li>• Accepts cookies</li>
                            <li>• 1-2 req/second</li>
                            <li>• Random delays</li>
                        </ul>
                        <div class="mt-2 text-green-400 text-xs">Result: Score ~0-20, Always Allowed</div>
                    </div>
                    <div class="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-3">
                        <div class="font-semibold text-yellow-400 mb-2">⚠️ Suspicious Scraper</div>
                        <ul class="text-gray-300 space-y-1">
                            <li>• Rotating UAs</li>
                            <li>• Ignores cookies</li>
                            <li>• 10+ req/second</li>
                            <li>• No delays</li>
                        </ul>
                        <div class="mt-2 text-yellow-400 text-xs">Result: Score ~50-79, Captcha/Slow</div>
                    </div>
                    <div class="bg-red-500/10 border border-red-500/30 rounded-lg p-3">
                        <div class="font-semibold text-red-400 mb-2">🚫 Bad Bot</div>
                        <ul class="text-gray-300 space-y-1">
                            <li>• Empty/bot UA</li>
                            <li>• No cookies at all</li>
                            <li>• 100+ req/minute</li>
                            <li>• Hammering same page</li>
                        </ul>
                        <div class="mt-2 text-red-400 text-xs">Result: Score 80+, Blocked!</div>
                    </div>
                </div>
            </div>

            <!-- curl Examples -->
            <div class="mt-6 bg-gray-800/50 rounded-xl p-4">
                <h3 class="font-semibold text-white mb-3">💻 Test with curl</h3>
                <div class="space-y-3 font-mono text-xs">
                    <div>
                        <span class="text-gray-400"># Good request (low score):</span>
                        <pre class="bg-gray-900 p-2 rounded mt-1 text-green-400 overflow-x-auto">curl -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64)" -c cookies.txt -b cookies.txt http://localhost:8585/dp/ELEC001</pre>
                    </div>
                    <div>
                        <span class="text-gray-400"># Bad request (high score - will get blocked fast):</span>
                        <pre class="bg-gray-900 p-2 rounded mt-1 text-red-400 overflow-x-auto">for i in {1..100}; do curl -s http://localhost:8585/s?k=test > /dev/null; done</pre>
                    </div>
                    <div>
                        <span class="text-gray-400"># Watch score increase in real-time in this dashboard!</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- Site Stats Cards -->
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
            <div class="bg-gradient-to-br from-blue-600 to-blue-700 rounded-xl p-6 text-center">
                <div class="text-3xl font-bold text-white" id="stat-urls">-</div>
                <div class="text-blue-200 text-sm mt-1">Total URLs</div>
            </div>
            <div class="bg-gradient-to-br from-green-600 to-green-700 rounded-xl p-6 text-center">
                <div class="text-3xl font-bold text-white" id="stat-products">-</div>
                <div class="text-green-200 text-sm mt-1">Products</div>
            </div>
            <div class="bg-gradient-to-br from-purple-600 to-purple-700 rounded-xl p-6 text-center">
                <div class="text-3xl font-bold text-white" id="stat-categories">-</div>
                <div class="text-purple-200 text-sm mt-1">Categories</div>
            </div>
            <div class="bg-gradient-to-br from-orange-600 to-orange-700 rounded-xl p-6 text-center">
                <div class="text-3xl font-bold text-white" id="stat-subcategories">-</div>
                <div class="text-orange-200 text-sm mt-1">Subcategories</div>
            </div>
        </div>

        <!-- Config & Stats Grid -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
            <div class="bg-gray-800 rounded-xl p-6 border border-gray-700">
                <h2 class="text-lg font-semibold text-green-400 mb-4">⚙️ Bot Detection Config</h2>
                <div id="config-panel" class="space-y-3 text-sm"></div>
            </div>
            <div class="bg-gray-800 rounded-xl p-6 border border-gray-700">
                <h2 class="text-lg font-semibold text-purple-400 mb-4">🔍 Live IP Reputation</h2>
                <div class="overflow-x-auto"><table class="w-full text-sm"><thead><tr class="text-gray-400 border-b border-gray-700"><th class="text-left py-2">IP</th><th>Reqs</th><th>Score</th><th>Status</th></tr></thead><tbody id="ip-body"></tbody></table></div>
            </div>
        </div>

        <!-- Request Log -->
        <div class="bg-gray-800 rounded-xl p-6 border border-gray-700">
            <div class="flex justify-between items-center mb-4">
                <h2 class="text-lg font-semibold text-blue-400">📜 Live Request Log</h2>
                <button onclick="fetch('/api/reset',{method:'POST'}).then(()=>load())" class="bg-red-600 hover:bg-red-700 px-3 py-1 rounded text-sm">Reset All</button>
            </div>
            <div id="log-container" class="max-h-64 overflow-y-auto space-y-1 text-xs font-mono"></div>
        </div>
    </div>

    <script>
        async function load() {
            const [statsRes, cfgRes] = await Promise.all([fetch('/api/stats'), fetch('/api/config')]);
            const data = await statsRes.json();
            const cfg = await cfgRes.json();

            // Update site stats
            if (data.site_stats) {
                document.getElementById('stat-urls').textContent = data.site_stats.total_urls || 0;
                document.getElementById('stat-products').textContent = data.site_stats.total_products || 0;
                document.getElementById('stat-categories').textContent = data.site_stats.total_categories || 0;
                document.getElementById('stat-subcategories').textContent = data.site_stats.total_subcategories || 0;
            }

            document.getElementById('config-panel').innerHTML = ` + "`" + `
                <div class="flex justify-between items-center mb-4 p-3 rounded-lg ${cfg.anti_bot_enabled ? 'bg-green-500/20 border border-green-500/50' : 'bg-red-500/20 border border-red-500/50'}">
                    <span class="font-semibold">🛡️ Anti-Bot Detection</span>
                    <label class="relative inline-flex items-center cursor-pointer">
                        <input type="checkbox" ${cfg.anti_bot_enabled ? 'checked' : ''} onchange="updateCfg('anti_bot_enabled', this.checked)" class="sr-only peer">
                        <div class="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-green-500"></div>
                        <span class="ml-2 text-sm font-medium ${cfg.anti_bot_enabled ? 'text-green-400' : 'text-red-400'}">${cfg.anti_bot_enabled ? 'ENABLED' : 'DISABLED'}</span>
                    </label>
                </div>
                <div class="flex justify-between items-center"><span>Rate Limit (req/min)</span><input type="number" value="${cfg.rate_limit_threshold}" onchange="updateCfg('rate_limit_threshold',+this.value)" class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right"></div>
                <div class="flex justify-between items-center"><span>Captcha Threshold</span><input type="number" value="${cfg.bot_score_threshold}" onchange="updateCfg('bot_score_threshold',+this.value)" class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right text-yellow-400"></div>
                <div class="flex justify-between items-center"><span>Block Threshold</span><input type="number" value="${cfg.block_score_threshold}" onchange="updateCfg('block_score_threshold',+this.value)" class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right text-red-400"></div>
                <div class="flex justify-between items-center"><span>No Cookie Penalty</span><input type="number" value="${cfg.no_cookie_penalty}" onchange="updateCfg('no_cookie_penalty',+this.value)" class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right"></div>
                <div class="flex justify-between items-center"><span>Bot UA Penalty</span><input type="number" value="${cfg.bot_ua_penalty}" onchange="updateCfg('bot_ua_penalty',+this.value)" class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right"></div>
                <div class="flex justify-between items-center"><span>Block Duration (min)</span><input type="number" value="${cfg.block_duration_mins}" onchange="updateCfg('block_duration_mins',+this.value)" class="w-20 bg-gray-700 border border-gray-600 rounded px-2 py-1 text-right"></div>
            ` + "`" + `;

            const ips = Object.values(data.ip_stats || {});
            document.getElementById('ip-body').innerHTML = ips.map(ip => ` + "`" + `<tr class="border-b border-gray-700/50"><td class="py-2 font-mono">${ip.ip}</td><td class="text-center">${ip.request_count}</td><td class="text-center"><span class="px-2 py-0.5 rounded ${ip.bot_score >= 80 ? 'bg-red-500/20 text-red-400' : ip.bot_score >= 50 ? 'bg-yellow-500/20 text-yellow-400' : 'bg-green-500/20 text-green-400'}">${ip.bot_score.toFixed(0)}</span></td><td class="text-center">${ip.is_blocked ? '<span class="text-red-400">BLOCKED</span>' : '<span class="text-green-400">Active</span>'}</td></tr>` + "`" + `).join('') || '<tr><td colspan="4" class="text-center py-4 text-gray-500">No requests yet - try browsing the store!</td></tr>';

            const logs = (data.request_log || []).reverse();
            document.getElementById('log-container').innerHTML = logs.map(l => ` + "`" + `<div class="flex gap-2 p-2 rounded ${l.status_code >= 400 ? 'bg-red-500/10' : 'bg-green-500/10'}"><span class="text-gray-500">${new Date(l.time).toLocaleTimeString()}</span><span class="font-bold">${l.method}</span><span class="flex-1 truncate">${l.path}</span><span class="${l.status_code >= 400 ? 'text-red-400' : 'text-green-400'}">${l.status_code}</span><span class="text-gray-500">Score:${l.bot_score.toFixed(0)}</span><span class="px-2 bg-gray-700 rounded">${l.action}</span></div>` + "`" + `).join('') || '<div class="text-gray-500 text-center py-4">No logs yet</div>';
        }
        async function updateCfg(k, v) { const b = {}; b[k] = v; await fetch('/api/config', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(b)}); load(); }
        load(); setInterval(load, 1000);
    </script>
</body></html>`
	return c.Type("html").SendString(html)
}

func getStats(c *fiber.Ctx) error {
	config.mu.RLock()
	defer config.mu.RUnlock()

	// Calculate site structure counts
	totalCategories := len(categories)
	totalSubcategories := 0
	for _, cat := range categories {
		totalSubcategories += len(cat.Subs)
	}
	totalProducts := len(products)
	// Total URLs = 1 (home) + categories + subcategories + products
	totalURLs := 1 + totalCategories + totalSubcategories + totalProducts

	return c.JSON(fiber.Map{
		"ip_stats":    config.IPStats,
		"request_log": config.RequestLog,
		"site_stats": fiber.Map{
			"total_urls":          totalURLs,
			"total_products":      totalProducts,
			"total_categories":    totalCategories,
			"total_subcategories": totalSubcategories,
		},
	})
}

func getConfig(c *fiber.Ctx) error {
	config.mu.RLock()
	defer config.mu.RUnlock()
	return c.JSON(fiber.Map{
		"anti_bot_enabled":     config.AntiBotEnabled,
		"rate_limit_threshold": config.RateLimitThreshold, "bot_score_threshold": config.BotScoreThreshold,
		"block_score_threshold": config.BlockScoreThreshold, "no_cookie_penalty": config.NoCookiePenalty,
		"bot_ua_penalty": config.BotUAPenalty, "ua_rotation_penalty": config.UARotationPenalty,
		"block_duration_mins": config.BlockDurationMins,
	})
}

func updateConfig(c *fiber.Ctx) error {
	config.mu.Lock()
	defer config.mu.Unlock()
	var updates map[string]interface{}
	if err := json.Unmarshal(c.Body(), &updates); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}
	for k, v := range updates {
		switch k {
		case "anti_bot_enabled":
			config.AntiBotEnabled = v.(bool)
		case "rate_limit_threshold":
			config.RateLimitThreshold = int(v.(float64))
		case "bot_score_threshold":
			config.BotScoreThreshold = v.(float64)
		case "block_score_threshold":
			config.BlockScoreThreshold = v.(float64)
		case "no_cookie_penalty":
			config.NoCookiePenalty = v.(float64)
		case "bot_ua_penalty":
			config.BotUAPenalty = v.(float64)
		case "ua_rotation_penalty":
			config.UARotationPenalty = v.(float64)
		case "block_duration_mins":
			config.BlockDurationMins = int(v.(float64))
		}
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func resetStats(c *fiber.Ctx) error {
	config.mu.Lock()
	defer config.mu.Unlock()
	config.IPStats = make(map[string]*IPStats)
	config.RequestLog = make([]RequestLogEntry, 0)
	return c.JSON(fiber.Map{"status": "ok"})
}
