# MongoDB Migration Guide

This guide describes the MongoDB schema and indexes used by Achievo.

## Collections

### games
Stores game metadata and detection information.

**Schema:**
```javascript
{
  _id: String,              // Game ID
  name: String,              // Game name
  type: String,              // "steam" or "non_steam"
  steam_app_id: String,      // Steam App ID (optional)
  executable_path: String,   // Full path to executable
  process_name: String,      // Process name
  install_directory: String, // Installation directory
  window_title: String,     // Window title (optional)
  first_detected: Date,      // First detection timestamp
  last_played: Date,         // Last played timestamp
  total_playtime: Number     // Total playtime in seconds
}
```

**Indexes:**
- `steam_app_id` (sparse) - For quick Steam game lookups

### sessions
Stores game playing sessions.

**Schema:**
```javascript
{
  _id: String,        // Session ID
  game_id: String,    // Reference to game
  process_id: Number, // Process ID
  start_time: Date,   // Session start
  end_time: Date,     // Session end (optional)
  duration: Number,   // Duration in seconds
  is_active: Boolean  // Whether session is active
}
```

**Indexes:**
- `game_id` + `start_time` (descending) - For session queries

### achievements
Stores achievement definitions.

**Schema:**
```javascript
{
  _id: String,           // Achievement ID
  game_id: String,       // Reference to game
  steam_app_id: String,  // Steam App ID (optional)
  name: String,          // Achievement name
  description: String,   // Achievement description
  icon_url: String,      // Icon URL (optional)
  icon_locked_url: String, // Locked icon URL (optional)
  is_hidden: Boolean,    // Whether achievement is hidden
  unlock_percent: Number, // Unlock percentage (optional)
  fetched_at: Date       // When definition was fetched
}
```

**Indexes:**
- `game_id` - For game achievement queries
- `steam_app_id` - For Steam achievement queries

### achievement_progress
Stores user's achievement progress.

**Schema:**
```javascript
{
  _id: String,          // Progress ID (game_id-achievement_id)
  achievement_id: String, // Reference to achievement
  game_id: String,      // Reference to game
  unlocked: Boolean,    // Whether unlocked
  unlocked_at: Date,    // Unlock timestamp (optional)
  progress: Number,     // Progress 0.0 to 1.0
  last_updated: Date    // Last update timestamp
}
```

**Indexes:**
- `game_id` + `achievement_id` (unique) - For progress lookups

### statistics
Stores gameplay statistics.

**Schema:**
```javascript
{
  _id: String,        // Stat ID (game_id-stat_name)
  game_id: String,    // Reference to game
  name: String,       // Stat name
  value: Number,      // Numeric value (optional)
  value_int: Number,  // Integer value (optional)
  value_string: String, // String value (optional)
  last_updated: Date // Last update timestamp
}
```

**Indexes:**
- `game_id` + `name` - For stat lookups

### events
Stores system events (achievement unlocks, stat updates, etc.).

**Schema:**
```javascript
{
  _id: String,        // Event ID
  type: String,       // Event type
  game_id: String,    // Reference to game
  timestamp: Date,    // Event timestamp
  payload: Object,   // Event-specific data
  synced: Boolean,   // Whether synced to backend
  synced_at: Date     // Sync timestamp (optional)
}
```

**Indexes:**
- `game_id` + `timestamp` (descending) - For event queries
- `synced` + `timestamp` - For unsynced event queries

## Index Creation

Indexes are automatically created when the Storage instance is initialized. The `createIndexes()` method in `internal/storage/storage.go` handles this.

## Migration Steps

### Initial Setup
1. Ensure MongoDB is running
2. Start Achievo service - indexes will be created automatically
3. Verify indexes:
   ```javascript
   use achievo
   db.games.getIndexes()
   db.sessions.getIndexes()
   db.achievements.getIndexes()
   db.achievement_progress.getIndexes()
   db.events.getIndexes()
   ```

### Adding New Indexes
If you need to add new indexes:

1. Add index creation code to `internal/storage/storage.go` in `createIndexes()`
2. Restart the service
3. Indexes will be created on startup

### Data Migration
If you need to migrate existing data:

1. Export data:
   ```bash
   mongoexport --db achievo --collection games --out games.json
   ```

2. Transform data (if needed)

3. Import data:
   ```bash
   mongoimport --db achievo --collection games --file games.json
   ```

## Recommended Indexes

All recommended indexes are created automatically. Additional indexes you might want:

- `events.timestamp` (single field) - For time-range queries
- `games.last_played` (descending) - For recent games
- `sessions.duration` - For playtime analysis

## Performance Considerations

- Use compound indexes for common query patterns
- Sparse indexes for optional fields (like `steam_app_id`)
- Monitor index usage with `db.collection.aggregate([{$indexStats: {}}])`
- Consider TTL indexes for old events if needed

## Backup and Restore

### Backup
```bash
mongodump --db achievo --out /backup/achievo
```

### Restore
```bash
mongorestore --db achievo /backup/achievo/achievo
```

