package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"achievo/internal/config"
	"achievo/internal/discovery"
	"achievo/internal/detector"
	"achievo/internal/models"
	"achievo/internal/monitor"
	"achievo/internal/rules"
	"achievo/internal/scanner"
	"achievo/internal/steam"
	"achievo/internal/storage"
	"achievo/internal/tracker"
)

// Service is the main orchestrator
type Service struct {
	config        *config.Config
	storage       *storage.Storage
	detector      *detector.Detector
	monitor       *monitor.Monitor
	steam         *steam.SteamClient
	ruleEngine    *rules.RuleEngine
	scanner       *scanner.Scanner
	tracker       *tracker.Tracker
	discoverer    *discovery.Discoverer
	ruleGenerator *rules.RuleGenerator
	running       bool
	activeGames   map[string]*GameContext
	mu            sync.RWMutex
	stopChan      chan struct{}
	startTime     time.Time
}

// GameContext tracks context for an active game
type GameContext struct {
	Game      *models.Game
	Session   *models.Session
	Rules     *rules.GameRules
	Watches   []string
}

// New creates a new Service
func New(cfg *config.Config) (*Service, error) {
	// Initialize storage
	stor, err := storage.New(cfg.MongoDB.URI, cfg.MongoDB.Database, cfg.MongoDB.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize components
	det := detector.New()
	mon := monitor.New()
	steamClient := steam.New(cfg.Steam.APIKey, stor)
	ruleEng := rules.New()
	scn := scanner.New()
	trk := tracker.New(stor)
	disc := discovery.New()
	ruleGen := rules.NewRuleGenerator(cfg.Paths.GameRules)

	service := &Service{
		config:        cfg,
		storage:       stor,
		detector:      det,
		monitor:       mon,
		steam:         steamClient,
		ruleEngine:    ruleEng,
		scanner:       scn,
		tracker:       trk,
		discoverer:    disc,
		ruleGenerator: ruleGen,
		activeGames:   make(map[string]*GameContext),
		stopChan:      make(chan struct{}),
		startTime:     time.Now(),
	}

	return service, nil
}

// Start starts the service
func (s *Service) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("service already running")
	}
	s.running = true
	s.mu.Unlock()

	log.Println("Starting Achievo service...")

	// Perform initial game discovery
	log.Println("Discovering games on mounted drives...")
	if err := s.discoverGames(); err != nil {
		log.Printf("Warning: game discovery failed: %v", err)
	}

	// Load game rules
	if err := s.loadGameRules(); err != nil {
		log.Printf("Warning: failed to load some game rules: %v", err)
	}

	// Register known games from storage
	if err := s.loadKnownGames(); err != nil {
		log.Printf("Warning: failed to load known games: %v", err)
	}

	// Start detection loop
	go s.detectionLoop()

	// Start file event processing
	go s.fileEventLoop()

	// Start rule evaluation loop
	go s.ruleEvaluationLoop()

	// Start session monitoring
	go s.sessionMonitoringLoop()

	// Start heartbeat logging
	go s.heartbeatLoop()

	// Start periodic discovery if enabled
	if s.config.Detection.DiscoveryEnabled && s.config.Detection.DiscoveryInterval > 0 {
		go s.periodicDiscoveryLoop()
	}

	log.Println("Achievo service started")
	return nil
}

// Stop stops the service
func (s *Service) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()

	log.Println("Stopping Achievo service...")

	// Stop all active monitoring
	s.mu.Lock()
	for gameID, ctx := range s.activeGames {
		s.stopGameMonitoring(gameID, ctx)
	}
	s.mu.Unlock()

	// Close scanner
	if err := s.scanner.Close(); err != nil {
		log.Printf("Error closing scanner: %v", err)
	}

	// Close storage
	if err := s.storage.Close(); err != nil {
		log.Printf("Error closing storage: %v", err)
	}

	log.Println("Achievo service stopped")
	return nil
}

// loadGameRules loads all game rule files (both manual and auto-generated)
func (s *Service) loadGameRules() error {
	rulesDir := s.config.Paths.GameRules
	if rulesDir == "" {
		return nil
	}

	// Load manual rule files first
	manualCount := 0
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, try to create it
			if err := os.MkdirAll(rulesDir, 0755); err != nil {
				return fmt.Errorf("failed to create rules directory: %w", err)
			}
			entries = []os.DirEntry{}
		} else {
			return fmt.Errorf("failed to read rules directory: %w", err)
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		path := filepath.Join(rulesDir, entry.Name())
		if err := s.ruleEngine.LoadRules(path); err != nil {
			log.Printf("Error: Failed to load manual rules from %s: %v", path, err)
			continue
		}

		log.Printf("Loaded manual rules from %s", path)
		manualCount++
	}

	// Load auto-generated rule files
	autoDir := filepath.Join(rulesDir, "auto")
	autoEntries, err := os.ReadDir(autoDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Auto directory doesn't exist yet, that's okay
			log.Printf("No auto-generated rules directory found at %s", autoDir)
		} else {
			log.Printf("Warning: Failed to read auto rules directory: %v", err)
		}
	} else {
		autoCount := 0
		for _, entry := range autoEntries {
			if entry.IsDir() {
				continue
			}

			ext := filepath.Ext(entry.Name())
			if ext != ".yaml" && ext != ".yml" && ext != ".json" {
				continue
			}

			path := filepath.Join(autoDir, entry.Name())
			if err := s.ruleEngine.LoadRules(path); err != nil {
				log.Printf("Error: Failed to load auto-generated rules from %s: %v", path, err)
				continue
			}

			log.Printf("Loaded auto-generated rules from %s", path)
			autoCount++
		}
		log.Printf("Loaded %d manual rule files and %d auto-generated rule files", manualCount, autoCount)
	}

	return nil
}

// loadKnownGames loads known games from storage
func (s *Service) loadKnownGames() error {
	// Load games from rule files and register them
	rulesDir := s.config.Paths.GameRules
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		path := filepath.Join(rulesDir, entry.Name())
		// Load rules first
		if err := s.ruleEngine.LoadRules(path); err != nil {
			continue
		}
		
		// Rules are loaded, games will be registered when they're detected
	}

	return nil
}

// discoverGames discovers games on mounted drives and generates rule files
func (s *Service) discoverGames() error {
	if !s.config.Detection.DiscoveryEnabled {
		log.Println("Game discovery is disabled in configuration")
		return nil
	}

	log.Println("Starting game discovery scan...")
	startTime := time.Now()

	discovered, err := s.discoverer.DiscoverGames()
	if err != nil {
		return fmt.Errorf("game discovery failed: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("Discovery scan completed in %v, found %d potential games", duration, len(discovered))

	newGamesCount := 0
	newRulesCount := 0

	for _, discGame := range discovered {
		// Convert to Game model
		game := discGame.ToGame()

		// Check if game already exists in storage
		existing, err := s.storage.GetGame(game.ID)
		if err == nil && existing != nil {
			// Game already known, skip
			continue
		}

		// Save game to storage
		if err := s.storage.SaveGame(game); err != nil {
			log.Printf("Error: Failed to save discovered game %s to MongoDB: %v", game.Name, err)
			continue
		}
		newGamesCount++
		log.Printf("Saved new game to database: %s (ID: %s, Type: %s)", game.Name, game.ID, game.Type)

		// Check if manual rule file exists
		manualPath := s.ruleGenerator.GetManualRuleFilePath(game)
		if _, err := os.Stat(manualPath); err == nil {
			// Manual rule file exists, skip auto-generation
			log.Printf("Manual rule file exists for %s, skipping auto-generation", game.Name)
			s.detector.RegisterGame(game)
			continue
		}

		// Generate auto rule file
		autoPath := s.ruleGenerator.GetAutoRuleFilePath(game)
		if _, err := os.Stat(autoPath); os.IsNotExist(err) {
			// Generate rule file
			if err := s.ruleGenerator.GenerateRuleFile(game); err != nil {
				log.Printf("Error: Failed to generate rule file for %s: %v", game.Name, err)
				continue
			}
			newRulesCount++
			log.Printf("Generated auto rule file for %s at %s", game.Name, autoPath)

			// Load the generated rules
			if err := s.ruleEngine.LoadRules(autoPath); err != nil {
				log.Printf("Error: Failed to load generated rules for %s: %v", game.Name, err)
			} else {
				log.Printf("Successfully loaded auto-generated rules for %s", game.Name)
			}
		}

		// Register game with detector
		s.detector.RegisterGame(game)

		// If Steam game, fetch schema
		if game.Type == models.GameTypeSteam && game.SteamAppID != "" {
			go func(appID, gameName string) {
				log.Printf("Fetching Steam schema for %s (App ID: %s)...", gameName, appID)
				if err := s.steam.FetchAndStoreSchema(appID); err != nil {
					log.Printf("Error: Failed to fetch Steam schema for %s: %v", appID, err)
				} else {
					log.Printf("Successfully fetched Steam schema for %s", gameName)
				}
			}(game.SteamAppID, game.Name)
		}
	}

	log.Printf("Discovery summary: %d new games saved, %d new rule files generated", newGamesCount, newRulesCount)
	return nil
}

// periodicDiscoveryLoop runs periodic discovery scans
func (s *Service) periodicDiscoveryLoop() {
	if s.config.Detection.DiscoveryInterval <= 0 {
		return
	}

	interval := time.Duration(s.config.Detection.DiscoveryInterval) * time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Periodic discovery enabled: scanning every %v", interval)

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			log.Println("Starting periodic game discovery scan...")
			if err := s.discoverGames(); err != nil {
				log.Printf("Error: Periodic discovery failed: %v", err)
			}
		}
	}
}

// heartbeatLoop logs periodic heartbeat messages
func (s *Service) heartbeatLoop() {
	ticker := time.NewTicker(60 * time.Second) // Heartbeat every minute
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.mu.RLock()
			activeCount := len(s.activeGames)
			uptime := time.Since(s.startTime)
			s.mu.RUnlock()

			log.Printf("[HEARTBEAT] Service running | Active games: %d | Uptime: %s",
				activeCount,
				uptime.Round(time.Second))
		}
	}
}

// detectionLoop continuously detects running games
func (s *Service) detectionLoop() {
	ticker := time.NewTicker(time.Duration(s.config.Detection.ScanInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.detectGames()
		}
	}
}

// detectGames detects currently running games
func (s *Service) detectGames() {
	detected, err := s.detector.Detect()
	if err != nil {
		log.Printf("Error detecting games: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for new games
	for _, game := range detected {
		if _, exists := s.activeGames[game.ID]; !exists {
			// New game detected
			s.startGameMonitoring(game)
		}
	}

	// Check for stopped games
	for gameID, ctx := range s.activeGames {
		if !s.detector.IsGameRunning(gameID) {
			s.stopGameMonitoring(gameID, ctx)
		}
	}
}

// startGameMonitoring starts monitoring a game
func (s *Service) startGameMonitoring(game *models.Game) {
	log.Printf("Starting monitoring for game: %s (%s)", game.Name, game.ID)

	// Get or create process ID
	processID, err := s.detector.GetProcessID(game.ID)
	if err != nil {
		log.Printf("Failed to get process ID for game %s: %v", game.ID, err)
		return
	}

	// Start session monitoring
	if err := s.monitor.StartMonitoring(game.ID, processID); err != nil {
		log.Printf("Failed to start monitoring for game %s: %v", game.ID, err)
		return
	}

	// Get session
	session, err := s.monitor.GetSession(game.ID)
	if err != nil {
		log.Printf("Failed to get session for game %s: %v", game.ID, err)
		return
	}

	// Save game to storage
	game.LastPlayed = time.Now()
	if err := s.storage.SaveGame(game); err != nil {
		log.Printf("Failed to save game %s: %v", game.ID, err)
	}

	// Save session
	if err := s.storage.SaveSession(session); err != nil {
		log.Printf("Failed to save session for game %s: %v", game.ID, err)
	}

	// Load game rules
	gameRules, err := s.ruleEngine.GetRules(game.ID)
	if err != nil {
		log.Printf("No rules found for game %s: %v", game.ID, err)
		gameRules = nil
	}

	// Create game context
	ctx := &GameContext{
		Game:    game,
		Session: session,
		Rules:   gameRules,
		Watches: make([]string, 0),
	}

	// Start file watching if rules exist
	if gameRules != nil && game.InstallDirectory != "" {
		s.startFileWatching(ctx)
	}

	// Fetch Steam achievements if Steam game
	if game.Type == models.GameTypeSteam && game.SteamAppID != "" {
		go s.fetchSteamAchievements(game.SteamAppID)
	}

	// Create game start event
	event := models.Event{
		ID:        fmt.Sprintf("event-%s-start-%d", game.ID, time.Now().Unix()),
		Type:      models.EventTypeGameStart,
		GameID:    game.ID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"game_name": game.Name,
			"process_id": processID,
		},
		Synced: false,
	}
	s.storage.SaveEvent(&event)

	s.activeGames[game.ID] = ctx
}

// stopGameMonitoring stops monitoring a game
func (s *Service) stopGameMonitoring(gameID string, ctx *GameContext) {
	log.Printf("Stopping monitoring for game: %s", gameID)

	// Stop session monitoring
	session, err := s.monitor.StopMonitoring(gameID)
	if err == nil && session != nil {
		// Update game playtime
		game := ctx.Game
		game.TotalPlaytime += session.Duration
		game.LastPlayed = time.Now()
		s.storage.SaveGame(game)

		// Save session
		s.storage.SaveSession(session)
	}

	// Stop file watching
	for _, watchPath := range ctx.Watches {
		s.scanner.StopWatching(watchPath)
	}

	// Create game stop event
	event := models.Event{
		ID:        fmt.Sprintf("event-%s-stop-%d", gameID, time.Now().Unix()),
		Type:      models.EventTypeGameStop,
		GameID:    gameID,
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"game_name": ctx.Game.Name,
		},
		Synced: false,
	}
	s.storage.SaveEvent(&event)

	delete(s.activeGames, gameID)
}

// startFileWatching starts file watching for a game
func (s *Service) startFileWatching(ctx *GameContext) {
	if ctx.Rules == nil || ctx.Game.InstallDirectory == "" {
		return
	}

	// Watch game directory
	patterns := []string{"*.log", "*.txt", "*.dat", "*.save", "*.sav"}
	if err := s.scanner.WatchDirectory(ctx.Game.InstallDirectory, patterns, true); err != nil {
		log.Printf("Failed to watch directory for game %s: %v", ctx.Game.ID, err)
		return
	}

	ctx.Watches = append(ctx.Watches, ctx.Game.InstallDirectory)
	log.Printf("Started watching directory: %s", ctx.Game.InstallDirectory)
}

// fetchSteamAchievements fetches achievements from Steam API
func (s *Service) fetchSteamAchievements(appID string) {
	log.Printf("Fetching Steam achievements for app ID: %s", appID)
	achievements, err := s.steam.FetchAchievements(appID)
	if err != nil {
		log.Printf("Failed to fetch Steam achievements for %s: %v", appID, err)
		return
	}

	log.Printf("Fetched %d achievements for Steam app %s", len(achievements), appID)
}

// fileEventLoop processes file system events
func (s *Service) fileEventLoop() {
	for {
		select {
		case <-s.stopChan:
			return
		case event, ok := <-s.scanner.GetEvents():
			if !ok {
				return
			}
			s.handleFileEvent(event)
		}
	}
}

// handleFileEvent handles a file system event
func (s *Service) handleFileEvent(event scanner.FileEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find which game this event belongs to
	for gameID, ctx := range s.activeGames {
		if ctx.Game.InstallDirectory != "" {
			rel, err := filepath.Rel(ctx.Game.InstallDirectory, event.Path)
			if err == nil && rel != ".." && !filepath.IsAbs(rel) {
				// This event belongs to this game
				s.evaluateRulesForFile(gameID, ctx, event)
				break
			}
		}
	}
}

// evaluateRulesForFile evaluates rules based on a file event
func (s *Service) evaluateRulesForFile(gameID string, ctx *GameContext, event scanner.FileEvent) {
	if ctx.Rules == nil {
		return
	}

	context := &rules.EvaluationContext{
		GameID:    gameID,
		FilePath:  event.Path,
		Timestamp: event.Timestamp,
	}

	// Read file content if needed
	if event.Type == "write" || event.Type == "create" {
		data, err := os.ReadFile(event.Path)
		if err == nil {
			context.FileContent = string(data)
		}
	}

	// Evaluate rules
	unlocked, err := s.ruleEngine.Evaluate(gameID, context)
	if err != nil {
		log.Printf("Error evaluating rules for game %s: %v", gameID, err)
		return
	}

	// Unlock achievements
	for _, achievementID := range unlocked {
		if err := s.tracker.UnlockAchievement(gameID, achievementID); err != nil {
			log.Printf("Failed to unlock achievement %s for game %s: %v", achievementID, gameID, err)
		} else {
			log.Printf("Unlocked achievement %s for game %s", achievementID, gameID)
		}
	}
}

// ruleEvaluationLoop periodically evaluates rules
func (s *Service) ruleEvaluationLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.evaluateAllRules()
		}
	}
}

// evaluateAllRules evaluates rules for all active games
func (s *Service) evaluateAllRules() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for gameID, ctx := range s.activeGames {
		if ctx.Rules == nil {
			continue
		}

		context := &rules.EvaluationContext{
			GameID:    gameID,
			Timestamp: time.Now(),
		}

		unlocked, err := s.ruleEngine.Evaluate(gameID, context)
		if err != nil {
			log.Printf("Error evaluating rules for game %s: %v", gameID, err)
			continue
		}

		for _, achievementID := range unlocked {
			if err := s.tracker.UnlockAchievement(gameID, achievementID); err != nil {
				log.Printf("Failed to unlock achievement %s for game %s: %v", achievementID, gameID, err)
			}
		}
	}
}

// sessionMonitoringLoop monitors active sessions
func (s *Service) sessionMonitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.updateSessions()
		}
	}
}

// updateSessions updates all active sessions
func (s *Service) updateSessions() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for gameID := range s.activeGames {
		if err := s.monitor.UpdateSession(gameID); err != nil {
			// Session might have ended
			continue
		}

		session, err := s.monitor.GetSession(gameID)
		if err == nil {
			s.storage.SaveSession(session)
		}
	}

	// Check for stopped processes
	stopped := s.monitor.CheckProcesses()
	for _, gameID := range stopped {
		if ctx, exists := s.activeGames[gameID]; exists {
			s.mu.RUnlock()
			s.mu.Lock()
			s.stopGameMonitoring(gameID, ctx)
			s.mu.Unlock()
			s.mu.RLock()
		}
	}
}

