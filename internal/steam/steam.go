package steam

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"achievo/internal/models"
	"achievo/internal/storage"
)

// SteamClient handles Steam Web API interactions
type SteamClient struct {
	apiKey  string
	client  *http.Client
	storage *storage.Storage
}

// SteamAPIResponse represents the response from Steam API
type SteamAPIResponse struct {
	PlayerStats struct {
		SteamID      string `json:"steamID"`
		GameName     string `json:"gameName"`
		Achievements []struct {
			APIName    string `json:"apiname"`
			Achieved   int    `json:"achieved"`
			UnlockTime int64  `json:"unlocktime"`
		} `json:"achievements"`
	} `json:"playerstats"`
}

// SteamSchemaResponse represents the schema response from Steam API
type SteamSchemaResponse struct {
	Game struct {
		GameName           string `json:"gameName"`
		GameVersion        string `json:"gameVersion"`
		AvailableGameStats struct {
			Achievements []struct {
				Name         string `json:"name"`
				DefaultValue int    `json:"defaultvalue"`
				DisplayName  string `json:"displayName"`
				Hidden       int    `json:"hidden"`
				Description  string `json:"description"`
				Icon         string `json:"icon"`
				IconGray     string `json:"icongray"`
			} `json:"achievements"`
		} `json:"availableGameStats"`
	} `json:"game"`
}

// New creates a new SteamClient
func New(apiKey string, storage *storage.Storage) *SteamClient {
	return &SteamClient{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		storage: storage,
	}
}

// FetchAchievements fetches achievement definitions for a Steam App ID
func (s *SteamClient) FetchAchievements(appID string) ([]models.Achievement, error) {
	// Check cache first
	cached, err := s.storage.GetAchievementsByGame(appID)
	if err == nil && len(cached) > 0 {
		// Check if cache is recent (less than 24 hours old)
		if len(cached) > 0 {
			recent := cached[0].FetchedAt.After(time.Now().Add(-24 * time.Hour))
			if recent {
				return cached, nil
			}
		}
	}

	// Fetch from Steam API
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?key=%s&appid=%s", s.apiKey, appID)

	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Steam API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Steam API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Result SteamSchemaResponse `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode Steam API response: %w", err)
	}

	// Convert to our models
	achievements := make([]models.Achievement, 0)
	for _, ach := range apiResp.Result.Game.AvailableGameStats.Achievements {
		achievement := models.Achievement{
			ID:            fmt.Sprintf("%s-%s", appID, ach.Name),
			GameID:        appID,
			SteamAppID:    appID,
			Name:          ach.DisplayName,
			Description:   ach.Description,
			IconURL:       ach.Icon,
			IconLockedURL: ach.IconGray,
			IsHidden:      ach.Hidden == 1,
			FetchedAt:     time.Now(),
		}
		achievements = append(achievements, achievement)

		// Save to storage
		if err := s.storage.SaveAchievement(&achievement); err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to save achievement %s: %v\n", achievement.ID, err)
		}
	}

	return achievements, nil
}

// GetAchievement retrieves a specific achievement definition
func (s *SteamClient) GetAchievement(appID, achievementID string) (*models.Achievement, error) {
	// Try to get from storage first
	achievements, err := s.storage.GetAchievementsByGame(appID)
	if err == nil {
		for _, ach := range achievements {
			if ach.ID == achievementID {
				return &ach, nil
			}
		}
	}

	// If not found, fetch all achievements
	achievements, err = s.FetchAchievements(appID)
	if err != nil {
		return nil, err
	}

	for _, ach := range achievements {
		if ach.ID == achievementID {
			return &ach, nil
		}
	}

	return nil, fmt.Errorf("achievement not found: %s", achievementID)
}

// CacheAchievements caches achievement definitions
func (s *SteamClient) CacheAchievements(appID string, achievements []models.Achievement) error {
	for _, ach := range achievements {
		if err := s.storage.SaveAchievement(&ach); err != nil {
			return fmt.Errorf("failed to cache achievement %s: %w", ach.ID, err)
		}
	}
	return nil
}

// FetchAndStoreSchema fetches and stores Steam achievement schema with retries and rate limiting
func (s *SteamClient) FetchAndStoreSchema(appID string) error {
	maxRetries := 3
	retryDelay := 2 * time.Second
	rateLimitDelay := 1 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Rate limiting: wait between requests
		if attempt > 1 {
			time.Sleep(rateLimitDelay)
		}

		achievements, err := s.fetchSchemaWithRetry(appID, attempt, retryDelay)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to fetch schema after %d attempts: %w", maxRetries, err)
			}
			log.Printf("Attempt %d failed, retrying: %v", attempt, err)
			time.Sleep(retryDelay * time.Duration(attempt))
			continue
		}

		// Validate schema
		if err := s.validateSchema(achievements); err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}

		// Store in MongoDB using UpsertAchievementDefs
		if err := s.storage.UpsertAchievementDefs(achievements); err != nil {
			return fmt.Errorf("failed to store achievements: %w", err)
		}

		log.Printf("Successfully fetched and stored %d achievements for Steam App ID %s", len(achievements), appID)
		return nil
	}

	return fmt.Errorf("unexpected error in FetchAndStoreSchema")
}

// fetchSchemaWithRetry fetches schema with a single retry attempt
func (s *SteamClient) fetchSchemaWithRetry(appID string, _ int, _ time.Duration) ([]models.Achievement, error) {
	url := fmt.Sprintf("https://api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/?key=%s&appid=%s", s.apiKey, appID)

	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting (429) or server errors (5xx)
	if resp.StatusCode == http.StatusTooManyRequests {
		// Wait longer for rate limit
		time.Sleep(5 * time.Second)
		return nil, fmt.Errorf("rate limited, will retry")
	}

	if resp.StatusCode >= 500 {
		// Server error, retry
		return nil, fmt.Errorf("server error %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Steam API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Result SteamSchemaResponse `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our models
	achievements := make([]models.Achievement, 0)
	for _, ach := range apiResp.Result.Game.AvailableGameStats.Achievements {
		achievement := models.Achievement{
			ID:            fmt.Sprintf("%s-%s", appID, ach.Name),
			GameID:        appID,
			SteamAppID:    appID,
			Name:          ach.DisplayName,
			Description:   ach.Description,
			IconURL:       ach.Icon,
			IconLockedURL: ach.IconGray,
			IsHidden:      ach.Hidden == 1,
			FetchedAt:     time.Now(),
		}
		achievements = append(achievements, achievement)
	}

	return achievements, nil
}

// validateSchema validates the fetched schema
func (s *SteamClient) validateSchema(achievements []models.Achievement) error {
	if len(achievements) == 0 {
		return fmt.Errorf("schema contains no achievements")
	}

	// Check for required fields
	for _, ach := range achievements {
		if ach.ID == "" {
			return fmt.Errorf("achievement missing ID")
		}
		if ach.Name == "" {
			return fmt.Errorf("achievement %s missing name", ach.ID)
		}
		if ach.SteamAppID == "" {
			return fmt.Errorf("achievement %s missing Steam App ID", ach.ID)
		}
	}

	return nil
}
