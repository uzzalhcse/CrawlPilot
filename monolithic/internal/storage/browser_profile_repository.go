package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/uzzalhcse/crawlify/pkg/models"
)

type BrowserProfileRepository struct {
	db *PostgresDB
}

func NewBrowserProfileRepository(db *PostgresDB) *BrowserProfileRepository {
	return &BrowserProfileRepository{db: db}
}

func (r *BrowserProfileRepository) Create(ctx context.Context, profile *models.BrowserProfile) error {
	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}

	query := `
		INSERT INTO browser_profiles (
			id, name, description, status, folder, tags,
			browser_type, executable_path, cdp_endpoint, launch_args,
			user_agent, platform, screen_width, screen_height, timezone, locale, languages,
			webgl_vendor, webgl_renderer, canvas_noise, hardware_concurrency, device_memory, fonts,
			do_not_track, disable_webrtc, geolocation_latitude, geolocation_longitude, geolocation_accuracy,
			proxy_enabled, proxy_type, proxy_server, proxy_username, proxy_password,
			cookies, local_storage, session_storage, indexed_db, clear_on_close,
			owner_id, shared_with, permissions,
			usage_count, last_used_at,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23,
			$24, $25, $26, $27, $28,
			$29, $30, $31, $32, $33,
			$34, $35, $36, $37, $38,
			$39, $40, $41,
			$42, $43,
			NOW(), NOW()
		)
		RETURNING created_at, updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		profile.ID, profile.Name, profile.Description, profile.Status, profile.Folder, profile.Tags,
		profile.BrowserType, profile.ExecutablePath, profile.CDPEndpoint, profile.LaunchArgs,
		profile.UserAgent, profile.Platform, profile.ScreenWidth, profile.ScreenHeight, profile.Timezone, profile.Locale, profile.Languages,
		profile.WebGLVendor, profile.WebGLRenderer, profile.CanvasNoise, profile.HardwareConcurrency, profile.DeviceMemory, profile.Fonts,
		profile.DoNotTrack, profile.DisableWebRTC, profile.GeoLatitude, profile.GeoLongitude, profile.GeoAccuracy,
		profile.ProxyEnabled, profile.ProxyType, profile.ProxyServer, profile.ProxyUsername, profile.ProxyPassword,
		profile.Cookies, profile.LocalStorage, profile.SessionStorage, profile.IndexedDB, profile.ClearOnClose,
		profile.OwnerID, profile.SharedWith, profile.Permissions,
		profile.UsageCount, profile.LastUsedAt,
	).Scan(&profile.CreatedAt, &profile.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create browser profile: %w", err)
	}

	return nil
}

func (r *BrowserProfileRepository) GetByID(ctx context.Context, id string) (*models.BrowserProfile, error) {
	query := `
		SELECT 
			id, name, description, status, folder, tags,
			browser_type, executable_path, cdp_endpoint, launch_args,
			user_agent, platform, screen_width, screen_height, timezone, locale, languages,
			webgl_vendor, webgl_renderer, canvas_noise, hardware_concurrency, device_memory, fonts,
			do_not_track, disable_webrtc, geolocation_latitude, geolocation_longitude, geolocation_accuracy,
			proxy_enabled, proxy_type, proxy_server, proxy_username, proxy_password,
			cookies, local_storage, session_storage, indexed_db, clear_on_close,
			owner_id, shared_with, permissions,
			usage_count, last_used_at,
			created_at, updated_at
		FROM browser_profiles
		WHERE id = $1
	`

	var profile models.BrowserProfile
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&profile.ID, &profile.Name, &profile.Description, &profile.Status, &profile.Folder, &profile.Tags,
		&profile.BrowserType, &profile.ExecutablePath, &profile.CDPEndpoint, &profile.LaunchArgs,
		&profile.UserAgent, &profile.Platform, &profile.ScreenWidth, &profile.ScreenHeight, &profile.Timezone, &profile.Locale, &profile.Languages,
		&profile.WebGLVendor, &profile.WebGLRenderer, &profile.CanvasNoise, &profile.HardwareConcurrency, &profile.DeviceMemory, &profile.Fonts,
		&profile.DoNotTrack, &profile.DisableWebRTC, &profile.GeoLatitude, &profile.GeoLongitude, &profile.GeoAccuracy,
		&profile.ProxyEnabled, &profile.ProxyType, &profile.ProxyServer, &profile.ProxyUsername, &profile.ProxyPassword,
		&profile.Cookies, &profile.LocalStorage, &profile.SessionStorage, &profile.IndexedDB, &profile.ClearOnClose,
		&profile.OwnerID, &profile.SharedWith, &profile.Permissions,
		&profile.UsageCount, &profile.LastUsedAt,
		&profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("browser profile not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get browser profile: %w", err)
	}

	return &profile, nil
}

func (r *BrowserProfileRepository) List(ctx context.Context, status, folder string, limit, offset int) ([]*models.BrowserProfile, error) {
	query := `
		SELECT 
			id, name, description, status, folder, tags,
			browser_type, executable_path, cdp_endpoint, launch_args,
			user_agent, platform, screen_width, screen_height, timezone, locale, languages,
			webgl_vendor, webgl_renderer, canvas_noise, hardware_concurrency, device_memory, fonts,
			do_not_track, disable_webrtc, geolocation_latitude, geolocation_longitude, geolocation_accuracy,
			proxy_enabled, proxy_type, proxy_server, proxy_username, proxy_password,
			cookies, local_storage, session_storage, indexed_db, clear_on_close,
			owner_id, shared_with, permissions,
			usage_count, last_used_at,
			created_at, updated_at
		FROM browser_profiles
	`
	args := []interface{}{}
	argPos := 1
	whereClauses := []string{}

	if status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, status)
		argPos++
	}

	if folder != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("folder = $%d", argPos))
		args = append(args, folder)
		argPos++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, limit)
		argPos++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, offset)
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list browser profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*models.BrowserProfile
	for rows.Next() {
		var profile models.BrowserProfile
		err := rows.Scan(
			&profile.ID, &profile.Name, &profile.Description, &profile.Status, &profile.Folder, &profile.Tags,
			&profile.BrowserType, &profile.ExecutablePath, &profile.CDPEndpoint, &profile.LaunchArgs,
			&profile.UserAgent, &profile.Platform, &profile.ScreenWidth, &profile.ScreenHeight, &profile.Timezone, &profile.Locale, &profile.Languages,
			&profile.WebGLVendor, &profile.WebGLRenderer, &profile.CanvasNoise, &profile.HardwareConcurrency, &profile.DeviceMemory, &profile.Fonts,
			&profile.DoNotTrack, &profile.DisableWebRTC, &profile.GeoLatitude, &profile.GeoLongitude, &profile.GeoAccuracy,
			&profile.ProxyEnabled, &profile.ProxyType, &profile.ProxyServer, &profile.ProxyUsername, &profile.ProxyPassword,
			&profile.Cookies, &profile.LocalStorage, &profile.SessionStorage, &profile.IndexedDB, &profile.ClearOnClose,
			&profile.OwnerID, &profile.SharedWith, &profile.Permissions,
			&profile.UsageCount, &profile.LastUsedAt,
			&profile.CreatedAt, &profile.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan browser profile: %w", err)
		}
		profiles = append(profiles, &profile)
	}

	return profiles, nil
}

func (r *BrowserProfileRepository) Update(ctx context.Context, profile *models.BrowserProfile) error {
	query := `
		UPDATE browser_profiles
		SET name = $2, description = $3, status = $4, folder = $5, tags = $6,
		    browser_type = $7, executable_path = $8, cdp_endpoint = $9, launch_args = $10,
		    user_agent = $11, platform = $12, screen_width = $13, screen_height = $14, timezone = $15, locale = $16, languages = $17,
		    webgl_vendor = $18, webgl_renderer = $19, canvas_noise = $20, hardware_concurrency = $21, device_memory = $22, fonts = $23,
		    do_not_track = $24, disable_webrtc = $25, geolocation_latitude = $26, geolocation_longitude = $27, geolocation_accuracy = $28,
		    proxy_enabled = $29, proxy_type = $30, proxy_server = $31, proxy_username = $32, proxy_password = $33,
		    cookies = $34, local_storage = $35, session_storage = $36, indexed_db = $37, clear_on_close = $38,
		    owner_id = $39, shared_with = $40, permissions = $41,
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query,
		profile.ID, profile.Name, profile.Description, profile.Status, profile.Folder, profile.Tags,
		profile.BrowserType, profile.ExecutablePath, profile.CDPEndpoint, profile.LaunchArgs,
		profile.UserAgent, profile.Platform, profile.ScreenWidth, profile.ScreenHeight, profile.Timezone, profile.Locale, profile.Languages,
		profile.WebGLVendor, profile.WebGLRenderer, profile.CanvasNoise, profile.HardwareConcurrency, profile.DeviceMemory, profile.Fonts,
		profile.DoNotTrack, profile.DisableWebRTC, profile.GeoLatitude, profile.GeoLongitude, profile.GeoAccuracy,
		profile.ProxyEnabled, profile.ProxyType, profile.ProxyServer, profile.ProxyUsername, profile.ProxyPassword,
		profile.Cookies, profile.LocalStorage, profile.SessionStorage, profile.IndexedDB, profile.ClearOnClose,
		profile.OwnerID, profile.SharedWith, profile.Permissions,
	)

	if err != nil {
		return fmt.Errorf("failed to update browser profile: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("browser profile not found: %s", profile.ID)
	}

	return nil
}

func (r *BrowserProfileRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM browser_profiles WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete browser profile: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("browser profile not found: %s", id)
	}

	return nil
}

func (r *BrowserProfileRepository) IncrementUsageCount(ctx context.Context, id string) error {
	query := `
		UPDATE browser_profiles
		SET usage_count = usage_count + 1, last_used_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment usage count: %w", err)
	}

	return nil
}

func (r *BrowserProfileRepository) GetFolders(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT folder FROM browser_profiles WHERE folder IS NOT NULL ORDER BY folder`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %w", err)
	}
	defer rows.Close()

	var folders []string
	for rows.Next() {
		var folder string
		if err := rows.Scan(&folder); err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}
		folders = append(folders, folder)
	}

	return folders, nil
}

func (r *BrowserProfileRepository) SearchByTags(ctx context.Context, tags []string, limit, offset int) ([]*models.BrowserProfile, error) {
	query := `
		SELECT 
			id, name, description, status, folder, tags,
			browser_type, executable_path, cdp_endpoint, launch_args,
			user_agent, platform, screen_width, screen_height, timezone, locale, languages,
			webgl_vendor, webgl_renderer, canvas_noise, hardware_concurrency, device_memory, fonts,
			do_not_track, disable_webrtc, geolocation_latitude, geolocation_longitude, geolocation_accuracy,
			proxy_enabled, proxy_type, proxy_server, proxy_username, proxy_password,
			cookies, local_storage, session_storage, indexed_db, clear_on_close,
			owner_id, shared_with, permissions,
			usage_count, last_used_at,
			created_at, updated_at
		FROM browser_profiles
		WHERE tags && $1
		ORDER BY created_at DESC
	`

	args := []interface{}{tags}
	argPos := 2

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, limit)
		argPos++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, offset)
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search profiles by tags: %w", err)
	}
	defer rows.Close()

	var profiles []*models.BrowserProfile
	for rows.Next() {
		var profile models.BrowserProfile
		err := rows.Scan(
			&profile.ID, &profile.Name, &profile.Description, &profile.Status, &profile.Folder, &profile.Tags,
			&profile.BrowserType, &profile.ExecutablePath, &profile.CDPEndpoint, &profile.LaunchArgs,
			&profile.UserAgent, &profile.Platform, &profile.ScreenWidth, &profile.ScreenHeight, &profile.Timezone, &profile.Locale, &profile.Languages,
			&profile.WebGLVendor, &profile.WebGLRenderer, &profile.CanvasNoise, &profile.HardwareConcurrency, &profile.DeviceMemory, &profile.Fonts,
			&profile.DoNotTrack, &profile.DisableWebRTC, &profile.GeoLatitude, &profile.GeoLongitude, &profile.GeoAccuracy,
			&profile.ProxyEnabled, &profile.ProxyType, &profile.ProxyServer, &profile.ProxyUsername, &profile.ProxyPassword,
			&profile.Cookies, &profile.LocalStorage, &profile.SessionStorage, &profile.IndexedDB, &profile.ClearOnClose,
			&profile.OwnerID, &profile.SharedWith, &profile.Permissions,
			&profile.UsageCount, &profile.LastUsedAt,
			&profile.CreatedAt, &profile.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan browser profile: %w", err)
		}
		profiles = append(profiles, &profile)
	}

	return profiles, nil
}
