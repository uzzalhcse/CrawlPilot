package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/uzzalhcse/crawlify/microservices/shared/database"
	"github.com/uzzalhcse/crawlify/microservices/shared/models"
)

// postgresBrowserProfileRepo implements BrowserProfileRepository using PostgreSQL
type postgresBrowserProfileRepo struct {
	db *database.DB
}

// NewBrowserProfileRepository creates a new PostgreSQL browser profile repository
func NewBrowserProfileRepository(db *database.DB) BrowserProfileRepository {
	return &postgresBrowserProfileRepo{db: db}
}

func (r *postgresBrowserProfileRepo) Create(ctx context.Context, profile *models.BrowserProfile) error {
	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}

	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now
	profile.SetDefaults()

	if err := profile.Validate(); err != nil {
		return err
	}

	// Marshal JSON fields
	tagsJSON, _ := json.Marshal(profile.Tags)
	launchArgsJSON, _ := json.Marshal(profile.LaunchArgs)
	languagesJSON, _ := json.Marshal(profile.Languages)
	fontsJSON, _ := json.Marshal(profile.Fonts)

	query := `
		INSERT INTO browser_profiles (
			id, name, description, status, folder, tags,
			driver_type, browser_type, executable_path, cdp_endpoint, launch_args,
			user_agent, platform, screen_width, screen_height, timezone, locale, languages,
			webgl_vendor, webgl_renderer, canvas_noise, hardware_concurrency, device_memory, fonts,
			do_not_track, disable_webrtc,
			geolocation_latitude, geolocation_longitude, geolocation_accuracy,
			proxy_enabled, proxy_type, proxy_server, proxy_username, proxy_password,
			usage_count, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, $24,
			$25, $26,
			$27, $28, $29,
			$30, $31, $32, $33, $34,
			$35, $36, $37
		)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		profile.ID, profile.Name, profile.Description, profile.Status, profile.Folder, string(tagsJSON),
		profile.DriverType, profile.BrowserType, profile.ExecutablePath, profile.CDPEndpoint, string(launchArgsJSON),
		profile.UserAgent, profile.Platform, profile.ScreenWidth, profile.ScreenHeight, profile.Timezone, profile.Locale, string(languagesJSON),
		profile.WebGLVendor, profile.WebGLRenderer, profile.CanvasNoise, profile.HardwareConcurrency, profile.DeviceMemory, string(fontsJSON),
		profile.DoNotTrack, profile.DisableWebRTC,
		profile.GeolocationLatitude, profile.GeolocationLongitude, profile.GeolocationAccuracy,
		profile.ProxyEnabled, profile.ProxyType, profile.ProxyServer, profile.ProxyUsername, profile.ProxyPassword,
		profile.UsageCount, profile.CreatedAt, profile.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create browser profile: %w", err)
	}

	return nil
}

func (r *postgresBrowserProfileRepo) Get(ctx context.Context, id string) (*models.BrowserProfile, error) {
	query := `
		SELECT id, name, description, status, folder, tags,
			driver_type, browser_type, executable_path, cdp_endpoint, launch_args,
			user_agent, platform, screen_width, screen_height, timezone, locale, languages,
			webgl_vendor, webgl_renderer, canvas_noise, hardware_concurrency, device_memory, fonts,
			do_not_track, disable_webrtc,
			geolocation_latitude, geolocation_longitude, geolocation_accuracy,
			proxy_enabled, proxy_type, proxy_server, proxy_username, proxy_password,
			usage_count, last_used_at, created_at, updated_at
		FROM browser_profiles
		WHERE id = $1 AND deleted_at IS NULL
	`

	var profile models.BrowserProfile
	var tagsJSON, launchArgsJSON, languagesJSON, fontsJSON []byte
	var folder, description, executablePath, cdpEndpoint, timezone, locale, webglVendor, webglRenderer *string
	var proxyType, proxyServer, proxyUsername, proxyPassword *string

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&profile.ID, &profile.Name, &description, &profile.Status, &folder, &tagsJSON,
		&profile.DriverType, &profile.BrowserType, &executablePath, &cdpEndpoint, &launchArgsJSON,
		&profile.UserAgent, &profile.Platform, &profile.ScreenWidth, &profile.ScreenHeight, &timezone, &locale, &languagesJSON,
		&webglVendor, &webglRenderer, &profile.CanvasNoise, &profile.HardwareConcurrency, &profile.DeviceMemory, &fontsJSON,
		&profile.DoNotTrack, &profile.DisableWebRTC,
		&profile.GeolocationLatitude, &profile.GeolocationLongitude, &profile.GeolocationAccuracy,
		&profile.ProxyEnabled, &proxyType, &proxyServer, &proxyUsername, &proxyPassword,
		&profile.UsageCount, &profile.LastUsedAt, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("browser profile not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get browser profile: %w", err)
	}

	// Handle nullable fields
	if description != nil {
		profile.Description = *description
	}
	if folder != nil {
		profile.Folder = *folder
	}
	if executablePath != nil {
		profile.ExecutablePath = *executablePath
	}
	if cdpEndpoint != nil {
		profile.CDPEndpoint = *cdpEndpoint
	}
	if timezone != nil {
		profile.Timezone = *timezone
	}
	if locale != nil {
		profile.Locale = *locale
	}
	if webglVendor != nil {
		profile.WebGLVendor = *webglVendor
	}
	if webglRenderer != nil {
		profile.WebGLRenderer = *webglRenderer
	}
	if proxyType != nil {
		profile.ProxyType = *proxyType
	}
	if proxyServer != nil {
		profile.ProxyServer = *proxyServer
	}
	if proxyUsername != nil {
		profile.ProxyUsername = *proxyUsername
	}
	if proxyPassword != nil {
		profile.ProxyPassword = *proxyPassword
	}

	// Unmarshal JSON fields
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &profile.Tags)
	}
	if len(launchArgsJSON) > 0 {
		json.Unmarshal(launchArgsJSON, &profile.LaunchArgs)
	}
	if len(languagesJSON) > 0 {
		json.Unmarshal(languagesJSON, &profile.Languages)
	}
	if len(fontsJSON) > 0 {
		json.Unmarshal(fontsJSON, &profile.Fonts)
	}

	return &profile, nil
}

func (r *postgresBrowserProfileRepo) List(ctx context.Context, filters BrowserProfileFilters) ([]*models.BrowserProfile, error) {
	query := `
		SELECT id, name, description, status, folder, tags,
			driver_type, browser_type, executable_path, cdp_endpoint, launch_args,
			user_agent, platform, screen_width, screen_height, timezone, locale, languages,
			webgl_vendor, webgl_renderer, canvas_noise, hardware_concurrency, device_memory, fonts,
			do_not_track, disable_webrtc,
			geolocation_latitude, geolocation_longitude, geolocation_accuracy,
			proxy_enabled, proxy_type, proxy_server, proxy_username, proxy_password,
			usage_count, last_used_at, created_at, updated_at
		FROM browser_profiles
		WHERE deleted_at IS NULL
	`

	args := []interface{}{}
	argPos := 1

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filters.Status)
		argPos++
	}

	if filters.Folder != "" {
		query += fmt.Sprintf(" AND folder = $%d", argPos)
		args = append(args, filters.Folder)
		argPos++
	}

	if filters.DriverType != "" {
		query += fmt.Sprintf(" AND driver_type = $%d", argPos)
		args = append(args, filters.DriverType)
		argPos++
	}

	query += " ORDER BY updated_at DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filters.Limit)
		argPos++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filters.Offset)
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list browser profiles: %w", err)
	}
	defer rows.Close()

	profiles := make([]*models.BrowserProfile, 0)

	for rows.Next() {
		var profile models.BrowserProfile
		var tagsJSON, launchArgsJSON, languagesJSON, fontsJSON []byte
		var folder, description, executablePath, cdpEndpoint, timezone, locale, webglVendor, webglRenderer *string
		var proxyType, proxyServer, proxyUsername, proxyPassword *string

		err := rows.Scan(
			&profile.ID, &profile.Name, &description, &profile.Status, &folder, &tagsJSON,
			&profile.DriverType, &profile.BrowserType, &executablePath, &cdpEndpoint, &launchArgsJSON,
			&profile.UserAgent, &profile.Platform, &profile.ScreenWidth, &profile.ScreenHeight, &timezone, &locale, &languagesJSON,
			&webglVendor, &webglRenderer, &profile.CanvasNoise, &profile.HardwareConcurrency, &profile.DeviceMemory, &fontsJSON,
			&profile.DoNotTrack, &profile.DisableWebRTC,
			&profile.GeolocationLatitude, &profile.GeolocationLongitude, &profile.GeolocationAccuracy,
			&profile.ProxyEnabled, &proxyType, &proxyServer, &proxyUsername, &proxyPassword,
			&profile.UsageCount, &profile.LastUsedAt, &profile.CreatedAt, &profile.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan browser profile: %w", err)
		}

		// Handle nullable fields
		if description != nil {
			profile.Description = *description
		}
		if folder != nil {
			profile.Folder = *folder
		}
		if executablePath != nil {
			profile.ExecutablePath = *executablePath
		}
		if cdpEndpoint != nil {
			profile.CDPEndpoint = *cdpEndpoint
		}
		if timezone != nil {
			profile.Timezone = *timezone
		}
		if locale != nil {
			profile.Locale = *locale
		}
		if webglVendor != nil {
			profile.WebGLVendor = *webglVendor
		}
		if webglRenderer != nil {
			profile.WebGLRenderer = *webglRenderer
		}
		if proxyType != nil {
			profile.ProxyType = *proxyType
		}
		if proxyServer != nil {
			profile.ProxyServer = *proxyServer
		}
		if proxyUsername != nil {
			profile.ProxyUsername = *proxyUsername
		}
		if proxyPassword != nil {
			profile.ProxyPassword = *proxyPassword
		}

		// Unmarshal JSON fields
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &profile.Tags)
		}
		if len(launchArgsJSON) > 0 {
			json.Unmarshal(launchArgsJSON, &profile.LaunchArgs)
		}
		if len(languagesJSON) > 0 {
			json.Unmarshal(languagesJSON, &profile.Languages)
		}
		if len(fontsJSON) > 0 {
			json.Unmarshal(fontsJSON, &profile.Fonts)
		}

		profiles = append(profiles, &profile)
	}

	return profiles, nil
}

func (r *postgresBrowserProfileRepo) Update(ctx context.Context, profile *models.BrowserProfile) error {
	profile.UpdatedAt = time.Now()

	if err := profile.Validate(); err != nil {
		return err
	}

	// Marshal JSON fields
	tagsJSON, _ := json.Marshal(profile.Tags)
	launchArgsJSON, _ := json.Marshal(profile.LaunchArgs)
	languagesJSON, _ := json.Marshal(profile.Languages)
	fontsJSON, _ := json.Marshal(profile.Fonts)

	query := `
		UPDATE browser_profiles SET
			name = $2, description = $3, status = $4, folder = $5, tags = $6,
			driver_type = $7, browser_type = $8, executable_path = $9, cdp_endpoint = $10, launch_args = $11,
			user_agent = $12, platform = $13, screen_width = $14, screen_height = $15, timezone = $16, locale = $17, languages = $18,
			webgl_vendor = $19, webgl_renderer = $20, canvas_noise = $21, hardware_concurrency = $22, device_memory = $23, fonts = $24,
			do_not_track = $25, disable_webrtc = $26,
			geolocation_latitude = $27, geolocation_longitude = $28, geolocation_accuracy = $29,
			proxy_enabled = $30, proxy_type = $31, proxy_server = $32, proxy_username = $33, proxy_password = $34,
			updated_at = $35
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Pool.Exec(ctx, query,
		profile.ID, profile.Name, profile.Description, profile.Status, profile.Folder, string(tagsJSON),
		profile.DriverType, profile.BrowserType, profile.ExecutablePath, profile.CDPEndpoint, string(launchArgsJSON),
		profile.UserAgent, profile.Platform, profile.ScreenWidth, profile.ScreenHeight, profile.Timezone, profile.Locale, string(languagesJSON),
		profile.WebGLVendor, profile.WebGLRenderer, profile.CanvasNoise, profile.HardwareConcurrency, profile.DeviceMemory, string(fontsJSON),
		profile.DoNotTrack, profile.DisableWebRTC,
		profile.GeolocationLatitude, profile.GeolocationLongitude, profile.GeolocationAccuracy,
		profile.ProxyEnabled, profile.ProxyType, profile.ProxyServer, profile.ProxyUsername, profile.ProxyPassword,
		profile.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update browser profile: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("browser profile not found: %s", profile.ID)
	}

	return nil
}

func (r *postgresBrowserProfileRepo) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE browser_profiles
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete browser profile: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("browser profile not found: %s", id)
	}

	return nil
}

func (r *postgresBrowserProfileRepo) Duplicate(ctx context.Context, id string) (*models.BrowserProfile, error) {
	// Get existing profile
	original, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Create copy with new ID
	duplicate := *original
	duplicate.ID = uuid.New().String()
	duplicate.Name = original.Name + " (Copy)"
	duplicate.UsageCount = 0
	duplicate.LastUsedAt = nil
	duplicate.CreatedAt = time.Now()
	duplicate.UpdatedAt = time.Now()

	if err := r.Create(ctx, &duplicate); err != nil {
		return nil, err
	}

	return &duplicate, nil
}

func (r *postgresBrowserProfileRepo) UpdateUsage(ctx context.Context, id string) error {
	query := `
		UPDATE browser_profiles
		SET usage_count = usage_count + 1, last_used_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.Pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update usage: %w", err)
	}

	return nil
}

func (r *postgresBrowserProfileRepo) GetFolders(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT folder FROM browser_profiles
		WHERE folder IS NOT NULL AND folder != '' AND deleted_at IS NULL
		ORDER BY folder
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %w", err)
	}
	defer rows.Close()

	folders := make([]string, 0)
	for rows.Next() {
		var folder string
		if err := rows.Scan(&folder); err != nil {
			return nil, err
		}
		folders = append(folders, folder)
	}

	return folders, nil
}
