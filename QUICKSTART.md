# Achievo Quick Start Guide

## 1. Prerequisites

- Go 1.21+
- MongoDB running locally (see MongoDB Setup below)
- Steam API key (get from https://steamcommunity.com/dev/apikey)

## 2. MongoDB Setup

### Install MongoDB

**Windows:**
1. Download from https://www.mongodb.com/try/download/community
2. Install and start MongoDB service
3. Verify: `mongosh` should connect

**Linux:**
```bash
sudo apt-get install mongodb
sudo systemctl start mongod
sudo systemctl enable mongod
```

**macOS:**
```bash
brew install mongodb-community
brew services start mongodb-community
```

### Verify MongoDB
```bash
mongosh
# Should show: MongoDB server version
```

## 3. Setup Achievo

```bash
# Clone repository
git clone https://github.com/realTNEU/Achievo.git
cd Achievo

# Install dependencies
go mod download

# Build
go build -o achievo.exe ./cmd/achievo
go build -o fetch-steam-schema.exe ./cmd/fetch-steam-schema

# Configure environment variables (recommended)
export ACHIEVO_MONGODB_URI="mongodb://localhost:27017"
export ACHIEVO_STEAM_API_KEY="your_steam_api_key_here"

# OR copy and edit config file
cp config.example.yaml config.yaml
# Edit config.yaml with your settings
```

## 4. Run Achievo

```bash
# With environment variables
./achievo.exe

# OR with config file
./achievo.exe --config config.yaml
```

The service will:
- Discover games on all mounted drives
- Generate rule files for newly discovered games
- Start monitoring for running games
- Log heartbeat messages every minute

## 5. Automatic Game Discovery

Achievo automatically discovers games on your system with comprehensive logging:

### How It Works

1. **On Startup**: Scans all mounted drives (C:\, D:\, etc. on Windows) and standard game paths
2. **Detection Methods**:
   - Finds executable files (.exe) with signature matching
   - Identifies common game folder patterns
   - Detects Steam games via `steam_appid.txt`
   - Recognizes game directories by metadata files (game.ini, config.ini, etc.)
   - Filters out system directories automatically
3. **Rule Generation**: Creates template rule files in `configs/games/auto/` with sanitized names
4. **Database Tracking**: Saves games and marks rules as auto-generated
5. **Periodic Scanning**: Optionally rescans at configurable intervals (default: 24 hours)
6. **Structured Logging**: All activity logged with `[INFO]`, `[WARN]`, `[ERROR]` prefixes

### Auto-Generated Rule Files

When a game is discovered, Achievo creates a rule template at:
```
configs/games/auto/[game-name].yaml
```

These templates include:
- ✅ Executable path placeholders
- ✅ Default save directory guesses (saves/, SaveGames/, etc.)
- ✅ Optional log file watch paths
- ✅ Memory signature sections (disabled by default)
- ✅ Template achievements for customization

### Editing Auto-Generated Rules

1. **Option 1**: Edit the file in `configs/games/auto/` directly
2. **Option 2**: Copy to `configs/games/` to create a manual rule (takes precedence)

**Manual rules always override auto-generated ones** - if a file exists in `configs/games/`, the auto version is ignored.

### Configuration

Enable/disable and configure discovery in `config.yaml`:

```yaml
detection:
  discovery_enabled: true   # Enable automatic discovery
  discovery_interval: 24     # Hours between scans (0 = only on startup)
  scan_interval: 5          # Seconds between process detection scans
```

### Performance Considerations

- **First Scan**: May take several minutes depending on drive size and number of directories
- **Subsequent Scans**: Faster (only checks for new games, skips known ones)
- **Periodic Scans**: Run in background, don't block service operation
- **Large Drives**: System directories are automatically excluded
- **Scan Interval**: Adjust `discovery_interval` to balance freshness vs. performance
- **Drive Exclusion**: System paths (Windows, System32, Program Files, etc.) are filtered automatically

### Review and Refine Generated Rules

1. **Check Auto Directory**: Look in `configs/games/auto/` after discovery
2. **Review Files**: Each file is named `game-[executablename].yaml`
3. **Edit Paths**: Update `save_paths` and `log_patterns` with actual game locations
4. **Add Achievements**: Expand template achievements with game-specific logic
5. **Promote to Manual**: Copy to `configs/games/` to make it a manual rule
6. **Mark Reviewed**: Rules can be marked as reviewed in the database

### Limitations

- **False Positives**: May detect non-game executables (can be manually removed from database)
- **Path Guessing**: Save/log paths are best-guess (may need manual adjustment)
- **Memory Scanning**: Disabled by default, requires explicit config enablement
- **Complex Achievements**: Simple templates only, complex logic needs manual rules
- **Schema Validation**: Invalid rule files are skipped with warnings (check logs)

## 6. Fetch Steam Schemas

For Steam games, fetch achievement definitions:

```bash
./fetch-steam-schema.exe -appid 730
```

This fetches and stores achievement definitions for Counter-Strike 2 (App ID 730).

## 7. Test

1. Start your game
2. Service will detect it automatically (if discovered or rule file exists)
3. Check logs for detection and monitoring messages
4. Trigger achievement conditions
5. Check MongoDB for events:

```bash
mongosh
use achievo
db.events.find().pretty()
db.achievement_progress.find().pretty()
```

## Common Commands

```bash
# Build
go build -o achievo.exe ./cmd/achievo

# Run
./achievo.exe --config config.yaml

# Check MongoDB
mongosh
use achievo
db.events.find().pretty()
db.achievement_progress.find().pretty()
```

## Troubleshooting

- **Game not detected**: Check process name matches exactly
- **Achievement not unlocking**: Verify rule conditions and file paths
- **MongoDB errors**: Ensure MongoDB is running and URI is correct
- **Steam API errors**: Verify API key is valid

See `docs/SETUP.md` for detailed setup instructions.

