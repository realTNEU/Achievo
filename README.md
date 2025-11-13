# Achievo - PC Game Achievement Tracker

A lightweight background service written in Go that tracks achievements and gameplay statistics for PC games, supporting both Steam games and locally installed non-Steam games. Features intelligent game detection powered by LLaMA 3.1 8B for accurate classification and automatic rule generation.

## Features

- **Intelligent Game Discovery**: Uses LLaMA 3.1 8B to analyze directory structures and identify games vs software utilities
- **Automatic Rule Generation**: Creates rule templates with LLM-inferred save/log paths
- **Steam Integration**: Fetches achievement definitions from Steam Web API
- **Hybrid Detection**: LLM-powered analysis with heuristic fallback
- **Custom Scan Paths**: Configure specific directories to scan
- **Session Tracking**: Monitors game sessions and playtime
- **Achievement Detection**: Multiple detection methods (file watching, log parsing, counters)
- **MongoDB Storage**: Persistent storage for games, sessions, achievements, and events

## Architecture Overview

### Core Components

1. **Game Discovery** (`internal/discovery`)
   - LLM-powered directory analysis using LLaMA 3.1 8B
   - Scans mounted drives and custom paths
   - Identifies game root folders, executables, and save/log paths
   - Falls back to heuristics if LLM unavailable

2. **Game Detection** (`internal/detector`)
   - Monitors running processes to detect game launches
   - Identifies games by executable name, window title, or process signatures
   - Maintains a registry of known games

3. **Process Monitor** (`internal/monitor`)
   - Tracks game process lifecycle (start/stop events)
   - Monitors playtime and session duration
   - Handles process state changes

4. **Steam Integration** (`internal/steam`)
   - Fetches achievement definitions from Steam Web API
   - Caches achievement metadata locally in MongoDB
   - Provides canonical achievement definitions for all games

5. **Rule Engine** (`internal/rules`)
   - Modular system for game-specific achievement detection
   - Supports multiple detection methods:
     - File watching (save files, config files)
     - Log pattern matching (regex-based)
     - Incremental counters
     - Memory signature scanning (read-only, disabled by default)
   - Evaluates rules continuously while games are running

6. **Storage Layer** (`internal/storage`)
   - MongoDB integration for persistent storage
   - Caches game sessions, playtime, stats, achievements
   - Maintains event queue for future backend sync

7. **Configuration Management** (`internal/config`)
   - Loads and manages application configuration
   - Handles game-specific rule files (JSON/YAML)
   - Manages API keys and database connections

8. **Achievement Tracker** (`internal/tracker`)
   - Maps detected events to achievement definitions
   - Tracks user progress
   - Emits achievement unlock events

## Project Structure

```
achievo/
├── cmd/
│   ├── achievo/
│   │   └── main.go              # Main service entry point
│   └── fetch-steam-schema/
│       └── main.go               # Steam schema fetcher utility
├── internal/
│   ├── config/                   # Configuration management
│   ├── detector/                  # Game detection
│   ├── discovery/                 # LLM-powered game discovery
│   │   └── llm_classifier.go     # LLaMA integration
│   ├── monitor/                  # Process monitoring
│   ├── steam/                    # Steam API integration
│   ├── rules/                    # Rule engine
│   ├── scanner/                  # File/log scanning
│   ├── tracker/                  # Achievement tracking
│   ├── storage/                  # MongoDB storage layer
│   └── models/                   # Data models
├── pkg/
│   └── memory/                   # Memory scanning utilities
├── configs/
│   └── games/                    # Game-specific rule files
│       ├── auto/                 # Auto-generated rules
│       └── *.yaml                # Manual rules
├── docs/
│   └── LLM_SETUP.md              # LLM setup guide
├── README.md                     # This file
├── QUICKSTART.md                 # Quick start guide
├── config.example.yaml           # Configuration template
└── go.mod                        # Go module definition
```

## Data Models

### Game
- ID, Name, Type (Steam/Non-Steam)
- Executable path, Process name
- Steam App ID (if applicable)
- Installation directory

### Achievement
- ID, Name, Description, Icon URL
- Game ID, Steam App ID
- Unlock conditions (rule references)

### Game Session
- Game ID, Start time, End time
- Playtime duration
- Process ID

### Achievement Progress
- Achievement ID, Game ID
- Unlocked status, Unlock timestamp
- Progress percentage (for incremental achievements)

### Statistic
- Game ID, Stat name, Value
- Last updated timestamp

### Event
- Type (achievement_unlock, stat_update, etc.)
- Game ID, Timestamp
- Payload (JSON)
- Sync status

## LLM-Powered Game Discovery

Achievo uses **LLaMA 3.1 8B** (via Ollama or LM Studio) to intelligently analyze game directories:

### How It Works

1. **Directory Analysis**: LLM analyzes directory structure, files, and metadata
2. **Game Classification**: Determines if directory is a real game vs software/utility
3. **Path Inference**: Identifies:
   - Game root folder
   - Main executable
   - Save file paths
   - Log file paths
   - Config file paths
4. **Rule Generation**: Auto-generates rule files with LLM-inferred paths

### Configuration

**Local Ollama:**
```yaml
detection:
  llm_enabled: true
  llm_api_url: "http://localhost:11434"  # Local Ollama
  llm_model_name: "llama3.1:8b"          # LLaMA 3.1 8B
  custom_scan_paths:
    - "F:/games"                          # Custom directories
```

**Remote Ollama Server:**
```yaml
detection:
  llm_enabled: true
  llm_api_url: "https://ollama.tneuserver.online"  # Remote server
  llm_model_name: "llama3.1:8b"
  custom_scan_paths:
    - "F:/games"
```

See [docs/LLM_SETUP.md](docs/LLM_SETUP.md) for detailed setup instructions.

## Automatic Game Discovery

### Drive Scanning
- Scans all mounted drives (C:\, D:\, etc. on Windows)
- Includes standard game paths (Steam directories, ~/Games)
- Supports custom scan paths via configuration
- Smart filtering excludes system directories automatically

### Detection Methods
- **LLM Analysis**: Primary method using LLaMA 3.1 8B
- **Heuristic Fallback**: Executable signatures, metadata files, directory patterns
- **Steam Detection**: Identifies Steam games via `steam_appid.txt`

### Auto Rule Generation
- Creates rule templates in `configs/games/auto/`
- Uses LLM-inferred paths when available
- Falls back to heuristic path detection
- Validates against JSON schema before loading

**Manual rule files take precedence** - if a manual rule exists in `configs/games/`, auto-generation is skipped.

## Game Rule Files

Each game has a JSON/YAML rule file that defines:
- Detection criteria (executable name, window title)
- Achievement unlock conditions
- File watch patterns
- Log parsing rules
- Memory signatures (disabled by default)

**Rule File Locations:**
- Manual rules: `configs/games/*.yaml` (user-created, take precedence)
- Auto-generated: `configs/games/auto/*.yaml` (templates, can be edited)

## Quick Start

1. **Prerequisites**:
   - Go 1.21+
   - MongoDB running locally
   - Steam API key (optional, for Steam games)
   - Ollama with LLaMA 3.1 8B (optional, for LLM-powered discovery)

2. **Install**:
   ```bash
   git clone https://github.com/realTNEU/Achievo.git
   cd Achievo
   go mod download
   go build -o achievo.exe ./cmd/achievo
   ```

3. **Configure**:
   ```bash
   cp config.example.yaml config.yaml
   # Edit config.yaml with your settings
   # Or use environment variables:
   export ACHIEVO_MONGODB_URI="mongodb://localhost:27017"
   export ACHIEVO_STEAM_API_KEY="your_key_here"
   ```

4. **Run**:
   ```bash
   ./achievo.exe
   ```

See [QUICKSTART.md](QUICKSTART.md) for detailed setup instructions.

## Configuration

The service uses a YAML configuration file (`config.yaml`) for:
- MongoDB connection settings
- Steam API key
- LLM settings (Ollama/LM Studio URL and model)
- Custom scan paths
- Discovery intervals
- Memory scanning (disabled by default)

See `config.example.yaml` for all available options.

## Development Status

This is v1 of the Achievo client, focusing on:
- ✅ LLM-powered game discovery
- Core detection and monitoring
- Steam achievement definition fetching
- Rule-based achievement detection
- Local storage and event queuing
- Extensible architecture for future enhancements

Future versions will include:
- Backend synchronization
- UI for viewing achievements
- Additional game platform support
- Plugin system for community contributions

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.
