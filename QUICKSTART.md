# Achievo Quick Start Guide

## 1. Prerequisites

- **Go 1.21+** - [Download](https://golang.org/dl/)
- **MongoDB** - [Download](https://www.mongodb.com/try/download/community)
- **Steam API Key** (optional) - Get from [Steam Web API](https://steamcommunity.com/dev/apikey)
- **Ollama** (optional, for LLM-powered discovery) - [Download](https://ollama.com/download)

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

## 3. LLM Setup (Optional but Recommended)

For intelligent game discovery, you can use either:
- **Local Ollama**: Run Ollama on your machine
- **Remote Ollama Server**: Use a remote server (e.g., `https://ollama.tneuserver.online`)

### Option A: Local Ollama Setup

**Install Ollama:**

**Windows:**
```powershell
winget install Ollama.Ollama
# Or download from https://ollama.com/download
```

**Linux:**
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

**macOS:**
```bash
brew install ollama
```

**Download LLaMA 3.1 8B Model:**
```bash
ollama pull llama3.1:8b
```

This will download ~4.7GB. The model provides comprehensive game analysis including path inference.

### Option B: Remote Ollama Server

If you have a remote Ollama server (e.g., `https://ollama.tneuserver.online`):

1. **Verify server is accessible:**
   ```bash
   curl -i https://ollama.tneuserver.online/api/tags
   ```

2. **Ensure model is available on server:**
   The server must have `llama3.1:8b` model installed.

3. **Configure Achievo:**
   ```yaml
   detection:
     llm_enabled: true
     llm_api_url: "https://ollama.tneuserver.online"
     llm_model_name: "llama3.1:8b"
   ```

**Note**: If you don't set up LLM, Achievo will use heuristic-based detection (still functional, but less accurate).

See [docs/LLM_SETUP.md](docs/LLM_SETUP.md) for detailed LLM setup instructions.

## 4. Setup Achievo

```bash
# Clone repository
git clone https://github.com/realTNEU/Achievo.git
cd Achievo

# Install dependencies
go mod download

# Build
go build -o achievo.exe ./cmd/achievo
go build -o fetch-steam-schema.exe ./cmd/fetch-steam-schema
```

## 5. Configuration

### Option 1: Environment Variables (Recommended for Secrets)

```bash
# Windows PowerShell
$env:ACHIEVO_MONGODB_URI="mongodb://localhost:27017"
$env:ACHIEVO_STEAM_API_KEY="your_steam_api_key_here"

# Linux/macOS
export ACHIEVO_MONGODB_URI="mongodb://localhost:27017"
export ACHIEVO_STEAM_API_KEY="your_steam_api_key_here"
```

### Option 2: Configuration File

```bash
# Copy example config
cp config.example.yaml config.yaml

# Edit config.yaml with your settings
```

**Example `config.yaml` (Local Ollama):**
```yaml
mongodb:
  uri: "mongodb://localhost:27017"
  database: "achievo"

steam:
  api_key: ""  # Or set via ACHIEVO_STEAM_API_KEY env var

detection:
  discovery_enabled: true
  discovery_interval: 24  # hours (0 = only on startup)
  custom_scan_paths:
    - "F:/games"  # Your game directories
  llm_enabled: true  # Enable LLM-powered discovery
  llm_api_url: "http://localhost:11434"  # Local Ollama
  llm_model_name: "llama3.1:8b"  # LLaMA 3.1 8B

memory_scanning:
  enabled: false  # Disabled by default for security
```

**Example `config.yaml` (Remote Ollama Server):**
```yaml
mongodb:
  uri: "mongodb://localhost:27017"
  database: "achievo"

steam:
  api_key: ""  # Or set via ACHIEVO_STEAM_API_KEY env var

detection:
  discovery_enabled: true
  discovery_interval: 24
  custom_scan_paths:
    - "F:/games"
  llm_enabled: true
  llm_api_url: "https://ollama.tneuserver.online"  # Remote Ollama server
  llm_model_name: "llama3.1:8b"

memory_scanning:
  enabled: false
```

## 6. Run Achievo

```bash
# With environment variables
./achievo.exe

# OR with config file
./achievo.exe --config config.yaml
```

The service will:
- ✅ Connect to MongoDB
- ✅ Discover games on all mounted drives (and custom paths)
- ✅ Use LLM to analyze and classify game directories (if enabled)
- ✅ Generate rule files for newly discovered games
- ✅ Start monitoring for running games
- ✅ Log heartbeat messages every minute

## 7. Automatic Game Discovery

### How It Works

1. **LLM Analysis** (if enabled):
   - Analyzes directory structure, files, and metadata
   - Determines if directory is a game vs software/utility
   - Infers game name, root folder, executable, save/log paths
   - Creates rule files with inferred paths

2. **Heuristic Fallback** (if LLM disabled/unavailable):
   - Finds executable files (.exe) with signature matching
   - Identifies common game folder patterns
   - Detects Steam games via `steam_appid.txt`
   - Recognizes game directories by metadata files

3. **Rule Generation**:
   - Creates template rule files in `configs/games/auto/`
   - Uses LLM-inferred paths when available
   - Falls back to heuristic path detection
   - Validates against JSON schema before loading

### Auto-Generated Rule Files

When a game is discovered, Achievo automatically creates a rule template at:
```
configs/games/auto/game-[executablename].yaml
```

**Example**: `configs/games/auto/game-mygame.yaml`

These templates include:
- ✅ **executable_path**: Full path to the game executable
- ✅ **save_paths**: LLM-inferred or heuristic save directory locations
- ✅ **log_patterns**: Detected log file paths with regex patterns
- ✅ **memory_signatures**: Empty array (disabled by default)
- ✅ **Template achievements**: Placeholder achievements ready for customization

**Important**: 
- Memory scanning is **disabled by default** for security. To enable it, set `memory_scanning.enabled: true` in your `config.yaml`.
- **Manual rules always override auto-generated ones** - if a file exists in `configs/games/`, the auto version is ignored.

### Editing Auto-Generated Rules

1. **Option 1**: Edit the file in `configs/games/auto/` directly
2. **Option 2**: Copy to `configs/games/` to create a manual rule (takes precedence)

### Configuration Options

```yaml
detection:
  discovery_enabled: true   # Enable automatic discovery
  discovery_interval: 24    # Hours between scans (0 = only on startup)
  custom_scan_paths:        # Custom directories to scan
    - "F:/games"
    - "D:/SteamLibrary/steamapps/common"
  llm_enabled: true         # Enable LLM-powered discovery
  llm_api_url: "http://localhost:11434"
  llm_model_name: "llama3.1:8b"
```

### Performance Considerations

- **First Scan**: May take several minutes depending on drive size
- **LLM Analysis**: ~1-3 seconds per directory (with LLaMA 3.1 8B)
- **Subsequent Scans**: Faster (only checks for new games)
- **Periodic Scans**: Run in background, don't block service operation
- **Large Drives**: System directories are automatically excluded

## 8. Fetch Steam Schemas

For Steam games, fetch achievement definitions:

```bash
./fetch-steam-schema.exe -appid 730
```

This fetches and stores achievement definitions for Counter-Strike 2 (App ID 730).

## 9. Test

1. Start your game
2. Service will detect it automatically (if discovered or rule file exists)
3. Check logs for detection and monitoring messages:
   ```
   [INFO] Game discovered: MyGame | ID: game-mygame-1234567890
   [INFO] Generated auto rule file for MyGame at configs\games\auto\game-mygame.yaml
   [LLM] Classified as GAME: MyGame (confidence: 0.95, reason: "Contains game executable and save directory")
   ```
4. Trigger achievement conditions
5. Check MongoDB for events:

```bash
mongosh
use achievo
db.events.find().pretty()
db.achievement_progress.find().pretty()
db.games.find().pretty()
```

## Common Commands

```bash
# Build
go build -o achievo.exe ./cmd/achievo
go build -o fetch-steam-schema.exe ./cmd/fetch-steam-schema

# Run
./achievo.exe

# Check MongoDB
mongosh
use achievo
db.games.find().pretty()
db.sessions.find().pretty()
db.events.find().pretty()
db.achievement_progress.find().pretty()

# Check Ollama (if using LLM)
ollama list
ollama show llama3.1:8b
```

## Troubleshooting

### Game Not Detected
- Check if game executable matches process name in rule file
- Verify game is in a scanned directory (mounted drive or custom path)
- Check logs for `[SKIP]` messages indicating why it was filtered

### LLM Not Working

**For Local Ollama:**
- Verify Ollama is running: `ollama list`
- Check model is downloaded: `ollama list` should show `llama3.1:8b`
- Test API: `curl http://localhost:11434/api/tags`
- Check firewall/network settings

**For Remote Ollama:**
- Test API: `curl -i https://ollama.tneuserver.online/api/tags`
- Verify HTTPS is working and server is accessible
- Check network connectivity and firewall rules
- Ensure model exists on remote server: Check server logs or API response

**Note**: Service will fall back to heuristics if LLM unavailable

### Achievement Not Unlocking
- Verify rule conditions and file paths in rule file
- Check if save/log paths are correct (may need manual adjustment)
- Review logs for rule evaluation messages

### MongoDB Errors
- Ensure MongoDB is running: `mongosh` should connect
- Verify URI is correct in config or environment variable
- Check MongoDB logs for connection issues

### Steam API Errors
- Verify API key is valid and set correctly
- Check API key in environment variable or config file
- Steam API is optional - service works without it

## Next Steps

1. **Review Discovered Games**: Check `configs/games/auto/` for auto-generated rules
2. **Edit Rules**: Customize achievement detection logic in rule files
3. **Fetch Steam Schemas**: Use `fetch-steam-schema.exe` for Steam games
4. **Monitor Logs**: Watch for game detection and achievement unlocks
5. **Check Database**: Query MongoDB to see stored games, sessions, and achievements

For more information, see:
- [README.md](README.md) - Project overview and architecture
- [docs/LLM_SETUP.md](docs/LLM_SETUP.md) - Detailed LLM setup guide
