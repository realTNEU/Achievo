package steam

import (
	"encoding/json"
	"fmt"
	"io"
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
