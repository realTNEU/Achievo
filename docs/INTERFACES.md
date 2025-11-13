# Interface Documentation

This document describes the key interfaces and their contracts in the Achievo system.

## Detector Interface

```go
type Detector interface {
    Detect() ([]*Game, error)
    RegisterGame(game *Game) error
    IsGameRunning(gameID string) bool
    GetProcessID(gameID string) (int, error)
}
```

**Purpose**: Identifies running games by scanning processes.

**Methods**:
- `Detect()`: Scans all running processes and returns detected games
- `RegisterGame()`: Registers a known game for detection
- `IsGameRunning()`: Checks if a specific game is currently running
- `GetProcessID()`: Gets the process ID for a running game

## Monitor Interface

```go
type Monitor interface {
    StartMonitoring(gameID string, processID int) error
    StopMonitoring(gameID string) (*Session, error)
    GetSession(gameID string) (*Session, error)
    UpdateSession(gameID string) error
    CheckProcesses() []string
}
```

**Purpose**: Tracks game session lifecycle and playtime.

**Methods**:
- `StartMonitoring()`: Begins monitoring a game session
- `StopMonitoring()`: Stops monitoring and returns final session data
- `GetSession()`: Retrieves current session information
- `UpdateSession()`: Updates session duration
- `CheckProcesses()`: Verifies monitored processes are still running

## SteamClient Interface

```go
type SteamClient interface {
    FetchAchievements(appID string) ([]Achievement, error)
    GetAchievement(appID, achievementID string) (*Achievement, error)
    CacheAchievements(appID string, achievements []Achievement) error
}
```

**Purpose**: Fetches and caches Steam achievement definitions.

**Methods**:
- `FetchAchievements()`: Fetches all achievement definitions for a Steam App ID
- `GetAchievement()`: Retrieves a specific achievement definition
- `CacheAchievements()`: Caches achievement definitions locally

## RuleEngine Interface

```go
type RuleEngine interface {
    LoadRules(filePath string) error
    Evaluate(gameID string, context *EvaluationContext) ([]string, error)
    GetRules(gameID string) (*GameRules, error)
    GetCounter(gameID, counterID string) (*Counter, error)
}
```

**Purpose**: Evaluates game-specific achievement unlock conditions.

**Methods**:
- `LoadRules()`: Loads rules from a JSON/YAML file
- `Evaluate()`: Evaluates all rules for a game given context
- `GetRules()`: Retrieves rules for a game
- `GetCounter()`: Gets a counter value for a game

## Rule Interface

```go
type Rule interface {
    Evaluate(context *EvaluationContext) (bool, error)
    GetID() string
}
```

**Purpose**: Represents a single achievement unlock condition.

**Methods**:
- `Evaluate()`: Evaluates the rule against the context
- `GetID()`: Returns the rule's unique identifier

## Scanner Interface

```go
type Scanner interface {
    WatchDirectory(path string, patterns []string, recursive bool) error
    StopWatching(path string) error
    GetEvents() <-chan FileEvent
    ParseLogFile(path string, patterns []LogPattern) ([]Match, error)
    ParseLogFileTail(path string, patterns []LogPattern, lines int) ([]Match, error)
    Close() error
}
```

**Purpose**: Monitors file system changes and parses log files.

**Methods**:
- `WatchDirectory()`: Starts watching a directory for changes
- `StopWatching()`: Stops watching a directory
- `GetEvents()`: Returns a channel of file system events
- `ParseLogFile()`: Parses a log file with regex patterns
- `ParseLogFileTail()`: Parses the tail of a log file
- `Close()`: Closes all watchers

## Tracker Interface

```go
type Tracker interface {
    TrackEvent(event Event) error
    CheckAchievement(gameID, achievementID string) (*AchievementProgress, error)
    UnlockAchievement(gameID, achievementID string) error
    UpdateProgress(gameID, achievementID string, progress float64) error
    GetProgress(gameID string) ([]AchievementProgress, error)
    TrackStatUpdate(gameID, statName string, value interface{}) error
}
```

**Purpose**: Tracks achievement progress and emits events.

**Methods**:
- `TrackEvent()`: Processes an event and updates tracking
- `CheckAchievement()`: Checks current progress on an achievement
- `UnlockAchievement()`: Unlocks an achievement
- `UpdateProgress()`: Updates progress for incremental achievements
- `GetProgress()`: Gets all achievement progress for a game
- `TrackStatUpdate()`: Tracks a statistic update

## Storage Interface

```go
type Storage interface {
    SaveGame(game *Game) error
    GetGame(gameID string) (*Game, error)
    GetGameBySteamAppID(appID string) (*Game, error)
    SaveSession(session *Session) error
    GetActiveSession(gameID string) (*Session, error)
    SaveAchievement(achievement *Achievement) error
    GetAchievementsByGame(gameID string) ([]Achievement, error)
    SaveAchievementProgress(progress *AchievementProgress) error
    GetAchievementProgress(gameID, achievementID string) (*AchievementProgress, error)
    GetAllProgress(gameID string) ([]AchievementProgress, error)
    SaveStatistic(stat *Statistic) error
    GetStatistic(gameID, statName string) (*Statistic, error)
    SaveEvent(event *Event) error
    GetUnsyncedEvents(limit int) ([]Event, error)
    MarkEventSynced(eventID string) error
    Close() error
}
```

**Purpose**: Provides persistent storage for all data models.

**Methods**: Comprehensive CRUD operations for all data models, plus event queue management.

## EvaluationContext

```go
type EvaluationContext struct {
    GameID      string
    FilePath    string
    FileContent string
    LogLine     string
    ProcessID   int
    Timestamp   time.Time
    Counters    map[string]*Counter
}
```

**Purpose**: Provides context for rule evaluation.

**Fields**:
- `GameID`: The game being evaluated
- `FilePath`: Path to a file that triggered evaluation
- `FileContent`: Content of a file (if applicable)
- `LogLine`: A log line that triggered evaluation
- `ProcessID`: Process ID of the running game
- `Timestamp`: When the evaluation is happening
- `Counters`: Map of counter values for the game

