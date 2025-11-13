# Achievo Quick Start Guide

## 1. Prerequisites

- Go 1.21+
- MongoDB running locally
- Steam API key (get from https://steamcommunity.com/dev/apikey)

## 2. Setup

```bash
# Install dependencies
go mod download

# Build
go build -o achievo.exe ./cmd/achievo

# Configure (edit config.yaml)
# - Set MongoDB URI
# - Set Steam API key
```

## 3. Create a Game Rule

Create `configs/games/my-game.yaml`:

```yaml
game_id: "my-game"
game_name: "My Game"

detection:
  process_name: "mygame.exe"

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

## 4. Run

```bash
./achievo.exe --config config.yaml
```

## 5. Test

1. Start your game
2. Service will detect it automatically
3. Trigger achievement condition (e.g., create the file)
4. Check MongoDB for unlock event

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

