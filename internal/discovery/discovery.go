package discovery

import (
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
	excludedPaths map[string]bool
	gameSignatures []GameSignature
}

// GameSignature represents patterns that identify game directories
type GameSignature struct {
	ExePatterns    []string // Executable patterns
	DirPatterns    []string // Directory name patterns
	MetadataFiles  []string // Files that indicate a game directory
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
}

// New creates a new Discoverer with comprehensive path exclusions
func New() *Discoverer {
	excluded := map[string]bool{
		// Windows system paths
		"Windows":                true,
		"Program Files":          true,
		"Program Files (x86)":    true,
		"ProgramData":            true,
		"System32":               true,
		"SysWOW64":               true,
		"$Recycle.Bin":           true,
		"System Volume Information": true,
		"Recovery":               true,
		"PerfLogs":               true,
		// Common non-game application paths
		"Programs":               true,
		"Applications":           true,
		// Linux system paths
		"proc":                   true,
		"sys":                    true,
		"dev":                    true,
		"boot":                   true,
		"lib":                    true,
		"lib64":                  true,
		"bin":                    true,
		"sbin":                   true,
		"usr":                    true,
		"var":                    true,
		"tmp":                    true,
		"run":                    true,
		"lost+found":             true,
	}

	return &Discoverer{
		excludedPaths:  excluded,
		gameSignatures: getDefaultGameSignatures(),
	}
}

// DiscoverGames scans all mounted drives for game installations with structured logging
func (d *Discoverer) DiscoverGames() ([]DiscoveredGame, error) {
	startTime := time.Now()
	log.Printf("[INFO] Starting game discovery scan at %s", startTime.Format(time.RFC3339))
	
	var allGames []DiscoveredGame
	
	// Get all mounted drives
	drives, err := d.getMountedDrives()
	if err != nil {
		log.Printf("[ERROR] Failed to get mounted drives: %v", err)
		return nil, fmt.Errorf("failed to get mounted drives: %w", err)
	}

	log.Printf("[INFO] Found %d mounted drives to scan", len(drives))

	scannedDrives := 0
	for _, drive := range drives {
		driveStart := time.Now()
		games, err := d.scanDrive(drive)
		driveDuration := time.Since(driveStart)
		
		if err != nil {
			log.Printf("[WARN] Failed to scan drive %s after %v: %v", drive, driveDuration, err)
			continue
		}
		
		scannedDrives++
		allGames = append(allGames, games...)
		log.Printf("[INFO] Scanned drive %s in %v, found %d potential games", drive, driveDuration, len(games))
	}

	totalDuration := time.Since(startTime)
	log.Printf("[INFO] Discovery scan completed in %v: scanned %d/%d drives, found %d total games", 
		totalDuration, scannedDrives, len(drives), len(allGames))

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
			if d.isLikelyGameDirectory(path) {
				game, err := d.analyzeGameDirectory(path)
				if err == nil && game != nil {
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

	return hasExe && (hasGameFiles || d.hasGameSubdirectories(dirPath))
}

// isSystemExe checks if an executable is a system file
func (d *Discoverer) isSystemExe(name string) bool {
	systemExes := []string{
		"uninstall", "setup", "installer", "launcher",
		"update", "patch", "config", "settings",
	}
	nameLower := strings.ToLower(name)
	for _, sys := range systemExes {
		if strings.Contains(nameLower, sys) {
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
func (d *Discoverer) analyzeGameDirectory(dirPath string) (*DiscoveredGame, error) {
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
		if ext == ".exe" && !d.isSystemExe(strings.ToLower(name)) {
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

