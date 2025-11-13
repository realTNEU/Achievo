package discovery

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"achievo/internal/models"
)

// Discoverer scans drives for game installations
type Discoverer struct {
	excludedPaths  map[string]bool
	gameSignatures []GameSignature
	llmClassifier  *LLMClassifier
}

// GameSignature represents patterns that identify game directories
type GameSignature struct {
	ExePatterns   []string // Executable patterns
	DirPatterns   []string // Directory name patterns
	MetadataFiles []string // Files that indicate a game directory
}

// DiscoveredGame represents a discovered game installation
type DiscoveredGame struct {
	Name             string
	ExecutablePath   string
	InstallDirectory string
	ProcessName      string
	Type             models.GameType
	SteamAppID       string
	Confidence       float64 // 0.0 to 1.0
	// LLM-inferred paths (optional, populated when LLM analysis is used)
	InferredSavePaths   []string
	InferredLogPaths    []string
	InferredConfigPaths []string
}

// New creates a new Discoverer with comprehensive path exclusions
func New() *Discoverer {
	excluded := map[string]bool{
		// Windows system paths
		"Windows":                   true,
		"Program Files":             true,
		"Program Files (x86)":       true,
		"ProgramData":               true,
		"System32":                  true,
		"SysWOW64":                  true,
		"$Recycle.Bin":              true,
		"System Volume Information": true,
		"Recovery":                  true,
		"PerfLogs":                  true,
		// Common non-game application paths
		"Programs":     true,
		"Applications": true,
		// Linux system paths
		"proc":       true,
		"sys":        true,
		"dev":        true,
		"boot":       true,
		"lib":        true,
		"lib64":      true,
		"bin":        true,
		"sbin":       true,
		"usr":        true,
		"var":        true,
		"tmp":        true,
		"run":        true,
		"lost+found": true,
	}

	return &Discoverer{
		excludedPaths:  excluded,
		gameSignatures: getDefaultGameSignatures(),
		llmClassifier:  NewLLMClassifier("", ""), // Default Ollama setup
	}
}

// NewWithLLM creates a discoverer with custom LLM configuration
func NewWithLLM(apiURL, modelName string) *Discoverer {
	excluded := map[string]bool{
		// Windows system paths
		"Windows":                   true,
		"Program Files":             true,
		"Program Files (x86)":       true,
		"ProgramData":               true,
		"System32":                  true,
		"SysWOW64":                  true,
		"$Recycle.Bin":              true,
		"System Volume Information": true,
		"Recovery":                  true,
		"PerfLogs":                  true,
		// Common non-game application paths
		"Programs":     true,
		"Applications": true,
		// Linux system paths
		"proc":       true,
		"sys":        true,
		"dev":        true,
		"boot":       true,
		"lib":        true,
		"lib64":      true,
		"bin":        true,
		"sbin":       true,
		"usr":        true,
		"var":        true,
		"tmp":        true,
		"run":        true,
		"lost+found": true,
	}

	return &Discoverer{
		excludedPaths:  excluded,
		gameSignatures: getDefaultGameSignatures(),
		llmClassifier:  NewLLMClassifier(apiURL, modelName),
	}
}

// DiscoverGames scans all mounted drives and custom paths for game installations
func (d *Discoverer) DiscoverGames(customPaths []string) ([]DiscoveredGame, error) {
	startTime := time.Now()
	log.Printf("[INFO] Starting game discovery scan at %s", startTime.Format(time.RFC3339))

	var allGames []DiscoveredGame

	// Get all mounted drives
	drives, err := d.getMountedDrives()
	if err != nil {
		log.Printf("[ERROR] Failed to get mounted drives: %v", err)
		return nil, fmt.Errorf("failed to get mounted drives: %w", err)
	}

	// Combine drives with custom scan paths
	allPaths := append(drives, customPaths...)

	log.Printf("[INFO] Found %d mounted drives and %d custom paths to scan", len(drives), len(customPaths))

	scannedPaths := 0
	totalSkipped := 0
	for _, path := range allPaths {
		pathStart := time.Now()
		games, err := d.scanDrive(path)
		pathDuration := time.Since(pathStart)

		if err != nil {
			log.Printf("[WARN] Failed to scan path %s after %v: %v", path, pathDuration, err)
			continue
		}

		// Filter out system utilities and non-games
		validGames := []DiscoveredGame{}
		for _, game := range games {
			if d.isSystemPath(game.ExecutablePath) || d.isSystemExe(game.ProcessName) {
				totalSkipped++
				log.Printf("[SKIP] Filtered out non-game: %s (%s)", game.Name, game.ProcessName)
				continue
			}
			validGames = append(validGames, game)
		}

		scannedPaths++
		allGames = append(allGames, validGames...)
		log.Printf("[INFO] Scanned path %s in %v | Found: %d games | Skipped: %d utilities",
			path, pathDuration, len(validGames), len(games)-len(validGames))
	}

	totalDuration := time.Since(startTime)
	log.Printf("[INFO] ===== Discovery Summary =====")
	log.Printf("[INFO] Duration: %v", totalDuration)
	log.Printf("[INFO] Paths scanned: %d/%d", scannedPaths, len(allPaths))
	log.Printf("[INFO] Games found: %d", len(allGames))
	log.Printf("[INFO] Utilities/drivers filtered: %d", totalSkipped)
	log.Printf("[INFO] ==============================")

	return allGames, nil
}

// getMountedDrives returns all mounted drive letters (Windows) or mount points (Linux)
// Also includes standard game install paths
func (d *Discoverer) getMountedDrives() ([]string, error) {
	var drives []string

	// Windows: C:\, D:\, etc.
	if runtime.GOOS == "windows" {
		for letter := 'A'; letter <= 'Z'; letter++ {
			drive := string(letter) + ":\\"
			if _, err := os.Stat(drive); err == nil {
				drives = append(drives, drive)
			}
		}

		// Add standard Windows game install paths
		standardPaths := []string{
			filepath.Join(os.Getenv("PROGRAMFILES"), "Steam", "steamapps", "common"),
			filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Steam", "steamapps", "common"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs"),
		}

		for _, path := range standardPaths {
			if path != "" && path != "\\" {
				if _, err := os.Stat(path); err == nil {
					drives = append(drives, path)
				}
			}
		}
	} else {
		// Linux/Unix: common mount points
		commonMounts := []string{"/", "/home", "/mnt", "/media", "/opt"}
		for _, mount := range commonMounts {
			if _, err := os.Stat(mount); err == nil {
				drives = append(drives, mount)
			}
		}

		// Add standard Linux game paths
		home := os.Getenv("HOME")
		if home != "" {
			linuxPaths := []string{
				filepath.Join(home, ".steam", "steam", "steamapps", "common"),
				filepath.Join(home, ".local", "share", "Steam", "steamapps", "common"),
				filepath.Join(home, "Games"),
			}
			for _, path := range linuxPaths {
				if _, err := os.Stat(path); err == nil {
					drives = append(drives, path)
				}
			}
		}
	}

	return drives, nil
}

// scanDrive recursively scans a drive for game installations
func (d *Discoverer) scanDrive(rootPath string) ([]DiscoveredGame, error) {
	var games []DiscoveredGame
	visited := make(map[string]bool)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue scanning
		}

		// Skip excluded paths
		if d.isExcluded(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if this looks like a game directory
		if info.IsDir() {
			// Skip system paths entirely
			if d.isSystemPath(path) {
				return filepath.SkipDir
			}

			if d.isLikelyGameDirectory(path) {
				game, err := d.analyzeGameDirectory(path)
				if err != nil {
					// Log rejection reason for debugging
					if strings.Contains(err.Error(), "rejected") || strings.Contains(err.Error(), "no executable") {
						// Silently skip - already filtered
					}
					return filepath.SkipDir
				}

				if game != nil {
					// Additional filtering: exclude if executable is in system path
					if d.isSystemPath(game.ExecutablePath) {
						return filepath.SkipDir
					}

					// Check if executable name suggests it's not a game
					if d.isSystemExe(game.ProcessName) {
						return filepath.SkipDir
					}

					// Normalize and check for duplicates
					normalized := d.normalizeGameName(game.Name)
					if !visited[normalized] {
						games = append(games, *game)
						visited[normalized] = true
					}
				}
				// Skip subdirectories of game directories
				return filepath.SkipDir
			}
		}

		return nil
	})

	return games, err
}

// isExcluded checks if a path should be excluded from scanning
func (d *Discoverer) isExcluded(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if d.excludedPaths[part] {
			return true
		}
	}
	return false
}

// isLikelyGameDirectory checks if a directory looks like a game installation
func (d *Discoverer) isLikelyGameDirectory(dirPath string) bool {
	// First check: if path contains software/utility indicators, likely not a game
	pathLower := strings.ToLower(dirPath)
	softwareIndicators := []string{
		"\\softwares\\", "\\software\\", "\\tools\\", "\\utilities\\",
		"\\xampp\\", "\\wamp\\", "\\mamp\\", "\\vmware\\",
		"\\virtualbox\\", "\\apache\\", "\\nginx\\",
		"\\superscan\\", "\\mercurymail\\", "\\sendmail\\",
	}
	for _, indicator := range softwareIndicators {
		if strings.Contains(pathLower, indicator) {
			return false // Likely a software tool, not a game
		}
	}

	// Try LLM classification if available (with extended timeout for model loading and streaming)
	if d.llmClassifier != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		if d.llmClassifier.IsAvailable(ctx) {
			isGame, confidence, reason, err := d.llmClassifier.ClassifyDirectory(ctx, dirPath)
			if err != nil {
				// Log error but don't block - fall through to heuristics
				log.Printf("[LLM] Classification failed for %s: %v (falling back to heuristics)", filepath.Base(dirPath), err)
			} else {
				// Use LLM result if confidence is high enough
				if confidence >= 0.6 {
					if isGame {
						log.Printf("[LLM] Classified as GAME: %s (confidence: %.2f, reason: %s)",
							filepath.Base(dirPath), confidence, reason)
						return true
					} else {
						log.Printf("[LLM] Classified as SOFTWARE: %s (confidence: %.2f, reason: %s)",
							filepath.Base(dirPath), confidence, reason)
						return false
					}
				}
				// If confidence is low, fall through to heuristics
			}
		} else {
			log.Printf("[LLM] LLM service not available, using heuristics for %s", filepath.Base(dirPath))
		}
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}

	hasExe := false
	hasGameFiles := false
	dirName := strings.ToLower(filepath.Base(dirPath))

	// Check for common game indicators
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		ext := filepath.Ext(name)

		// Check for executables
		if ext == ".exe" {
			// Exclude common system executables
			if !d.isSystemExe(name) {
				hasExe = true
			}
		}

		// Check for game metadata files
		if d.isGameMetadataFile(name) {
			hasGameFiles = true
		}

		// Check directory name patterns
		for _, sig := range d.gameSignatures {
			for _, pattern := range sig.DirPatterns {
				matched, _ := filepath.Match(strings.ToLower(pattern), dirName)
				if matched {
					return true
				}
			}
		}
	}

	// Additional check: if directory name suggests it's software/tool, reject
	softwareNames := []string{
		"superscan", "vmware", "virtualbox", "mercury", "mail",
		"php", "apache", "mysql", "xampp", "wamp", "mamp",
		"sendmail", "postfix", "nginx", "iis",
	}
	for _, swName := range softwareNames {
		if strings.Contains(dirName, swName) {
			return false // Software tool, not a game
		}
	}

	return hasExe && (hasGameFiles || d.hasGameSubdirectories(dirPath))
}

// isSystemExe checks if an executable is a system file, driver, or utility
func (d *Discoverer) isSystemExe(name string) bool {
	nameLower := strings.ToLower(name)

	// System utilities and drivers
	systemExes := []string{
		"uninstall", "setup", "installer", "launcher",
		"update", "patch", "config", "settings",
		"driver", "service", "svc", "wrapper", "cache",
		"streamer", "stream", "server", "client",
		"manager", "tool", "utility", "helper",
		"dll", "inject", "shim", "cli",
		"7z", "zip", "rar", "extract",
		"nvstreamer", "nvfvsdk", "nvoawrapper",
		"dviout", "tex", "latex",
		"backup", "restore", "wiibackup",
		"scuba", "iw5mp", "gfexperience",
		// Development tools
		"gui-32", "gui-64", "python", "py", "pip",
		"setuptools", "wheel", "distutils",
		"node", "npm", "yarn", "npx",
		// Network/security tools
		"superscan", "scan", "nmap", "wireshark", "netcat",
		// Virtualization software
		"vmware", "virtualbox", "vbox", "qemu", "hyper-v",
		// Email/server software
		"mercury", "mailtodisk", "sendmail", "postfix", "exim",
		"apache", "nginx", "iis", "xampp", "wamp", "mamp",
		// Web server components
		"php", "php-cgi", "php-fpm", "mysql", "mariadb",
		"postgres", "mongodb", "redis",
		// Generic names
		"games", // Too generic, often a folder name
	}

	for _, sys := range systemExes {
		if strings.Contains(nameLower, sys) {
			return true
		}
	}

	// Exclude executables in driver/system paths
	return false
}

// isSystemPath checks if a path is a system/driver/utility location
func (d *Discoverer) isSystemPath(path string) bool {
	pathLower := strings.ToLower(path)

	systemPaths := []string{
		"\\esupport\\", "\\edriver\\",
		"\\texlive\\", "\\tex\\",
		"\\nvidia\\", "\\amd\\", "\\intel\\",
		"\\drivers\\", "\\driver\\",
		"\\system32\\", "\\syswow64\\",
		"\\program files\\common files\\",
		"\\windows\\",
		"\\cache\\",
		// Development tools and environments
		"\\pyenv\\", "\\python\\", "\\site-packages\\",
		"\\node_modules\\", "\\npm\\", "\\yarn\\",
		"\\.git\\", "\\venv\\", "\\env\\", "\\virtualenv\\",
		"\\anaconda\\", "\\conda\\", "\\miniconda\\",
		"\\appdata\\local\\programs\\python\\",
		"\\appdata\\roaming\\python\\",
		// Specific system temp locations (not user temp for testing)
		"\\windows\\temp\\", "\\windows\\tmp\\",
		// Software/utilities directories (not games)
		"\\softwares\\", "\\software\\", "\\tools\\", "\\utilities\\",
		"\\xampp\\", "\\wamp\\", "\\mamp\\", "\\lampp\\",
		"\\vmware\\", "\\virtualbox\\", "\\qemu\\",
		"\\apache\\", "\\nginx\\", "\\iis\\",
		"\\php\\", "\\mysql\\", "\\postgres\\",
		// Network/security tools
		"\\superscan\\", "\\nmap\\", "\\wireshark\\",
		// Email servers
		"\\mercurymail\\", "\\sendmail\\", "\\postfix\\",
	}

	for _, sysPath := range systemPaths {
		if strings.Contains(pathLower, sysPath) {
			return true
		}
	}
	return false
}

// isGameMetadataFile checks if a file indicates a game directory
func (d *Discoverer) isGameMetadataFile(name string) bool {
	gameFiles := []string{
		"steam_appid.txt", "game.ini", "config.ini",
		"version.txt", "readme.txt", "changelog.txt",
		".steam", "steam_api.dll", "steam_api64.dll",
	}
	nameLower := strings.ToLower(name)
	for _, gf := range gameFiles {
		if nameLower == gf || strings.Contains(nameLower, gf) {
			return true
		}
	}
	return false
}

// hasGameSubdirectories checks for common game subdirectories
func (d *Discoverer) hasGameSubdirectories(dirPath string) bool {
	commonDirs := []string{"saves", "save", "data", "config", "logs", "log"}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirName := strings.ToLower(entry.Name())
			for _, common := range commonDirs {
				if dirName == common {
					return true
				}
			}
		}
	}
	return false
}

// analyzeGameDirectory analyzes a directory to extract game information
// Uses LLM analysis if available, falls back to heuristics
func (d *Discoverer) analyzeGameDirectory(dirPath string) (*DiscoveredGame, error) {
	// Try LLM analysis first if available (with extended timeout for model loading, streaming, and retries)
	if d.llmClassifier != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		if d.llmClassifier.IsAvailable(ctx) {
			log.Printf("[LLM] Analyzing directory: %s", filepath.Base(dirPath))
			analysis, err := d.llmClassifier.AnalyzeGameDirectory(ctx, dirPath)
			if err != nil {
				log.Printf("[LLM] Analysis failed for %s: %v (falling back to heuristics)", filepath.Base(dirPath), err)
			} else if analysis != nil && analysis.IsGame && analysis.Confidence >= 0.6 {
				// Use LLM analysis results
				log.Printf("[LLM] Analysis successful for %s (confidence: %.2f)", filepath.Base(dirPath), analysis.Confidence)
				return d.createGameFromLLMAnalysis(analysis, dirPath)
			} else if analysis != nil {
				log.Printf("[LLM] Analysis result: not a game or low confidence (%.2f) for %s", analysis.Confidence, filepath.Base(dirPath))
			}
		} else {
			log.Printf("[LLM] LLM service not available, using heuristics for %s", filepath.Base(dirPath))
		}
	}

	// Fall back to heuristic analysis
	return d.analyzeGameDirectoryHeuristic(dirPath)
}

// createGameFromLLMAnalysis creates a DiscoveredGame from LLM analysis
func (d *Discoverer) createGameFromLLMAnalysis(analysis *GameAnalysis, analyzedPath string) (*DiscoveredGame, error) {
	gameRoot := analysis.GameRoot
	if !filepath.IsAbs(gameRoot) {
		gameRoot = filepath.Join(analyzedPath, gameRoot)
	}

	// Find the main executable
	var exePath string
	var processName string

	if analysis.MainExecutable != "" {
		// Try to find the executable in the game root
		exePath = filepath.Join(gameRoot, analysis.MainExecutable)
		if _, err := os.Stat(exePath); os.IsNotExist(err) {
			// Try in analyzed path
			exePath = filepath.Join(analyzedPath, analysis.MainExecutable)
			if _, err := os.Stat(exePath); os.IsNotExist(err) {
				// Search for it
				exePath = d.findExecutableInPath(gameRoot, analysis.MainExecutable)
			}
		}
		processName = analysis.MainExecutable
	} else {
		// Fallback: find any .exe in game root
		exePath, processName = d.findMainExecutable(gameRoot)
	}

	if exePath == "" {
		return nil, fmt.Errorf("no executable found in game root: %s", gameRoot)
	}

	// Check for Steam App ID
	steamAppID := d.detectSteamAppID(gameRoot)

	// Use LLM-provided game name or infer from directory
	gameName := analysis.GameName
	if gameName == "" {
		gameName = d.normalizeGameName(filepath.Base(gameRoot))
	}

	gameType := models.GameTypeNonSteam
	if steamAppID != "" {
		gameType = models.GameTypeSteam
	}

	return &DiscoveredGame{
		Name:                gameName,
		ExecutablePath:      exePath,
		InstallDirectory:    gameRoot,
		ProcessName:         processName,
		Type:                gameType,
		SteamAppID:          steamAppID,
		Confidence:          analysis.Confidence,
		InferredSavePaths:   analysis.SavePaths,
		InferredLogPaths:    analysis.LogPaths,
		InferredConfigPaths: analysis.ConfigPaths,
	}, nil
}

// findExecutableInPath searches for an executable in a directory
func (d *Discoverer) findExecutableInPath(dirPath, exeName string) string {
	var foundPath string
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(info.Name(), exeName) {
			foundPath = path
			return filepath.SkipAll
		}
		return nil
	})
	return foundPath
}

// findMainExecutable finds the main executable in a directory
func (d *Discoverer) findMainExecutable(dirPath string) (string, string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext == ".exe" && !d.isSystemExe(strings.ToLower(name)) {
			return filepath.Join(dirPath, name), name
		}
	}
	return "", ""
}

// analyzeGameDirectoryHeuristic analyzes a directory using heuristics (fallback)
func (d *Discoverer) analyzeGameDirectoryHeuristic(dirPath string) (*DiscoveredGame, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var exePath string
	var processName string
	confidence := 0.5

	// Find main executable
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)
		nameLower := strings.ToLower(name)

		// Skip if it's a system/utility executable
		if ext == ".exe" && !d.isSystemExe(nameLower) {
			// Additional check: reject if executable name suggests software tool
			softwareExePatterns := []string{
				"superscan", "vmware", "virtualbox", "mercury", "mail",
				"php", "php-cgi", "php-fpm", "sendmail", "postfix",
				"apache", "nginx", "mysql", "mariadb",
			}
			isSoftwareTool := false
			for _, pattern := range softwareExePatterns {
				if strings.Contains(nameLower, pattern) {
					isSoftwareTool = true
					break
				}
			}
			if isSoftwareTool {
				continue // Skip software tools
			}

			// Prefer executables that match directory name
			dirName := strings.ToLower(filepath.Base(dirPath))
			exeName := strings.ToLower(strings.TrimSuffix(name, ext))

			if strings.Contains(dirName, exeName) || strings.Contains(exeName, dirName) {
				exePath = filepath.Join(dirPath, name)
				processName = name
				confidence = 0.9
				break
			} else if exePath == "" {
				exePath = filepath.Join(dirPath, name)
				processName = name
			}
		}
	}

	if exePath == "" {
		return nil, fmt.Errorf("no executable found")
	}

	// Check for Steam App ID
	steamAppID := d.detectSteamAppID(dirPath)

	// Normalize game name
	gameName := d.normalizeGameName(filepath.Base(dirPath))

	// Reject games with generic names that are likely not games
	genericNames := []string{
		"games", "game", "apps", "app", "programs", "program",
		"tools", "tool", "utilities", "utility", "bin", "binaries",
		"software", "applications", "executables",
	}
	gameNameLower := strings.ToLower(gameName)
	for _, generic := range genericNames {
		if gameNameLower == generic {
			return nil, fmt.Errorf("rejected generic name: %s", gameName)
		}
	}

	gameType := models.GameTypeNonSteam
	if steamAppID != "" {
		gameType = models.GameTypeSteam
		confidence = 0.95
	}

	return &DiscoveredGame{
		Name:             gameName,
		ExecutablePath:   exePath,
		InstallDirectory: dirPath,
		ProcessName:      processName,
		Type:             gameType,
		SteamAppID:       steamAppID,
		Confidence:       confidence,
	}, nil
}

// detectSteamAppID detects Steam App ID from steam_appid.txt
func (d *Discoverer) detectSteamAppID(dirPath string) string {
	steamIDFile := filepath.Join(dirPath, "steam_appid.txt")
	data, err := os.ReadFile(steamIDFile)
	if err != nil {
		return ""
	}

	appID := strings.TrimSpace(string(data))
	// Validate it's a number
	if matched, _ := regexp.MatchString(`^\d+$`, appID); matched {
		return appID
	}
	return ""
}

// normalizeGameName normalizes a game name for matching
func (d *Discoverer) normalizeGameName(name string) string {
	// Remove common suffixes
	suffixes := []string{"game", "edition", "remastered", "remaster", "definitive"}
	nameLower := strings.ToLower(name)

	for _, suffix := range suffixes {
		if strings.HasSuffix(nameLower, " "+suffix) {
			nameLower = strings.TrimSuffix(nameLower, " "+suffix)
		}
	}

	// Remove special characters
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	normalized := reg.ReplaceAllString(nameLower, "")

	return normalized
}

// getDefaultGameSignatures returns default game directory signatures
func getDefaultGameSignatures() []GameSignature {
	return []GameSignature{
		{
			DirPatterns: []string{"*game*", "*games*"},
		},
		{
			DirPatterns: []string{"steamapps", "common"},
		},
	}
}

// ToGame converts a DiscoveredGame to a models.Game
func (dg *DiscoveredGame) ToGame() *models.Game {
	discoverer := New()
	gameID := fmt.Sprintf("game-%s-%d", discoverer.normalizeGameName(dg.Name), time.Now().Unix())
	return &models.Game{
		ID:               gameID,
		Name:             dg.Name,
		Type:             dg.Type,
		SteamAppID:       dg.SteamAppID,
		ExecutablePath:   dg.ExecutablePath,
		ProcessName:      dg.ProcessName,
		InstallDirectory: dg.InstallDirectory,
		FirstDetected:    time.Now(),
		LastPlayed:       time.Now(),
	}
}
