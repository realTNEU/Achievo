# Achievo - PC Game Achievement Tracker

A lightweight background service written in Go that tracks achievements and gameplay statistics for PC games, supporting both Steam games and locally installed pirated/non-Steam games.

## Architecture Overview

### Core Components

1. **Game Detection Module** (`internal/detector`)
   - Monitors running processes to detect game launches
   - Identifies games by executable name, window title, or process signatures
   - Maintains a registry of known games

2. **Process Monitor** (`internal/monitor`)
   - Tracks game process lifecycle (start/stop events)
   - Monitors playtime and session duration
   - Handles process state changes

3. **Steam Integration** (`internal/steam`)
   - Fetches achievement definitions from Steam Web API
   - Caches achievement metadata locally
   - Provides canonical achievement definitions for all games

4. **Rule Engine** (`internal/rules`)
   - Modular system for game-specific achievement detection
   - Supports multiple detection methods:
     - File watching (save files, config files)
     - Log pattern matching (regex-based)
     - Incremental counters
     - Memory signature scanning (read-only)
   - Evaluates rules continuously while games are running

5. **File/Log Scanners** (`internal/scanner`)
   - Watches game directories for file changes
   - Parses log files with regex patterns
   - Monitors save file modifications

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
│   └── achievo/
│       └── main.go              # Main service entry point
├── internal/
│   ├── config/                  # Configuration management
│   ├── detector/                # Game detection
│   ├── monitor/                 # Process monitoring
│   ├── steam/                   # Steam API integration
│   ├── rules/                   # Rule engine
│   ├── scanner/                 # File/log scanning
│   ├── tracker/                 # Achievement tracking
│   ├── storage/                 # MongoDB storage layer
│   └── models/                  # Data models
├── pkg/
│   └── memory/                  # Memory scanning utilities
├── configs/
│   └── games/                   # Game-specific rule files
├── docs/                        # Architecture documentation
└── scripts/                     # Utility scripts
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

## Configuration

The service uses a YAML configuration file (`config.yaml`) for:
- MongoDB connection settings
- Steam API key
- Game detection rules
- File watch paths
- Log levels

## Automatic Game Discovery

Achievo automatically scans all mounted drives to discover game installations:

- **Recursive Scanning**: Searches all drives for game directories
- **Smart Detection**: Identifies games by:
  - Executable files (.exe)
  - Common install paths and folder patterns
  - Game metadata files (steam_appid.txt, game.ini, etc.)
  - Directory signatures (saves/, logs/, data/ folders)
- **Auto Rule Generation**: Creates rule templates in `configs/games/auto/` for newly discovered games
- **Periodic Scanning**: Optionally rescans drives periodically (configurable interval)

### Auto-Generated Rule Files

When a game is discovered without an existing rule file, Achievo automatically creates a template in `configs/games/auto/` with:
- Executable path placeholders
- Default save directory guesses
- Optional log file watch paths
- Empty memory signature sections (disabled by default)
- Template achievements for customization

**Manual rule files take precedence** - if a manual rule exists in `configs/games/`, auto-generation is skipped for that game.

## Game Rule Files

Each game has a JSON/YAML rule file that defines:
- Detection criteria (executable name, window title)
- Achievement unlock conditions
- File watch patterns
- Log parsing rules
- Memory signatures

**Rule File Locations:**
- Manual rules: `configs/games/*.yaml` (user-created, take precedence)
- Auto-generated: `configs/games/auto/*.yaml` (templates, can be edited)

## Usage

```bash
# Build the service
go build -o achievo.exe ./cmd/achievo

# Run the service
./achievo.exe --config config.yaml
```

## Development Status

This is v1 of the Achievo client, focusing on:
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

