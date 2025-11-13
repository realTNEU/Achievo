package discovery

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LLMClassifier uses a lightweight LLM to classify directories as games
type LLMClassifier struct {
	apiURL    string
	modelName string
	enabled   bool
	client    *http.Client
}

// NewLLMClassifier creates a new LLM-based classifier
// Supports Ollama (default) or any OpenAI-compatible API
func NewLLMClassifier(apiURL, modelName string) *LLMClassifier {
	if apiURL == "" {
		apiURL = "http://localhost:11434" // Default Ollama URL
	}
	if modelName == "" {
		modelName = "phi3:mini" // Small, fast model
	}

	return &LLMClassifier{
		apiURL:    apiURL,
		modelName: modelName,
		enabled:   true,
		client: &http.Client{
			Timeout: 180 * time.Second, // Extended timeout for remote servers, model loading, and streaming (3 minutes)
		},
	}
}

// Disable disables the LLM classifier (falls back to heuristics)
func (lc *LLMClassifier) Disable() {
	lc.enabled = false
}

// Enable enables the LLM classifier
func (lc *LLMClassifier) Enable() {
	lc.enabled = true
}

// IsAvailable checks if the LLM service is available
func (lc *LLMClassifier) IsAvailable(ctx context.Context) bool {
	if !lc.enabled {
		return false
	}

	// Try to ping the API (works for both local and remote servers)
	apiURL := lc.apiURL
	// Ensure URL doesn't have trailing slash
	if strings.HasSuffix(apiURL, "/") {
		apiURL = strings.TrimSuffix(apiURL, "/")
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/tags", apiURL), nil)
	if err != nil {
		return false
	}

	resp, err := lc.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// ClassifyDirectory analyzes a directory and determines if it's a game
func (lc *LLMClassifier) ClassifyDirectory(ctx context.Context, dirPath string) (isGame bool, confidence float64, reason string, err error) {
	result, err := lc.AnalyzeGameDirectory(ctx, dirPath)
	if err != nil {
		return false, 0, "", err
	}
	return result.IsGame, result.Confidence, result.Reason, nil
}

// AnalyzeGameDirectory performs comprehensive game analysis using LLM
func (lc *LLMClassifier) AnalyzeGameDirectory(ctx context.Context, dirPath string) (*GameAnalysis, error) {
	if !lc.enabled {
		return nil, fmt.Errorf("LLM classifier is disabled")
	}

	log.Printf("[LLM] Starting analysis for: %s", dirPath)

	// Gather comprehensive directory information
	dirInfo := lc.gatherDirectoryInfo(dirPath)
	if dirInfo == nil {
		return nil, fmt.Errorf("failed to gather directory info")
	}

	// Create enhanced prompt for detailed analysis
	prompt := lc.createDetailedAnalysisPrompt(dirInfo, dirPath)
	log.Printf("[LLM] Prompt created (length: %d chars), calling LLM API...", len(prompt))

	// Call LLM with retry logic
	startTime := time.Now()
	response, err := lc.callLLM(ctx, prompt)
	duration := time.Since(startTime)
	
	if err != nil {
		log.Printf("[LLM] LLM call failed for %s after %v: %v", dirPath, duration, err)
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	log.Printf("[LLM] LLM response received for: %s (length: %d chars, duration: %v)", dirPath, len(response), duration)

	// Parse detailed response
	return lc.parseDetailedResponse(response, dirPath)
}

// GameAnalysis contains comprehensive game information from LLM
type GameAnalysis struct {
	IsGame         bool     `json:"is_game"`
	Confidence     float64  `json:"confidence"`
	Reason         string   `json:"reason"`
	GameName       string   `json:"game_name"`
	GameRoot       string   `json:"game_root"`
	MainExecutable string   `json:"main_executable"`
	SavePaths      []string `json:"save_paths"`
	LogPaths       []string `json:"log_paths"`
	ConfigPaths    []string `json:"config_paths"`
}

// gatherDirectoryInfo collects information about a directory
func (lc *LLMClassifier) gatherDirectoryInfo(dirPath string) *DirectoryInfo {
	info := &DirectoryInfo{
		Path:           dirPath,
		DirectoryName:  filepath.Base(dirPath),
		Executables:    []string{},
		Subdirectories: []string{},
		Files:          []string{},
		HasGameFiles:   false,
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			info.Subdirectories = append(info.Subdirectories, name)
		} else {
			ext := strings.ToLower(filepath.Ext(name))
			if ext == ".exe" {
				info.Executables = append(info.Executables, name)
			} else {
				info.Files = append(info.Files, name)
			}

			// Check for game metadata files
			nameLower := strings.ToLower(name)
			if strings.Contains(nameLower, "steam_appid") ||
				strings.Contains(nameLower, "game.ini") ||
				strings.Contains(nameLower, "config.ini") ||
				strings.Contains(nameLower, ".unity3d") ||
				strings.Contains(nameLower, ".pak") {
				info.HasGameFiles = true
			}
		}
	}

	return info
}

// DirectoryInfo contains information about a directory for LLM analysis
type DirectoryInfo struct {
	Path           string   `json:"path"`
	DirectoryName  string   `json:"directory_name"`
	Executables    []string `json:"executables"`
	Subdirectories []string `json:"subdirectories"`
	Files          []string `json:"files"`
	HasGameFiles   bool     `json:"has_game_files"`
}

// createDetailedAnalysisPrompt creates an enhanced prompt for comprehensive game analysis
func (lc *LLMClassifier) createDetailedAnalysisPrompt(info *DirectoryInfo, dirPath string) string {
	// Limit the number of items to avoid token limits
	maxItems := 15
	executables := info.Executables
	if len(executables) > maxItems {
		executables = executables[:maxItems]
	}
	subdirs := info.Subdirectories
	if len(subdirs) > maxItems {
		subdirs = subdirs[:maxItems]
	}
	files := info.Files
	if len(files) > maxItems {
		files = files[:maxItems]
	}

	prompt := fmt.Sprintf(`You are an expert at analyzing PC game installations. Analyze this directory structure and provide detailed information.

Directory Path: %s
Directory Name: %s
Executables Found: %s
Subdirectories: %s
Files: %s
Has Game Metadata: %v

Your task:
1. Determine if this is a REAL PC GAME (not software, utility, or development tool)
2. If it's a game, identify:
   - The actual game name (not folder name)
   - The game root folder (may be this directory or a parent)
   - The main executable file name
   - Likely save file paths (common locations: saves/, SaveGames/, AppData/, Documents/)
   - Log file paths (common: logs/, Logs/, game.log)
   - Config file paths (common: config/, Config/, settings/)

Rules:
- GAMES: Have game executables, save folders, game data files, Steam App IDs, or game engines
- NOT GAMES: Development tools, servers, virtualization software, network tools, email clients, web servers, database servers, system utilities, installers

Respond ONLY with valid JSON:
{
  "is_game": true/false,
  "confidence": 0.0-1.0,
  "reason": "brief explanation",
  "game_name": "actual game name or empty string",
  "game_root": "relative path to game root from analyzed directory or empty",
  "main_executable": "executable filename or empty",
  "save_paths": ["path1", "path2"],
  "log_paths": ["path1", "path2"],
  "config_paths": ["path1", "path2"]
}`,
		dirPath,
		info.DirectoryName,
		strings.Join(executables, ", "),
		strings.Join(subdirs, ", "),
		strings.Join(files, ", "),
		info.HasGameFiles)

	return prompt
}

// createClassificationPrompt creates a prompt for the LLM (legacy, for simple classification)
func (lc *LLMClassifier) createClassificationPrompt(info *DirectoryInfo) string {
	// Limit the number of items to avoid token limits
	maxItems := 10
	executables := info.Executables
	if len(executables) > maxItems {
		executables = executables[:maxItems]
	}
	subdirs := info.Subdirectories
	if len(subdirs) > maxItems {
		subdirs = subdirs[:maxItems]
	}
	files := info.Files
	if len(files) > maxItems {
		files = files[:maxItems]
	}

	prompt := fmt.Sprintf(`Analyze this directory and determine if it's a PC GAME or SOFTWARE/UTILITY.

Directory: %s
Directory Name: %s
Executables: %s
Subdirectories: %s
Files: %s
Has Game Metadata: %v

Rules:
- GAMES have game executables, save folders, game data files, or Steam App IDs
- SOFTWARE/UTILITIES include: development tools, servers, virtualization, network tools, email clients, web servers, database servers, system utilities
- If unsure, classify as SOFTWARE/UTILITY

Respond ONLY with valid JSON:
{
  "is_game": true/false,
  "confidence": 0.0-1.0,
  "reason": "brief explanation"
}`,
		info.Path,
		info.DirectoryName,
		strings.Join(executables, ", "),
		strings.Join(subdirs, ", "),
		strings.Join(files, ", "),
		info.HasGameFiles)

	return prompt
}

// LLMRequest represents a request to the LLM API
type LLMRequest struct {
	Model   string     `json:"model"`
	Prompt  string     `json:"prompt"`
	Stream  bool       `json:"stream"`
	Options LLMOptions `json:"options"`
}

// LLMOptions contains LLM generation options
type LLMOptions struct {
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
	MaxTokens   int     `json:"num_predict"`
}

// LLMResponse represents a response from the LLM API
type LLMResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// ClassificationResponse represents the parsed classification result
type ClassificationResponse struct {
	IsGame         bool     `json:"is_game"`
	Confidence     float64  `json:"confidence"`
	Reason         string   `json:"reason"`
	GameName       string   `json:"game_name,omitempty"`
	GameRoot       string   `json:"game_root,omitempty"`
	MainExecutable string   `json:"main_executable,omitempty"`
	SavePaths      []string `json:"save_paths,omitempty"`
	LogPaths       []string `json:"log_paths,omitempty"`
	ConfigPaths    []string `json:"config_paths,omitempty"`
}

// callLLM calls the LLM API (Ollama-compatible)
// Retries up to 3 times with exponential backoff to handle model loading delays
func (lc *LLMClassifier) callLLM(ctx context.Context, prompt string) (string, error) {
	// Ensure URL doesn't have trailing slash
	apiURL := lc.apiURL
	if strings.HasSuffix(apiURL, "/") {
		apiURL = strings.TrimSuffix(apiURL, "/")
	}
	url := fmt.Sprintf("%s/api/generate", apiURL)

	reqBody := LLMRequest{
		Model:  lc.modelName,
		Prompt: prompt,
		Stream: false,
		Options: LLMOptions{
			Temperature: 0.2, // Lower temperature for more consistent analysis
			TopP:        0.9,
			MaxTokens:   500, // Longer response for detailed analysis
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Retry logic for handling model loading delays
	maxRetries := 3
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2s, 4s, 8s
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			log.Printf("[LLM] Retry attempt %d/%d after %v (model may still be loading)", attempt+1, maxRetries, backoff)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Create request with context that has extended timeout
		// Use the parent context's timeout, but ensure at least 180s for this request
		// Note: The HTTP client has a 180s timeout, so this context should match or exceed it
		// This allows time for model loading, generation, and streaming
		reqCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
		req, err := http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			cancel()
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		// Make the request - the HTTP client timeout (60s) will be the limiting factor
		// but we allow the context to extend to 90s for retries
		resp, err := lc.client.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			// Check if it's a timeout/connection error that might be due to model loading
			errStr := err.Error()
			if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection") || 
			   strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "canceled") {
				log.Printf("[LLM] Request failed (attempt %d/%d): %v - model may still be loading, will retry", attempt+1, maxRetries, err)
				continue
			}
			// For other errors, return immediately
			return "", fmt.Errorf("non-retryable error: %w", err)
		}

		// Check status code
		if resp.StatusCode == http.StatusOK {
			// Don't cancel context until we've finished reading the stream
			// This prevents Ollama from closing the connection prematurely
			defer func() {
				resp.Body.Close()
				cancel()
			}()

			// Ollama streams responses even with stream:false
			// We need to read the stream line by line and accumulate the response
			// Use a larger buffer to avoid blocking on large responses
			var fullResponse strings.Builder
			scanner := bufio.NewScanner(resp.Body)
			
			// Increase buffer size to handle large responses (default is 64KB)
			buf := make([]byte, 0, 256*1024) // 256KB buffer
			scanner.Buffer(buf, 512*1024)     // Max 512KB per line
			
			chunkCount := 0
			done := false
			
			// Read stream continuously - this keeps the connection alive
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}
				
				chunkCount++
				
				// Parse each JSON line from the stream
				var chunk LLMResponse
				if err := json.Unmarshal([]byte(line), &chunk); err != nil {
					// Try to handle non-streaming response (single JSON object)
					if chunkCount == 1 {
						// Might be a single JSON response, try parsing as complete response
						var singleResp LLMResponse
						if err2 := json.Unmarshal([]byte(line), &singleResp); err2 == nil {
							if singleResp.Done {
								log.Printf("[LLM] Received single non-streaming response")
								return singleResp.Response, nil
							}
						}
					}
					// Log error with line preview
					linePreview := line
					if len(linePreview) > 100 {
						linePreview = linePreview[:100] + "..."
					}
					log.Printf("[LLM] Failed to parse chunk %d: %v, line preview: %s", chunkCount, err, linePreview)
					continue
				}
				
				// Accumulate response text
				if chunk.Response != "" {
					fullResponse.WriteString(chunk.Response)
				}
				
				// If done, we have the complete response
				if chunk.Done {
					done = true
					log.Printf("[LLM] Received done signal after %d chunks", chunkCount)
					break
				}
			}
			
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("failed to read response stream after %d chunks: %w", chunkCount, err)
			}
			
			// If we didn't get a done signal but have response, use it anyway
			if !done && chunkCount > 0 {
				log.Printf("[LLM] Stream ended without done signal, using accumulated response (%d chunks)", chunkCount)
			}
			
			result := fullResponse.String()
			if result == "" {
				return "", fmt.Errorf("empty response from LLM after %d chunks", chunkCount)
			}
			
			log.Printf("[LLM] Successfully received complete response (%d chunks, %d chars)", chunkCount, len(result))
			return result, nil
		}

		// Handle non-200 status codes
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()

		// Retry on 499 (client closed) or 500 (server error) - might be model loading
		if resp.StatusCode == 499 || resp.StatusCode == 500 {
			lastErr = fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
			log.Printf("[LLM] Request failed with status %d (attempt %d/%d): %s - model may still be loading", resp.StatusCode, attempt+1, maxRetries, string(body))
			continue
		}

		// For other status codes, return error immediately
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// All retries exhausted
	return "", fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// parseDetailedResponse parses the LLM response to extract comprehensive game analysis
func (lc *LLMClassifier) parseDetailedResponse(response string, basePath string) (*GameAnalysis, error) {
	response = strings.TrimSpace(response)

	// Find JSON object in response
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")
	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return nil, fmt.Errorf("no JSON found in response: %s", response)
	}

	jsonStr := response[startIdx : endIdx+1]

	var result ClassificationResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// Try to parse manually if JSON is malformed
		return lc.parseDetailedResponseManually(response, basePath)
	}

	// Convert to GameAnalysis
	analysis := &GameAnalysis{
		IsGame:         result.IsGame,
		Confidence:     result.Confidence,
		Reason:         result.Reason,
		GameName:       result.GameName,
		MainExecutable: result.MainExecutable,
		SavePaths:      result.SavePaths,
		LogPaths:       result.LogPaths,
		ConfigPaths:    result.ConfigPaths,
	}

	// Resolve game root path
	if result.GameRoot != "" {
		if filepath.IsAbs(result.GameRoot) {
			analysis.GameRoot = result.GameRoot
		} else {
			analysis.GameRoot = filepath.Join(basePath, result.GameRoot)
		}
	} else {
		analysis.GameRoot = basePath
	}

	return analysis, nil
}

// parseResponse parses the LLM response to extract classification (legacy)
func (lc *LLMClassifier) parseResponse(response string) (isGame bool, confidence float64, reason string, err error) {
	analysis, err := lc.parseDetailedResponse(response, "")
	if err != nil {
		return false, 0, "", err
	}
	return analysis.IsGame, analysis.Confidence, analysis.Reason, nil
}

// parseDetailedResponseManually tries to parse response if JSON parsing fails
func (lc *LLMClassifier) parseDetailedResponseManually(response string, basePath string) (*GameAnalysis, error) {
	responseLower := strings.ToLower(response)

	analysis := &GameAnalysis{
		GameRoot:    basePath,
		SavePaths:   []string{},
		LogPaths:    []string{},
		ConfigPaths: []string{},
	}

	// Look for keywords
	if strings.Contains(responseLower, `"is_game": true`) ||
		strings.Contains(responseLower, "is_game: true") ||
		(strings.Contains(responseLower, "true") && strings.Contains(responseLower, "game")) {
		analysis.IsGame = true
		analysis.Confidence = 0.7
	} else {
		analysis.IsGame = false
		analysis.Confidence = 0.7
	}

	// Extract reason if possible
	if idx := strings.Index(responseLower, "reason"); idx != -1 {
		reasonStart := idx + 6
		if reasonStart < len(response) {
			analysis.Reason = response[reasonStart:]
			if endIdx := strings.Index(analysis.Reason, "\""); endIdx != -1 {
				analysis.Reason = analysis.Reason[:endIdx]
			}
		}
	}

	if analysis.Reason == "" {
		analysis.Reason = "LLM classification"
	}

	return analysis, nil
}

// parseResponseManually tries to parse response if JSON parsing fails (legacy)
func (lc *LLMClassifier) parseResponseManually(response string) (isGame bool, confidence float64, reason string, err error) {
	analysis, err := lc.parseDetailedResponseManually(response, "")
	if err != nil {
		return false, 0, "", err
	}
	return analysis.IsGame, analysis.Confidence, analysis.Reason, nil
}
