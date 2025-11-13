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

## 5. Auto-Discovery

Achievo automatically:
- Scans all mounted drives for game installations
- Detects games by executable files and directory signatures
- Generates placeholder rule files in `configs/games/`
- Registers games for monitoring

No manual rule file creation needed for basic tracking!

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

