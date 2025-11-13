package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Scanner monitors files and parses logs
type Scanner struct {
	watchers map[string]*fsnotify.Watcher
	watches  map[string]*WatchConfig
	mu       sync.RWMutex
	events   chan FileEvent
}

// WatchConfig represents a watch configuration
type WatchConfig struct {
	Path      string
	Patterns  []string
	Recursive bool
}

// FileEvent represents a file system event
type FileEvent struct {
	Type      string // create, write, remove, rename
	Path      string
	Timestamp time.Time
}

// LogPattern represents a log parsing pattern
type LogPattern struct {
	Pattern *regexp.Regexp
	Extract []string // named groups to extract
}

// Match represents a log pattern match
type Match struct {
	Pattern   string
	Groups    map[string]string
	Line      string
	Timestamp time.Time
}

// New creates a new Scanner
func New() *Scanner {
	return &Scanner{
		watchers: make(map[string]*fsnotify.Watcher),
		watches:  make(map[string]*WatchConfig),
		events:   make(chan FileEvent, 100),
	}
}

// WatchDirectory watches a directory for changes
func (s *Scanner) WatchDirectory(path string, patterns []string, recursive bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already watching
	if _, exists := s.watches[path]; exists {
		return fmt.Errorf("already watching path: %s", path)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Add the directory
	if err := watcher.Add(path); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to add path to watcher: %w", err)
	}

	// If recursive, add subdirectories
	if recursive {
		if err := s.addRecursive(watcher, path); err != nil {
			watcher.Close()
			return fmt.Errorf("failed to add recursive paths: %w", err)
		}
	}

	s.watchers[path] = watcher
	s.watches[path] = &WatchConfig{
		Path:      path,
		Patterns:  patterns,
		Recursive: recursive,
	}

	// Start watching goroutine
	go s.watchLoop(path, watcher)

	return nil
}

// addRecursive adds all subdirectories recursively
func (s *Scanner) addRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}

// watchLoop handles file system events
func (s *Scanner) watchLoop(path string, watcher *fsnotify.Watcher) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Check if file matches any pattern
			s.mu.RLock()
			config, exists := s.watches[path]
			s.mu.RUnlock()

			if !exists {
				continue
			}

			// Check patterns
			matched := len(config.Patterns) == 0
			if !matched {
				for _, pattern := range config.Patterns {
					matched, _ := filepath.Match(pattern, filepath.Base(event.Name))
					if matched {
						break
					}
				}
			}

			if matched {
				fileEvent := FileEvent{
					Type:      s.mapEventType(event.Op),
					Path:      event.Name,
					Timestamp: time.Now(),
				}

				select {
				case s.events <- fileEvent:
				default:
					// Channel full, skip event
				}
			}

			// If it's a new directory and recursive, add it
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					s.mu.RLock()
					config, exists := s.watches[path]
					s.mu.RUnlock()
					if exists && config.Recursive {
						watcher.Add(event.Name)
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watcher error for %s: %v\n", path, err)
		}
	}
}

// mapEventType maps fsnotify operations to our event types
func (s *Scanner) mapEventType(op fsnotify.Op) string {
	if op&fsnotify.Create == fsnotify.Create {
		return "create"
	}
	if op&fsnotify.Write == fsnotify.Write {
		return "write"
	}
	if op&fsnotify.Remove == fsnotify.Remove {
		return "remove"
	}
	if op&fsnotify.Rename == fsnotify.Rename {
		return "rename"
	}
	return "unknown"
}

// StopWatching stops watching a path
func (s *Scanner) StopWatching(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	watcher, exists := s.watchers[path]
	if !exists {
		return fmt.Errorf("not watching path: %s", path)
	}

	if err := watcher.Close(); err != nil {
		return err
	}

	delete(s.watchers, path)
	delete(s.watches, path)

	return nil
}

// GetEvents returns the event channel
func (s *Scanner) GetEvents() <-chan FileEvent {
	return s.events
}

// ParseLogFile parses a log file with given patterns
func (s *Scanner) ParseLogFile(path string, patterns []LogPattern) ([]Match, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	var matches []Match
	lines := regexp.MustCompile(`\r?\n`).Split(string(data), -1)

	for lineNum, line := range lines {
		for _, pattern := range patterns {
			if pattern.Pattern.MatchString(line) {
				match := Match{
					Pattern:   pattern.Pattern.String(),
					Groups:    make(map[string]string),
					Line:      line,
					Timestamp: time.Now(),
				}

				// Extract named groups
				submatches := pattern.Pattern.FindStringSubmatch(line)
				names := pattern.Pattern.SubexpNames()
				for i, name := range names {
					if i > 0 && name != "" && i < len(submatches) {
						match.Groups[name] = submatches[i]
					}
				}

				matches = append(matches, match)
			}
		}

		// Limit to prevent excessive memory usage
		if lineNum > 10000 {
			break
		}
	}

	return matches, nil
}

// ParseLogFileTail parses only the tail of a log file (last N lines)
func (s *Scanner) ParseLogFileTail(path string, patterns []LogPattern, lines int) ([]Match, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	allLines := regexp.MustCompile(`\r?\n`).Split(string(data), -1)
	start := len(allLines) - lines
	if start < 0 {
		start = 0
	}

	var matches []Match
	for i := start; i < len(allLines); i++ {
		line := allLines[i]
		for _, pattern := range patterns {
			if pattern.Pattern.MatchString(line) {
				match := Match{
					Pattern:   pattern.Pattern.String(),
					Groups:    make(map[string]string),
					Line:      line,
					Timestamp: time.Now(),
				}

				submatches := pattern.Pattern.FindStringSubmatch(line)
				names := pattern.Pattern.SubexpNames()
				for j, name := range names {
					if j > 0 && name != "" && j < len(submatches) {
						match.Groups[name] = submatches[j]
					}
				}

				matches = append(matches, match)
			}
		}
	}

	return matches, nil
}

// Close closes all watchers
func (s *Scanner) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lastErr error
	for path, watcher := range s.watchers {
		if err := watcher.Close(); err != nil {
			lastErr = err
		}
		delete(s.watchers, path)
		delete(s.watches, path)
	}

	close(s.events)
	return lastErr
}
