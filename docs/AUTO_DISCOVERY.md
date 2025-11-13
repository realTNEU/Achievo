# Automatic Game Discovery System

## Overview

Achievo's automatic game discovery system scans all mounted drives to find game installations and automatically generates rule templates for newly discovered games.

## How It Works

### Discovery Process

1. **Drive Scanning**: On startup (and periodically if enabled), Achievo scans all mounted drives:
   - Windows: C:\, D:\, E:\, etc.
   - Linux: /, /home, /mnt, /media

2. **Game Detection**: Identifies potential game directories by:
   - **Executable Files**: Finds .exe files (excluding system executables)
   - **Directory Patterns**: Recognizes common game folder names
   - **Metadata Files**: Detects `steam_appid.txt`, `game.ini`, `config.ini`, etc.
   - **Directory Signatures**: Looks for game-like subdirectories (saves/, logs/, data/, config/)

3. **Game Registration**: For each discovered game:
   - Creates a Game record in MongoDB
   - Generates a rule template file
   - Registers the game with the detector
   - Fetches Steam schemas if applicable

### Rule File Generation

When a game is discovered without an existing rule file, Achievo creates a template at:
```
configs/games/auto/[normalized-game-name].yaml
```

#### Template Contents

The auto-generated rule file includes:

1. **Detection Configuration**:
   - Process name (from executable)
   - Executable path (full path placeholder)
   - Window title (if detected)

2. **Placeholder Achievements**:
   - **Save Game Detection**: Auto-detected or default save path
   - **Log File Watching**: Auto-detected or default log path
   - **Memory Signature**: Template with disabled flag
   - **Custom Template**: Example achievement structure

3. **Path Inference**:
   - Checks for common save directories (saves/, SaveGames/, etc.)
   - Looks for log directories (logs/, log/, etc.)
   - Checks Windows AppData for save files
   - Provides defaults if nothing found

## Configuration

### Enable/Disable Discovery

In `config.yaml`:
```yaml
detection:
  discovery_enabled: true  # Enable automatic discovery
  discovery_interval: 24    # Hours between scans (0 = only on startup)
```

### Performance Tuning

- **First Scan**: May take several minutes on large drives
- **Subsequent Scans**: Only checks for new games (much faster)
- **Periodic Scans**: Run in background, configurable interval
- **Excluded Paths**: System directories are automatically excluded

## Manual vs Auto-Generated Rules

### Priority System

1. **Manual Rules** (`configs/games/*.yaml`):
   - User-created rule files
   - Take precedence over auto-generated rules
   - If a manual rule exists, auto-generation is skipped

2. **Auto-Generated Rules** (`configs/games/auto/*.yaml`):
   - Template files created by discovery
   - Can be edited directly
   - Can be copied to `configs/games/` to promote to manual

### Workflow

1. **Discovery finds a game** → Checks for manual rule
2. **If manual exists** → Uses manual rule, skips auto-generation
3. **If no manual** → Generates auto rule in `configs/games/auto/`
4. **User can edit** → Either edit auto file or copy to manual

## Editing Auto-Generated Rules

### Option 1: Edit Auto File Directly
```bash
# Edit the auto-generated file
nano configs/games/auto/my-game.yaml
```

### Option 2: Promote to Manual Rule
```bash
# Copy to manual rules directory
cp configs/games/auto/my-game.yaml configs/games/my-game.yaml
# Now edit the manual version
nano configs/games/my-game.yaml
```

## Rule File Structure

### Auto-Generated Template

```yaml
game_id: "game-example-1234567890"
game_name: "Example Game"

detection:
  process_name: "game.exe"
  executable: "C:\\Games\\Example\\game.exe"

achievements:
  - id: "ach-save-game"
    name: "Progress Saved (Auto-detected)"
    rules:
      - id: "rule-save-file"
        type: "file_watch"
        config:
          path: "saves/*.sav"
          condition: "modified"
          comment: "Auto-detected save file path. Adjust if incorrect."

  - id: "ach-log-detected"
    name: "Game Activity Detected (Auto-detected)"
    rules:
      - id: "rule-log-pattern"
        type: "log_pattern"
        config:
          pattern: ".*"
          file: "logs/game.log"
          comment: "Auto-detected log file. Customize pattern for specific achievements."

  - id: "ach-memory-example"
    name: "Memory Signature Example (Placeholder - Disabled)"
    rules:
      - id: "rule-memory-sig"
        type: "memory_signature"
        config:
          pattern: "48 89 5C 24 ?? ?? 00"
          offset: 0
          enabled: false
          comment: "Memory signature scanning is disabled by default."
```

## Limitations

1. **False Positives**: May detect non-game executables
   - Solution: Manually remove or exclude in config

2. **Path Guessing**: Save/log paths are best-guess
   - Solution: Edit rule file with actual paths

3. **Complex Achievements**: Simple templates only
   - Solution: Manually create detailed rules

4. **Memory Scanning**: Requires manual configuration
   - Solution: Enable in config and provide patterns

5. **Performance**: Large drives take time to scan
   - Solution: Adjust discovery interval or exclude paths

## Best Practices

1. **Review Auto-Generated Rules**: Check `configs/games/auto/` after first scan
2. **Promote Important Games**: Copy frequently-played games to manual rules
3. **Customize Paths**: Update save/log paths with actual locations
4. **Add Achievements**: Expand template achievements with game-specific logic
5. **Test Rules**: Verify rules work before relying on them

## Troubleshooting

### Games Not Discovered

- Check if executable exists and is accessible
- Verify game directory has recognizable structure
- Check logs for discovery errors
- Manually create rule file if needed

### Incorrect Paths in Templates

- Edit the auto-generated rule file
- Update save/log paths with actual locations
- Test file watching with actual game files

### Too Many False Positives

- Review discovered games in MongoDB
- Remove unwanted games from database
- Adjust discovery heuristics if needed

## Logging

Discovery actions are logged with clear prefixes:

- `Starting game discovery scan...` - Discovery started
- `Discovery scan completed in X, found Y potential games` - Scan summary
- `Saved new game to database: [name]` - New game registered
- `Generated auto rule file for [name] at [path]` - Rule file created
- `Discovery summary: X new games saved, Y new rule files generated` - Final summary

Check logs to monitor discovery activity and troubleshoot issues.

