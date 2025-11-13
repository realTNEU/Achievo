# Game Rule File Format

Game rule files define how to detect games and unlock achievements. They can be written in YAML or JSON format.

## File Structure

```yaml
game_id: "unique-game-identifier"
game_name: "Display Name"

detection:
  process_name: "game.exe"
  window_title: "Game Window Title"  # optional
  executable: "C:\\Path\\To\\game.exe"  # optional

achievements:
  - id: "achievement-id"
    name: "Achievement Name"
    rules:
      - id: "rule-id"
        type: "rule_type"
        config:
          # rule-specific configuration
```

## Detection Configuration

The `detection` section defines how to identify the game:

- `process_name` (required): The executable name of the game process
- `window_title` (optional): The window title of the game
- `executable` (optional): Full path to the game executable

## Achievement Rules

Each achievement can have multiple rules. **All rules must pass** for the achievement to unlock (AND logic).

### Rule Types

#### 1. File Watch Rule

Watches for file system events.

```yaml
- id: "rule-1"
  type: "file_watch"
  config:
    path: "saves/achievement.dat"  # relative to game directory
    condition: "exists"  # exists, modified, contains
    content: "pattern"  # required if condition is "contains"
```

**Conditions**:
- `exists`: File exists
- `modified`: File was modified recently (within last minute)
- `contains`: File content matches a regex pattern (requires `content` field)

#### 2. Log Pattern Rule

Matches patterns in log files.

```yaml
- id: "rule-2"
  type: "log_pattern"
  config:
    pattern: "Achievement.*unlocked"  # regex pattern
    counter_id: "counter-name"  # optional: increment counter on match
```

**Features**:
- Uses regex for pattern matching
- Can extract named groups from matches
- Can increment counters when matched

#### 3. Counter Rule

Tracks incremental values.

```yaml
- id: "rule-3"
  type: "counter"
  config:
    counter_id: "kills"  # counter identifier
    threshold: 100  # value to reach
```

**Usage**:
- Counters are incremented by log pattern rules
- Counter persists during game session
- Achievement unlocks when counter reaches threshold

#### 4. Memory Signature Rule

Scans process memory for patterns (placeholder - not yet implemented).

```yaml
- id: "rule-4"
  type: "memory_signature"
  config:
    pattern: "48 89 5C 24 ?? ?? 00"  # hex pattern with ?? as wildcard
    offset: 0  # optional memory offset
```

## Examples

### Simple File Watch

```yaml
achievements:
  - id: "ach-save-game"
    name: "Progress Saved"
    rules:
      - id: "rule-save"
        type: "file_watch"
        config:
          path: "saves/autosave.sav"
          condition: "modified"
```

### Counter-Based Achievement

```yaml
achievements:
  - id: "ach-100-kills"
    name: "100 Kills"
    rules:
      - id: "rule-counter"
        type: "counter"
        config:
          counter_id: "kills"
          threshold: 100
      - id: "rule-increment"
        type: "log_pattern"
        config:
          pattern: "Killed.*enemy"
          counter_id: "kills"
```

### Multiple Conditions

```yaml
achievements:
  - id: "ach-complete"
    name: "Game Complete"
    rules:
      - id: "rule-final-boss"
        type: "log_pattern"
        config:
          pattern: "Defeated.*Final Boss"
      - id: "rule-save-file"
        type: "file_watch"
        config:
          path: "saves/final.sav"
          condition: "exists"
```

### Log Pattern with Extraction

```yaml
achievements:
  - id: "ach-level-up"
    name: "Level Up"
    rules:
      - id: "rule-level"
        type: "log_pattern"
        config:
          pattern: "Level up to (?P<level>\\d+)"
          # Extracted groups available in rule evaluation context
```

## Best Practices

1. **Use descriptive IDs**: Make rule and achievement IDs unique and meaningful
2. **Test patterns**: Verify regex patterns work correctly
3. **Relative paths**: Use paths relative to game installation directory
4. **Counter naming**: Use clear, unique counter names
5. **Multiple rules**: Combine different rule types for complex achievements
6. **Error handling**: Rules should be robust to missing files or invalid data

## Rule Evaluation

Rules are evaluated:
- When file events occur (file watch rules)
- When log lines are processed (log pattern rules)
- Periodically (counter rules, memory signature rules)
- On demand (manual evaluation)

All rules for an achievement must pass simultaneously for the achievement to unlock.

