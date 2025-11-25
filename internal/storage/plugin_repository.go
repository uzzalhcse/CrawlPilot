package storage

import (
	"context"
	"fmt"

	"github.com/uzzalhcse/crawlify/pkg/models"
)

// PluginRepository handles database operations for plugins
type PluginRepository struct {
	db *PostgresDB
}

// NewPluginRepository creates a new plugin repository
func NewPluginRepository(db *PostgresDB) *PluginRepository {
	return &PluginRepository{db: db}
}

// CreatePlugin creates a new plugin
func (r *PluginRepository) CreatePlugin(ctx context.Context, plugin *models.Plugin) error {
	query := `
		INSERT INTO plugins (
			id, name, slug, description, author_name, author_email,
			repository_url, documentation_url, phase_type, plugin_type,
			category, tags, is_verified
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		plugin.ID, plugin.Name, plugin.Slug, plugin.Description,
		plugin.AuthorName, plugin.AuthorEmail, plugin.RepositoryURL,
		plugin.DocumentationURL, plugin.PhaseType, plugin.PluginType,
		plugin.Category, plugin.Tags, plugin.IsVerified,
	)
	return err
}

// GetPluginByID retrieves a plugin by ID
func (r *PluginRepository) GetPluginByID(ctx context.Context, id string) (*models.Plugin, error) {
	var plugin models.Plugin
	query := `SELECT * FROM plugins WHERE id = $1`
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&plugin.ID, &plugin.Name, &plugin.Slug, &plugin.Description,
		&plugin.AuthorName, &plugin.AuthorEmail, &plugin.RepositoryURL,
		&plugin.DocumentationURL, &plugin.PhaseType, &plugin.PluginType,
		&plugin.Category, &plugin.Tags, &plugin.IsVerified,
		&plugin.TotalDownloads, &plugin.TotalInstalls, &plugin.AverageRating,
		&plugin.CreatedAt, &plugin.UpdatedAt,
	)
	return &plugin, err
}

// GetPluginBySlug retrieves a plugin by slug
func (r *PluginRepository) GetPluginBySlug(ctx context.Context, slug string) (*models.Plugin, error) {
	var plugin models.Plugin
	query := `SELECT * FROM plugins WHERE slug = $1`
	err := r.db.Pool.QueryRow(ctx, query, slug).Scan(
		&plugin.ID, &plugin.Name, &plugin.Slug, &plugin.Description,
		&plugin.AuthorName, &plugin.AuthorEmail, &plugin.RepositoryURL,
		&plugin.DocumentationURL, &plugin.PhaseType, &plugin.PluginType,
		&plugin.Category, &plugin.Tags, &plugin.IsVerified,
		&plugin.TotalDownloads, &plugin.TotalInstalls, &plugin.AverageRating,
		&plugin.CreatedAt, &plugin.UpdatedAt,
	)
	return &plugin, err
}

// ListPlugins lists plugins with comprehensive filtering
func (r *PluginRepository) ListPlugins(ctx context.Context, filters models.PluginFilters) ([]*models.Plugin, error) {
	// Build dynamic query with filters
	query := `
		SELECT 
			id, name, slug, description, author_name, author_email,
			repository_url, documentation_url, phase_type, plugin_type,
			category, tags, is_verified, total_downloads, total_installs,
			average_rating, created_at, updated_at
		FROM plugins 
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	// Filter by phase type
	if filters.PhaseType != "" {
		query += ` AND phase_type = $` + fmt.Sprintf("%d", argCount)
		args = append(args, filters.PhaseType)
		argCount++
	}

	// Filter by category
	if filters.Category != "" {
		query += ` AND category = $` + fmt.Sprintf("%d", argCount)
		args = append(args, filters.Category)
		argCount++
	}

	// Filter by tags (array contains)
	if len(filters.Tags) > 0 {
		query += ` AND tags && $` + fmt.Sprintf("%d", argCount)
		args = append(args, filters.Tags)
		argCount++
	}

	// Filter by verified status
	if filters.VerifiedOnly {
		query += ` AND is_verified = true`
	}

	// Search query (name or description)
	if filters.SearchQuery != "" {
		query += ` AND (name ILIKE $` + fmt.Sprintf("%d", argCount) + ` OR description ILIKE $` + fmt.Sprintf("%d", argCount) + `)`
		searchPattern := "%" + filters.SearchQuery + "%"
		args = append(args, searchPattern)
		argCount++
	}

	// Sorting
	sortBy := "total_downloads DESC" // Default sort
	if filters.SortBy != "" {
		switch filters.SortBy {
		case "name":
			sortBy = "name ASC"
		case "created_at":
			sortBy = "created_at DESC"
		case "rating":
			sortBy = "average_rating DESC"
		case "installs":
			sortBy = "total_installs DESC"
		}
	}
	query += ` ORDER BY ` + sortBy

	// Pagination
	limit := 50
	if filters.Limit > 0 && filters.Limit <= 100 {
		limit = filters.Limit
	}
	query += ` LIMIT $` + fmt.Sprintf("%d", argCount)
	args = append(args, limit)
	argCount++

	if filters.Offset > 0 {
		query += ` OFFSET $` + fmt.Sprintf("%d", argCount)
		args = append(args, filters.Offset)
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plugins []*models.Plugin
	for rows.Next() {
		var p models.Plugin
		err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description,
			&p.AuthorName, &p.AuthorEmail, &p.RepositoryURL,
			&p.DocumentationURL, &p.PhaseType, &p.PluginType,
			&p.Category, &p.Tags, &p.IsVerified,
			&p.TotalDownloads, &p.TotalInstalls, &p.AverageRating,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, &p)
	}

	return plugins, rows.Err()
}

// UpdatePlugin updates a plugin
func (r *PluginRepository) UpdatePlugin(ctx context.Context, plugin *models.Plugin) error {
	query := `
		UPDATE plugins SET
			name = $1, description = $2, category = $3, tags = $4
		WHERE id = $5
	`
	_, err := r.db.Pool.Exec(ctx, query,
		plugin.Name, plugin.Description, plugin.Category,
		plugin.Tags, plugin.ID,
	)
	return err
}

// DeletePlugin deletes a plugin
func (r *PluginRepository) DeletePlugin(ctx context.Context, id string) error {
	query := `DELETE FROM plugins WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

// PublishVersion publishes a new plugin version
func (r *PluginRepository) PublishVersion(ctx context.Context, version *models.PluginVersion) error {
	query := `
		INSERT INTO plugin_versions (
			id, plugin_id, version, changelog, is_stable,
			linux_amd64_binary_path, config_schema
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		version.ID, version.PluginID, version.Version,
		version.Changelog, version.IsStable,
		version.LinuxAmd64BinaryPath, version.ConfigSchema,
	)
	return err
}

// GetLatestVersion retrieves the latest stable version
func (r *PluginRepository) GetLatestVersion(ctx context.Context, pluginID string) (*models.PluginVersion, error) {
	var version models.PluginVersion
	query := `
		SELECT 
			id, plugin_id, version,
			COALESCE(changelog, '') as changelog,
			is_stable,
			COALESCE(min_crawlify_version, '') as min_crawlify_version,
			COALESCE(linux_amd64_binary_path, '') as linux_amd64_binary_path,
			COALESCE(linux_arm64_binary_path, '') as linux_arm64_binary_path,
			COALESCE(darwin_amd64_binary_path, '') as darwin_amd64_binary_path,
			COALESCE(darwin_arm64_binary_path, '') as darwin_arm64_binary_path,
			COALESCE(binary_hash, '') as binary_hash,
			COALESCE(binary_size_bytes, 0) as binary_size_bytes,
			config_schema,
			downloads,
			published_at
		FROM plugin_versions 
		WHERE plugin_id = $1 AND is_stable = true
		ORDER BY published_at DESC 
		LIMIT 1
	`
	err := r.db.Pool.QueryRow(ctx, query, pluginID).Scan(
		&version.ID, &version.PluginID, &version.Version,
		&version.Changelog, &version.IsStable, &version.MinCrawlifyVersion,
		&version.LinuxAmd64BinaryPath, &version.LinuxArm64BinaryPath,
		&version.DarwinAmd64BinaryPath, &version.DarwinArm64BinaryPath,
		&version.BinaryHash, &version.BinarySizeBytes, &version.ConfigSchema,
		&version.Downloads, &version.PublishedAt,
	)
	return &version, err
}

// GetVersionByID retrieves a specific version by ID
func (r *PluginRepository) GetVersionByID(ctx context.Context, versionID string) (*models.PluginVersion, error) {
	var version models.PluginVersion
	query := `
		SELECT 
			id, plugin_id, version,
			COALESCE(changelog, '') as changelog,
			is_stable,
			COALESCE(min_crawlify_version, '') as min_crawlify_version,
			COALESCE(linux_amd64_binary_path, '') as linux_amd64_binary_path,
			COALESCE(linux_arm64_binary_path, '') as linux_arm64_binary_path,
			COALESCE(darwin_amd64_binary_path, '') as darwin_amd64_binary_path,
			COALESCE(darwin_arm64_binary_path, '') as darwin_arm64_binary_path,
			COALESCE(binary_hash, '') as binary_hash,
			COALESCE(binary_size_bytes, 0) as binary_size_bytes,
			config_schema,
			downloads,
			published_at
		FROM plugin_versions 
		WHERE id = $1
	`
	err := r.db.Pool.QueryRow(ctx, query, versionID).Scan(
		&version.ID, &version.PluginID, &version.Version,
		&version.Changelog, &version.IsStable, &version.MinCrawlifyVersion,
		&version.LinuxAmd64BinaryPath, &version.LinuxArm64BinaryPath,
		&version.DarwinAmd64BinaryPath, &version.DarwinArm64BinaryPath,
		&version.BinaryHash, &version.BinarySizeBytes, &version.ConfigSchema,
		&version.Downloads, &version.PublishedAt,
	)
	return &version, err
}

// ListVersions lists all versions
func (r *PluginRepository) ListVersions(ctx context.Context, pluginID string) ([]*models.PluginVersion, error) {
	query := `
		SELECT 
			id, plugin_id, version, 
			COALESCE(changelog, '') as changelog,
			is_stable,
			COALESCE(min_crawlify_version, '') as min_crawlify_version,
			COALESCE(linux_amd64_binary_path, '') as linux_amd64_binary_path,
			COALESCE(linux_arm64_binary_path, '') as linux_arm64_binary_path,
			COALESCE(darwin_amd64_binary_path, '') as darwin_amd64_binary_path,
			COALESCE(darwin_arm64_binary_path, '') as darwin_arm64_binary_path,
			COALESCE(binary_hash, '') as binary_hash,
			COALESCE(binary_size_bytes, 0) as binary_size_bytes,
			config_schema,
			downloads,
			published_at
		FROM plugin_versions 
		WHERE plugin_id = $1 
		ORDER BY published_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, pluginID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*models.PluginVersion
	for rows.Next() {
		var v models.PluginVersion
		err := rows.Scan(
			&v.ID, &v.PluginID, &v.Version,
			&v.Changelog, &v.IsStable, &v.MinCrawlifyVersion,
			&v.LinuxAmd64BinaryPath, &v.LinuxArm64BinaryPath,
			&v.DarwinAmd64BinaryPath, &v.DarwinArm64BinaryPath,
			&v.BinaryHash, &v.BinarySizeBytes, &v.ConfigSchema,
			&v.Downloads, &v.PublishedAt,
		)
		if err != nil {
			return nil, err
		}
		versions = append(versions, &v)
	}

	return versions, rows.Err()
}

// InstallPlugin records a plugin installation
func (r *PluginRepository) InstallPlugin(ctx context.Context, installation *models.PluginInstallation) error {
	query := `
		INSERT INTO plugin_installations (id, plugin_id, plugin_version_id, workspace_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (plugin_id, workspace_id) DO UPDATE SET
			plugin_version_id = $3, installed_at = NOW()
	`
	_, err := r.db.Pool.Exec(ctx, query,
		installation.ID, installation.PluginID,
		installation.PluginVersionID, installation.WorkspaceID,
	)
	return err
}

// UninstallPlugin removes a plugin installation
func (r *PluginRepository) UninstallPlugin(ctx context.Context, pluginID, workspaceID string) error {
	query := `DELETE FROM plugin_installations WHERE plugin_id = $1 AND workspace_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, pluginID, workspaceID)
	return err
}

// GetInstallation retrieves a plugin installation
func (r *PluginRepository) GetInstallation(ctx context.Context, pluginID, workspaceID string) (*models.PluginInstallation, error) {
	var installation models.PluginInstallation
	query := `SELECT * FROM plugin_installations WHERE plugin_id = $1 AND workspace_id = $2`
	err := r.db.Pool.QueryRow(ctx, query, pluginID, workspaceID).Scan(
		&installation.ID, &installation.PluginID, &installation.PluginVersionID,
		&installation.WorkspaceID, &installation.InstalledAt,
		&installation.LastUsedAt, &installation.UsageCount,
	)
	return &installation, err
}

// ListInstalledPlugins lists all installed plugins
func (r *PluginRepository) ListInstalledPlugins(ctx context.Context, workspaceID string) ([]*models.Plugin, error) {
	query := `
		SELECT p.* FROM plugins p
		INNER JOIN plugin_installations pi ON p.id = pi.plugin_id
		WHERE pi.workspace_id = $1
		ORDER BY pi.installed_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plugins []*models.Plugin
	for rows.Next() {
		var p models.Plugin
		err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description,
			&p.AuthorName, &p.AuthorEmail, &p.RepositoryURL,
			&p.DocumentationURL, &p.PhaseType, &p.PluginType,
			&p.Category, &p.Tags, &p.IsVerified,
			&p.TotalDownloads, &p.TotalInstalls, &p.AverageRating,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, &p)
	}

	return plugins, rows.Err()
}

// Stub implementations for other methods
func (r *PluginRepository) TrackUsage(ctx context.Context, pluginID, workspaceID string) error {
	query := `UPDATE plugin_installations SET usage_count = usage_count + 1, last_used_at = NOW() WHERE plugin_id = $1 AND workspace_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, pluginID, workspaceID)
	return err
}

func (r *PluginRepository) CreateReview(ctx context.Context, review *models.PluginReview) error {
	query := `
		INSERT INTO plugin_reviews (id, plugin_id, user_id, rating, review_text)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (plugin_id, user_id) DO UPDATE SET
			rating = $4, review_text = $5, updated_at = NOW()
	`
	_, err := r.db.Pool.Exec(ctx, query,
		review.ID, review.PluginID, review.UserID, review.Rating, review.ReviewText,
	)
	return err
}

func (r *PluginRepository) ListReviews(ctx context.Context, pluginID string, limit, offset int) ([]*models.PluginReview, error) {
	query := `
		SELECT id, plugin_id, user_id, rating, review_text, created_at, updated_at
		FROM plugin_reviews
		WHERE plugin_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, pluginID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*models.PluginReview
	for rows.Next() {
		var review models.PluginReview
		err := rows.Scan(
			&review.ID, &review.PluginID, &review.UserID,
			&review.Rating, &review.ReviewText,
			&review.CreatedAt, &review.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, &review)
	}

	return reviews, rows.Err()
}

func (r *PluginRepository) GetCategories(ctx context.Context) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			COALESCE(category, 'Uncategorized') as category,
			COUNT(*) as plugin_count
		FROM plugins
		GROUP BY category
		ORDER BY plugin_count DESC
	`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []map[string]interface{}
	for rows.Next() {
		var category string
		var count int
		err := rows.Scan(&category, &count)
		if err != nil {
			return nil, err
		}
		categories = append(categories, map[string]interface{}{
			"name":  category,
			"count": count,
		})
	}

	return categories, rows.Err()
}

func (r *PluginRepository) GetPopularPlugins(ctx context.Context, limit int) ([]*models.Plugin, error) {
	return r.ListPlugins(ctx, models.PluginFilters{Limit: limit})
}

func (r *PluginRepository) SearchPlugins(ctx context.Context, searchQuery string, limit int) ([]*models.Plugin, error) {
	// Use the comprehensive ListPlugins with search query filter
	return r.ListPlugins(ctx, models.PluginFilters{
		SearchQuery: searchQuery,
		Limit:       limit,
	})
}
