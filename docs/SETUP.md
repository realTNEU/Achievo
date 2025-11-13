# Achievo Setup Guide

## Prerequisites

1. **Go 1.21 or later** - [Download](https://golang.org/dl/)
2. **MongoDB** - [Download](https://www.mongodb.com/try/download/community)
3. **Steam API Key** - Get one from [Steam Web API](https://steamcommunity.com/dev/apikey)

## Installation

1. Clone or download the Achievo repository
2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the service:
   ```bash
   go build -o achievo.exe ./cmd/achievo
   ```

## Configuration

1. Copy the example configuration:
   ```bash
   cp config.yaml config.local.yaml
   ```

2. Edit `config.local.yaml` with your settings:
   ```yaml
   mongodb:
     uri: "mongodb://localhost:27017"
     database: "achievo"
     timeout: 10

   steam:
     api_key: "YOUR_STEAM_API_KEY_HERE"

   detection:
     scan_interval: 5  # seconds between detection scans
     process_check: true

   logging:
     level: "info"  # debug, info, warn, error
     format: "text"  # json or text

   paths:
     game_rules: "configs/games"
     cache: ".cache"
   ```

## MongoDB Setup

1. Start MongoDB:
   ```bash
   # Windows
   mongod

   # Linux/Mac
   sudo systemctl start mongod
   ```

2. Verify MongoDB is running:
   ```bash
   mongosh
   ```

## Running the Service

```bash
./achievo.exe --config config.local.yaml
```

The service will:
- Start monitoring for running games
- Load game rules from `configs/games/`
- Connect to MongoDB
- Begin tracking achievements

## Adding Game Rules

Create a YAML or JSON file in `configs/games/` for each game you want to track.

Example (`configs/games/my-game.yaml`):
```yaml
game_id: "my-game-001"
game_name: "My Game"

detection:
  process_name: "mygame.exe"
  window_title: "My Game"
  executable: "C:\\Games\\MyGame\\mygame.exe"

achievements:
  - id: "ach-first"
    name: "First Achievement"
    rules:
      - id: "rule-1"
        type: "file_watch"
        config:
          path: "saves/achievement.dat"
          condition: "exists"
```

## Steam Games

For Steam games, the service will automatically:
1. Detect the game by process name
2. Fetch achievement definitions from Steam API
3. Cache them locally in MongoDB
4. Use them for tracking

You need to register Steam games by creating a rule file with the Steam App ID:
```yaml
game_id: "steam-game-123"
game_name: "Steam Game"
steam_app_id: "123456"  # Steam App ID

detection:
  process_name: "steamgame.exe"
```

## Troubleshooting

### MongoDB Connection Issues
- Ensure MongoDB is running
- Check the connection URI in config
- Verify network/firewall settings

### Game Not Detected
- Check process name matches exactly (case-insensitive)
- Verify executable path if specified
- Check game rules file is in `configs/games/`

### Achievements Not Unlocking
- Verify rule conditions are correct
- Check file paths are relative to game directory
- Review log output for rule evaluation errors
- Ensure file watching is working (check logs)

### Steam API Issues
- Verify API key is correct
- Check Steam API rate limits
- Ensure Steam App ID is correct

## Development

### Building
```bash
go build -o achievo.exe ./cmd/achievo
```

### Running Tests
```bash
go test ./...
```

### Code Formatting
```bash
go fmt ./...
```

### Linting
```bash
golangci-lint run
```

## Next Steps

- Add more game rule files
- Customize detection rules
- Monitor the event queue for future backend sync
- Extend rule engine with custom rule types

