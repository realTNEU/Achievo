# Achievo v1 - Implementation Summary

## Overview

Achievo v1 is a production-ready desktop background service written in Go that tracks achievements and gameplay statistics for PC games. It supports both Steam games (via Steam Web API) and locally installed pirated/non-Steam games (via a modular rule engine).

## Architecture

The system is built with a modular, extensible architecture:

### Core Modules

1. **Detector** (`internal/detector`) - Detects running games by scanning processes
2. **Monitor** (`internal/monitor`) - Tracks game session lifecycle and playtime
3. **Steam** (`internal/steam`) - Fetches achievement definitions from Steam Web API
4. **Rules** (`internal/rules`) - Evaluates game-specific achievement unlock conditions
5. **Scanner** (`internal/scanner`) - Monitors file system changes and parses logs
6. **Tracker** (`internal/tracker`) - Maps events to achievements and tracks progress
7. **Storage** (`internal/storage`) - MongoDB integration for persistent storage
8. **Service** (`internal/service`) - Main orchestrator that coordinates all modules

### Data Models

- **Game** - Game metadata and detection info
- **Achievement** - Achievement definitions (from Steam or custom)
- **AchievementProgress** - User's achievement progress
- **Session** - Game playing sessions
- **Statistic** - Gameplay statistics
- **Event** - System events (unlocks, updates, etc.)

## Key Features

### Steam Integration
- Fetches achievement definitions from Steam Web API
- Caches definitions locally in MongoDB
- Uses cached definitions as canonical source
- Does NOT read user's personal Steam achievements

### Rule Engine
- Modular JSON/YAML-based rule system
- Multiple detection methods:
  - **File Watching**: Monitor save files, config files
  - **Log Pattern Matching**: Regex-based log parsing
  - **Counters**: Incremental value tracking
  - **Memory Signatures**: Placeholder for read-only memory scanning
- Continuous rule evaluation while games run

### Game Detection
- Automatic process scanning
- Configurable detection criteria (process name, executable path, window title)
- Supports both Steam and non-Steam games
- Tracks game sessions and playtime

### Storage
- MongoDB for persistent storage
- Caches all data locally
- Event queue for future backend sync
- Indexed for performance

## Project Structure

```
achievo/
├── cmd/achievo/          # Main service entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── detector/          # Game detection
│   ├── monitor/           # Process monitoring
│   ├── steam/             # Steam API integration
│   ├── rules/             # Rule engine
│   ├── scanner/           # File/log scanning
│   ├── tracker/           # Achievement tracking
│   ├── storage/           # MongoDB storage
│   ├── service/           # Main orchestrator
│   └── models/            # Data models
├── pkg/memory/            # Memory scanning utilities (placeholder)
├── configs/games/         # Game-specific rule files
└── docs/                  # Documentation
```

## Usage

1. **Configure**: Edit `config.yaml` with MongoDB URI and Steam API key
2. **Add Game Rules**: Create YAML/JSON files in `configs/games/`
3. **Run**: Execute `./achievo.exe --config config.yaml`
4. **Monitor**: Service runs in background, tracking games automatically

## Rule File Format

Each game has a rule file defining:
- Detection criteria (process name, executable path)
- Achievement unlock conditions (file watches, log patterns, counters)

See `docs/RULE_FORMAT.md` for detailed format documentation.

## Extensibility

The architecture is designed for future expansion:

- **Plugin System**: Interface-based design allows custom rule types
- **Platform Support**: Modular structure supports adding new platforms
- **Backend Sync**: Event queue ready for future backend integration
- **Memory Scanning**: Placeholder for platform-specific memory scanning
- **UI**: No GUI in v1, but data models ready for UI layer

## Production Readiness

- ✅ Clean, idiomatic Go code
- ✅ Comprehensive error handling
- ✅ Modular, testable architecture
- ✅ Well-documented interfaces
- ✅ Configuration management
- ✅ Persistent storage
- ✅ Event queuing system
- ✅ Graceful shutdown handling

## Future Enhancements

- Backend synchronization
- UI for viewing achievements
- Additional game platforms
- Memory signature scanning (platform-specific)
- Plugin system for community contributions
- Advanced rule types
- Achievement statistics and analytics

## Dependencies

- MongoDB Driver
- fsnotify (file watching)
- gopsutil (process monitoring)
- YAML parser
- HTTP client (for Steam API)

## License

This is a foundation for a universal PC achievement-tracking platform, designed for long-term maintainability and extensibility.

