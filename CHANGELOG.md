# Changelog

All notable changes to Achievo will be documented in this file.

## [1.0.0] - 2025-01-XX

### Added
- **LLM-Powered Game Discovery**: Integration with LLaMA 3.1 8B for intelligent game classification
- **Automatic Rule Generation**: Auto-generates rule files with LLM-inferred save/log paths
- **Custom Scan Paths**: Configure specific directories to scan in addition to mounted drives
- **Hybrid Detection System**: LLM analysis with heuristic fallback
- **Comprehensive Game Analysis**: LLM identifies game root, executable, name, and paths
- **Steam Integration**: Fetches achievement definitions from Steam Web API
- **Session Tracking**: Monitors game sessions and playtime
- **Achievement Detection**: Multiple detection methods (file watching, log parsing, counters)
- **MongoDB Storage**: Persistent storage for games, sessions, achievements, and events
- **Rule Engine**: Modular JSON/YAML-based rule system
- **Process Monitoring**: Cross-platform game detection (Windows/Linux)

### Configuration
- LLM settings (API URL, model name)
- Custom scan paths
- Discovery intervals
- Memory scanning (disabled by default)

### Documentation
- Comprehensive README with architecture overview
- Quick start guide with setup instructions
- LLM setup guide with troubleshooting
- Example configuration files
- Example rule files

