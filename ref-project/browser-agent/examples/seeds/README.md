# API Key Seeds

This directory contains text files for bulk seeding API keys into MongoDB.

## File Format

Each text file should contain one API key per line with optional metadata:

```
API_KEY [name=NAME] [rate_limit=N] [daily_limit=N]
```

### Fields:
- **API_KEY** (required): The actual API key
- **name** (optional): A friendly name for the key
- **rate_limit** (optional): Requests per minute limit
- **daily_limit** (optional): Requests per day limit

### Example:
```
AIzaSyABCDEFGHIJKLMNOPQRSTUVWXYZ1234567 name=gemini-production rate_limit=60 daily_limit=1500
sk-proj-ABCDEFGHIJKLMNOP... name=openai-dev rate_limit=500 daily_limit=10000
```

## Comments

Lines starting with `#` are treated as comments and will be ignored.
Empty lines are also ignored.

## Files

- `gemini_api_keys.txt` - Gemini API keys
- `openai_api_keys.txt` - OpenAI API keys

## Usage

Use the seeder CLI command to bulk import keys from these files. See the main documentation for seeder usage instructions.
