package recovery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uzzalhcse/crawlify/microservices/shared/logger"
	"go.uber.org/zap"
)

// ProxyManager manages proxy rotation and health tracking
type ProxyManager struct {
	pool       *pgxpool.Pool
	proxies    []*Proxy
	currentIdx int
	mu         sync.RWMutex

	// Domain-specific proxy tracking
	domainProxies map[string][]string // domain -> working proxy IDs
}

// NewProxyManager creates a new proxy manager
func NewProxyManager(pool *pgxpool.Pool) (*ProxyManager, error) {
	pm := &ProxyManager{
		pool:          pool,
		proxies:       make([]*Proxy, 0),
		domainProxies: make(map[string][]string),
	}

	// Load proxies from database
	if err := pm.loadProxies(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load proxies: %w", err)
	}

	return pm, nil
}

// loadProxies loads all valid proxies from the database
func (pm *ProxyManager) loadProxies(ctx context.Context) error {
	query := `
		SELECT id, proxy_id, server, username, password, proxy_address, port,
		       valid, last_verified, country_code, city_name, asn_name, asn_number,
		       confidence_high, proxy_type, failure_count, success_count, 
		       last_used, is_healthy, created_at, updated_at
		FROM proxies
		WHERE valid = true AND is_healthy = true
		ORDER BY failure_count ASC, success_count DESC
	`

	rows, err := pm.pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query proxies: %w", err)
	}
	defer rows.Close()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.proxies = make([]*Proxy, 0)

	for rows.Next() {
		p := &Proxy{}
		err := rows.Scan(
			&p.ID, &p.ProxyID, &p.Server, &p.Username, &p.Password,
			&p.ProxyAddress, &p.Port, &p.Valid, &p.LastVerified,
			&p.CountryCode, &p.CityName, &p.ASNName, &p.ASNNumber,
			&p.ConfidenceHigh, &p.ProxyType, &p.FailureCount, &p.SuccessCount,
			&p.LastUsed, &p.IsHealthy, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			logger.Warn("Failed to scan proxy row", zap.Error(err))
			continue
		}
		pm.proxies = append(pm.proxies, p)
	}

	logger.Info("Loaded proxies from database", zap.Int("count", len(pm.proxies)))
	return nil
}

// GetNext returns the next available proxy using round-robin
func (pm *ProxyManager) GetNext(ctx context.Context) (*Proxy, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return nil, fmt.Errorf("no proxies available")
	}

	// Find next healthy proxy
	startIdx := pm.currentIdx
	for {
		proxy := pm.proxies[pm.currentIdx]
		pm.currentIdx = (pm.currentIdx + 1) % len(pm.proxies)

		if proxy.IsHealthy && proxy.Valid {
			// Update last used
			go pm.updateLastUsed(proxy.ID)
			return proxy, nil
		}

		// Checked all proxies
		if pm.currentIdx == startIdx {
			break
		}
	}

	// No healthy proxies, return first one anyway
	return pm.proxies[0], nil
}

// GetForDomain returns a proxy that works well for a specific domain
func (pm *ProxyManager) GetForDomain(ctx context.Context, domain string) (*Proxy, error) {
	pm.mu.RLock()
	workingIDs, hasSpecific := pm.domainProxies[domain]
	pm.mu.RUnlock()

	if hasSpecific && len(workingIDs) > 0 {
		// Return a known-working proxy for this domain
		for _, id := range workingIDs {
			if proxy := pm.getByID(id); proxy != nil && proxy.IsHealthy {
				go pm.updateLastUsed(proxy.ID)
				return proxy, nil
			}
		}
	}

	// Fall back to round-robin
	return pm.GetNext(ctx)
}

// getByID finds a proxy by ID
func (pm *ProxyManager) getByID(id string) *Proxy {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, p := range pm.proxies {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// RecordSuccess records a successful request with a proxy
func (pm *ProxyManager) RecordSuccess(ctx context.Context, proxyID, domain string) error {
	// Update database
	query := `
		UPDATE proxies 
		SET success_count = success_count + 1, 
		    failure_count = GREATEST(0, failure_count - 1),
		    is_healthy = true,
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := pm.pool.Exec(ctx, query, proxyID)
	if err != nil {
		return fmt.Errorf("failed to record success: %w", err)
	}

	// Track domain-specific working proxy
	pm.mu.Lock()
	if pm.domainProxies[domain] == nil {
		pm.domainProxies[domain] = make([]string, 0)
	}
	// Add to working list if not already present
	found := false
	for _, id := range pm.domainProxies[domain] {
		if id == proxyID {
			found = true
			break
		}
	}
	if !found {
		pm.domainProxies[domain] = append(pm.domainProxies[domain], proxyID)
	}
	pm.mu.Unlock()

	// Update in-memory state
	if proxy := pm.getByID(proxyID); proxy != nil {
		proxy.SuccessCount++
		proxy.IsHealthy = true
	}

	return nil
}

// RecordFailure records a failed request with a proxy
func (pm *ProxyManager) RecordFailure(ctx context.Context, proxyID, domain string, pattern ErrorPattern) error {
	// Calculate if proxy should be marked unhealthy
	// Block-related patterns are more serious
	healthyUpdate := "is_healthy"
	if pattern == PatternBlocked || pattern == PatternRateLimited {
		healthyUpdate = "CASE WHEN failure_count >= 5 THEN false ELSE is_healthy END"
	}

	query := fmt.Sprintf(`
		UPDATE proxies 
		SET failure_count = failure_count + 1,
		    is_healthy = %s,
		    updated_at = NOW()
		WHERE id = $1
	`, healthyUpdate)

	_, err := pm.pool.Exec(ctx, query, proxyID)
	if err != nil {
		return fmt.Errorf("failed to record failure: %w", err)
	}

	// Remove from domain-specific working list
	pm.mu.Lock()
	if working := pm.domainProxies[domain]; working != nil {
		for i, id := range working {
			if id == proxyID {
				pm.domainProxies[domain] = append(working[:i], working[i+1:]...)
				break
			}
		}
	}
	pm.mu.Unlock()

	// Update in-memory state
	if proxy := pm.getByID(proxyID); proxy != nil {
		proxy.FailureCount++
		if proxy.FailureCount >= 5 && (pattern == PatternBlocked || pattern == PatternRateLimited) {
			proxy.IsHealthy = false
		}
	}

	logger.Debug("Recorded proxy failure",
		zap.String("proxy_id", proxyID),
		zap.String("domain", domain),
		zap.String("pattern", string(pattern)),
	)

	return nil
}

// MarkUnhealthy marks a proxy as unhealthy
func (pm *ProxyManager) MarkUnhealthy(ctx context.Context, proxyID string) error {
	query := `UPDATE proxies SET is_healthy = false, updated_at = NOW() WHERE id = $1`
	_, err := pm.pool.Exec(ctx, query, proxyID)
	if err != nil {
		return fmt.Errorf("failed to mark unhealthy: %w", err)
	}

	if proxy := pm.getByID(proxyID); proxy != nil {
		proxy.IsHealthy = false
	}

	return nil
}

// updateLastUsed updates the last_used timestamp
func (pm *ProxyManager) updateLastUsed(proxyID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE proxies SET last_used = NOW() WHERE id = $1`
	pm.pool.Exec(ctx, query, proxyID)
}

// Refresh reloads proxies from the database
func (pm *ProxyManager) Refresh(ctx context.Context) error {
	return pm.loadProxies(ctx)
}

// Count returns the number of available proxies
func (pm *ProxyManager) Count() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.proxies)
}

// CountHealthy returns the number of healthy proxies
func (pm *ProxyManager) CountHealthy() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	count := 0
	for _, p := range pm.proxies {
		if p.IsHealthy {
			count++
		}
	}
	return count
}

// SeedFromJSON seeds proxies from a JSON file (for initial setup)
func (pm *ProxyManager) SeedFromJSON(ctx context.Context, jsonData []byte) error {
	var proxies []Proxy
	if err := json.Unmarshal(jsonData, &proxies); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	inserted := 0
	for _, p := range proxies {
		// Generate ID if not present
		if p.ID == "" {
			p.ID = generateID()
		}

		query := `
			INSERT INTO proxies (
				id, proxy_id, server, username, password, proxy_address, port,
				valid, last_verified, country_code, city_name, asn_name, asn_number,
				confidence_high, proxy_type, failure_count, success_count, 
				last_used, is_healthy, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7,
				$8, $9, $10, $11, $12, $13,
				$14, $15, 0, 0,
				NOW(), true, NOW(), NOW()
			)
			ON CONFLICT (proxy_id) DO UPDATE SET
				server = EXCLUDED.server,
				username = EXCLUDED.username,
				password = EXCLUDED.password,
				valid = EXCLUDED.valid,
				last_verified = EXCLUDED.last_verified,
				country_code = EXCLUDED.country_code,
				city_name = EXCLUDED.city_name,
				updated_at = NOW()
		`

		_, err := pm.pool.Exec(ctx, query,
			p.ID, p.ProxyID, p.Server, p.Username, p.Password,
			p.ProxyAddress, p.Port, p.Valid, p.LastVerified,
			p.CountryCode, p.CityName, p.ASNName, p.ASNNumber,
			p.ConfidenceHigh, p.ProxyType,
		)
		if err != nil {
			logger.Warn("Failed to insert proxy", zap.String("proxy_id", p.ProxyID), zap.Error(err))
			continue
		}
		inserted++
	}

	logger.Info("Seeded proxies from JSON", zap.Int("inserted", inserted), zap.Int("total", len(proxies)))

	// Reload proxies
	return pm.loadProxies(ctx)
}

// generateID generates a random ID
func generateID() string {
	bytes := make([]byte, 12)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
