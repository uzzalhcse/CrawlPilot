# API Key Seeder

A command-line tool for bulk importing and managing API keys in MongoDB.

## Features

- 🌱 **Bulk Import**: Import multiple API keys from text files
- 📋 **List Keys**: View all stored API keys for a provider
- 📊 **Statistics**: Get usage statistics for your API keys
- 🗑️ **Batch Delete**: Remove all keys for a specific provider
- 🔄 **Smart Deduplication**: Skip keys that already exist in the database
- 🏷️ **Metadata Support**: Add custom metadata to each key (name, rate limits, etc.)

## Installation

Build the seeder:

```bash
cd exp/v2/cmd/seeder
go build -o seeder
```

Or run directly:

```bash
go run main.go [OPTIONS]
```

## Configuration

The seeder uses environment variables for MongoDB connection:

```bash
# .env file
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=crawler_agent
```

Or specify via command-line flags:

```bash
seeder --mongo-uri mongodb://localhost:27017 --mongo-db crawler_agent
```

## Usage

### 1. Prepare Your API Keys File

Create a text file with one API key per line. You can add optional metadata:

**Format:**
```
API_KEY [name=NAME] [rate_limit=N] [daily_limit=N]
```

**Example (`gemini_api_keys.txt`):**
```
# Gemini Production Keys
AIzaSyABCDEFGHIJKLMNOPQRSTUVWXYZ1234567 name=gemini-prod-1 rate_limit=60 daily_limit=1500
AIzaSyBCDEFGHIJKLMNOPQRSTUVWXYZ1234568 name=gemini-prod-2 rate_limit=60 daily_limit=1500

# Gemini Development Keys
AIzaSyCDEFGHIJKLMNOPQRSTUVWXYZ1234569 name=gemini-dev-1 rate_limit=30 daily_limit=500
```

Lines starting with `#` are comments and will be ignored.

### 2. Import API Keys

```bash
# Import Gemini keys
./seeder --file gemini_api_keys.txt --provider gemini

# Import with custom limits
./seeder --file gemini_api_keys.txt --provider gemini --rate-limit 100 --daily-limit 2000

# Import without skipping duplicates (will error on duplicates)
./seeder --file keys.txt --provider gemini --skip-existing=false
```

### 3. List API Keys

```bash
# List all keys for Gemini
./seeder --list --provider gemini

# List all keys for OpenAI
./seeder --list --provider openai
```

**Output:**
```
═══════════════════════════════════════
Found 3 key(s):
═══════════════════════════════════════
1. 🟢 Active AIzaSyABCD...Z1234567
   Provider: gemini
   Usage: 45 times
   Failures: 0
   Rate Limit: 60/min
   Daily Limit: 1500/day
   Last Used: 2025-01-15 14:23:45
   ---
2. 🟢 Active AIzaSyBCDE...Z1234568
   Provider: gemini
   Usage: 38 times
   Failures: 0
   ...
```

### 4. View Statistics

```bash
./seeder --stats --provider gemini
```

**Output:**
```
═══════════════════════════════════════
📊 Statistics for: gemini
═══════════════════════════════════════
   Total Keys: 5
   Active Keys: 5
   Inactive Keys: 0
   Total Usage: 234 requests
═══════════════════════════════════════
```

### 5. Delete All Keys (Use with Caution!)

```bash
./seeder --delete-all --provider gemini
```

This will prompt for confirmation:
```
⚠️  WARNING: This will delete ALL API keys for provider 'gemini'
Type 'yes' to confirm:
```

## File Format Examples

### Simple Format (No Metadata)
```
AIzaSyABCDEFGHIJKLMNOPQRSTUVWXYZ1234567
AIzaSyBCDEFGHIJKLMNOPQRSTUVWXYZ1234568
AIzaSyCDEFGHIJKLMNOPQRSTUVWXYZ1234569
```

### With Metadata
```
AIzaSyABCDEFGHIJKLMNOPQRSTUVWXYZ1234567 name=production-key-1 rate_limit=100 daily_limit=5000
AIzaSyBCDEFGHIJKLMNOPQRSTUVWXYZ1234568 name=production-key-2 rate_limit=100 daily_limit=5000
AIzaSyCDEFGHIJKLMNOPQRSTUVWXYZ1234569 name=dev-key-1 rate_limit=30 daily_limit=1000
```

### With Comments
```
# Production Keys - High Limits
AIzaSyABCDEFGHIJKLMNOPQRSTUVWXYZ1234567 rate_limit=200 daily_limit=10000

# Development Keys - Low Limits
AIzaSyBCDEFGHIJKLMNOPQRSTUVWXYZ1234568 rate_limit=30 daily_limit=500

# Testing Keys - No Limits
AIzaSyCDEFGHIJKLMNOPQRSTUVWXYZ1234569
```

## Command Reference

### Seeding Operation
```bash
seeder --file <path> --provider <name> [OPTIONS]
```

**Required:**
- `--file`: Path to API keys file
- `--provider`: Provider name (gemini, openai, etc.)

**Optional:**
- `--rate-limit`: Rate limit per minute (default: 60)
- `--daily-limit`: Daily request limit (default: 1000)
- `--skip-existing`: Skip duplicate keys (default: true)

### List Operation
```bash
seeder --list --provider <name>
```

### Statistics Operation
```bash
seeder --stats --provider <name>
```

### Delete Operation
```bash
seeder --delete-all --provider <name>
```

### Global Options
- `--mongo-uri`: MongoDB URI (default: from env or localhost)
- `--mongo-db`: Database name (default: from env or crawler_agent)
- `--log-level`: Log level: debug, info, warn, error (default: info)

## Example Workflows

### Initial Setup for Multiple Providers

```bash
# Seed Gemini keys
./seeder --file seeds/gemini_api_keys.txt --provider gemini --rate-limit 60

# Seed OpenAI keys
./seeder --file seeds/openai_api_keys.txt --provider openai --rate-limit 500

# Verify
./seeder --stats --provider gemini
./seeder --stats --provider openai
```

### Updating Keys

```bash
# Add new keys (existing keys will be skipped)
./seeder --file new_gemini_keys.txt --provider gemini

# Or replace all keys
./seeder --delete-all --provider gemini
./seeder --file gemini_api_keys.txt --provider gemini
```

### Monitoring

```bash
# Check key status
./seeder --list --provider gemini

# View usage statistics
./seeder --stats --provider gemini

# Debug mode for detailed logs
./seeder --list --provider gemini --log-level debug
```

## Integration with API Key Rotation

After seeding keys, they'll be automatically used by the API key rotation system:

```go
// Your application code
manager := apikey.NewManager(&apikey.ManagerConfig{
    MongoClient: mongoClient,
    SyncTTL:     5 * time.Minute,
})

// Keys are automatically loaded
model, err := apikey.NewGeminiModel(ctx, manager, "gemini-2.0-flash-exp", nil)
```

## Troubleshooting

### Duplicate Key Error
```
Error: API key already exists for provider gemini
```
**Solution**: Use `--skip-existing=true` (default) or remove the duplicate key from the database first.

### Connection Error
```
Error: Failed to connect to MongoDB
```
**Solution**: Check your MongoDB URI and ensure MongoDB is running.

### Permission Error
```
Error: Failed to open file: permission denied
```
**Solution**: Check file permissions: `chmod 644 gemini_api_keys.txt`

## Best Practices

1. **Keep Keys Secure**: Store key files outside of version control (add to `.gitignore`)
2. **Use Comments**: Document your keys with comments in the file
3. **Set Appropriate Limits**: Configure rate limits based on your provider's quotas
4. **Regular Monitoring**: Check statistics regularly to detect unusual usage patterns
5. **Backup Before Delete**: Always backup your database before using `--delete-all`

## Security Notes

- Never commit API key files to version control
- Use environment variables for MongoDB credentials
- Restrict access to the seeder binary in production
- Regularly rotate your API keys
- Monitor for unusual usage patterns

## See Also

- [API Key Rotation Documentation](../../API_KEY_ROTATION.md)
- [Quick Start Guide](../../QUICKSTART_ROTATION.md)
- [Example Seeds](../../examples/seeds/)
