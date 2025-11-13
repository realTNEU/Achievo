# Achievo Architecture Documentation

## System Design

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Achievo Background Service                │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Detector   │  │   Monitor    │  │   Scanner    │      │
│  │              │  │              │  │              │      │
│  │ - Process    │  │ - Lifecycle  │  │ - File Watch │      │
│  │   Detection  │  │ - Playtime   │  │ - Log Parse  │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                 │                  │              │
│         └─────────────────┼──────────────────┘             │
│                            │                                 │
│                   ┌────────▼────────┐                        │
│                   │   Rule Engine   │                        │
│                   │                 │                        │
│                   │ - Evaluate     │                        │
│                   │ - Match Rules  │                        │
│                   └────────┬────────┘                        │
│                            │                                 │
│                   ┌────────▼────────┐                        │
│                   │     Tracker      │                        │
│                   │                  │                        │
│                   │ - Map Events    │                        │
│                   │ - Track Progress│                        │
│                   └────────┬────────┘                        │
│                            │                                 │
│  ┌──────────────┐  ┌──────▼───────┐  ┌──────────────┐      │
│  │    Steam     │  │   Storage     │  │    Config    │      │
│  │              │  │               │  │              │      │
│  │ - Fetch Defs │  │ - MongoDB     │  │ - Load Rules │      │
│  │ - Cache      │  │ - Cache       │  │ - Settings   │      │
│  └──────────────┘  └───────────────┘  └──────────────┘      │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## Module Responsibilities

### 1. Detector (`internal/detector`)
**Purpose**: Identify when games are running

**Responsibilities**:
- Scan running processes periodically
- Match processes against known game signatures
- Register game launch events
- Maintain game registry

**Interfaces**:
```go
type Detector interface {
    Detect() ([]Game, error)
    RegisterGame(game Game) error
    IsGameRunning(gameID string) bool
}
```

### 2. Monitor (`internal/monitor`)
**Purpose**: Track game session lifecycle

**Responsibilities**:
- Monitor process start/stop events
- Calculate playtime
- Track session duration
- Emit session events

**Interfaces**:
```go
type Monitor interface {
    StartMonitoring(gameID string, processID int) error
    StopMonitoring(gameID string) error
    GetSession(gameID string) (*Session, error)
}
```

### 3. Steam (`internal/steam`)
**Purpose**: Fetch and cache Steam achievement definitions

**Responsibilities**:
- Query Steam Web API for achievement definitions
- Cache definitions locally (MongoDB)
- Provide achievement metadata
- Handle API rate limiting

**Interfaces**:
```go
type SteamClient interface {
    FetchAchievements(appID string) ([]Achievement, error)
    GetAchievement(appID, achievementID string) (*Achievement, error)
    CacheAchievements(appID string, achievements []Achievement) error
}
```

### 4. Rules (`internal/rules`)
**Purpose**: Evaluate game-specific achievement unlock conditions

**Responsibilities**:
- Load rule files (JSON/YAML)
- Evaluate file watch rules
- Match log patterns
- Check counter conditions
- Trigger memory scans

**Interfaces**:
```go
type RuleEngine interface {
    LoadRules(gameID string) error
    Evaluate(gameID string, context *EvaluationContext) ([]Event, error)
    RegisterRule(gameID string, rule Rule) error
}

type Rule interface {
    Evaluate(context *EvaluationContext) (bool, error)
    GetID() string
}
```

### 5. Scanner (`internal/scanner`)
**Purpose**: Monitor files and parse logs

**Responsibilities**:
- Watch game directories for changes
- Parse log files with regex
- Extract relevant data
- Emit file change events

**Interfaces**:
```go
type Scanner interface {
    WatchDirectory(path string, patterns []string) error
    ParseLogFile(path string, patterns []LogPattern) ([]Match, error)
    StopWatching(path string) error
}
```

### 6. Tracker (`internal/tracker`)
**Purpose**: Map events to achievements and track progress

**Responsibilities**:
- Map detected events to achievement definitions
- Track user achievement progress
- Emit unlock events
- Handle incremental achievements

**Interfaces**:
```go
type Tracker interface {
    TrackEvent(event Event) error
    CheckAchievement(gameID, achievementID string) (*Progress, error)
    UnlockAchievement(gameID, achievementID string) error
    GetProgress(gameID string) ([]Progress, error)
}
```

### 7. Storage (`internal/storage`)
**Purpose**: Persistent data storage and caching

**Responsibilities**:
- MongoDB connection management
- CRUD operations for all models
- Event queue management
- Cache management

**Interfaces**:
```go
type Storage interface {
    SaveSession(session *Session) error
    SaveAchievementProgress(progress *AchievementProgress) error
    SaveEvent(event *Event) error
    GetUnsyncedEvents() ([]Event, error)
    MarkEventSynced(eventID string) error
}
```

## Data Flow

### Game Launch Flow
1. Detector identifies new game process
2. Monitor creates new session
3. Rule Engine loads game-specific rules
4. Scanner starts watching game files
5. Tracker initializes achievement tracking

### Achievement Detection Flow
1. Scanner detects file change or log match
2. Rule Engine evaluates rules with new context
3. Rule matches achievement condition
4. Tracker checks if achievement already unlocked
5. If new, Tracker creates unlock event
6. Storage saves event to queue
7. Storage updates achievement progress

### Steam Integration Flow
1. Detector identifies Steam game (by App ID)
2. Tracker checks if achievements cached locally
3. If not cached, Steam client fetches from API
4. Steam client caches definitions in MongoDB
5. Tracker uses cached definitions for tracking

## Rule Engine Design

### Rule Types

1. **File Watch Rule**
   - Watches specific file paths
   - Triggers on file creation/modification
   - Can check file content or metadata

2. **Log Pattern Rule**
   - Monitors log files
   - Uses regex to match patterns
   - Extracts values from matches

3. **Counter Rule**
   - Tracks incremental values
   - Compares against thresholds
   - Supports multiple counters per achievement

4. **Memory Signature Rule**
   - Scans process memory (read-only)
   - Matches byte patterns
   - Extracts values from memory offsets

### Rule File Format

```yaml
game:
  id: "game-123"
  name: "Example Game"
  executable: "game.exe"
  
detection:
  process_name: "game.exe"
  window_title: "Example Game"
  
achievements:
  - id: "ach-001"
    name: "First Steps"
    rules:
      - type: "file_watch"
        path: "saves/achievement.dat"
        condition: "exists"
      - type: "log_pattern"
        file: "logs/game.log"
        pattern: "Achievement.*First Steps"
        
  - id: "ach-002"
    name: "100 Kills"
    rules:
      - type: "counter"
        source: "log_pattern"
        pattern: "Killed.*enemy"
        threshold: 100
```

## Security Considerations

- Memory scanning is read-only, non-invasive
- No code injection or process modification
- API keys stored securely in configuration
- Database credentials encrypted at rest
- Event queue can be synced securely in future

## Extensibility

- Plugin system for custom rule types
- Interface-based design allows easy swapping
- Modular architecture supports new platforms
- Rule files enable community contributions
- Event system allows future integrations

