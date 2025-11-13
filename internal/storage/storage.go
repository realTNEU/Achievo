package storage

import (
	"context"
	"fmt"
	"time"

	"achievo/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Storage provides database operations
type Storage struct {
	client   *mongo.Client
	database *mongo.Database
	ctx      context.Context
	cancel   context.CancelFunc
}

// New creates a new Storage instance
func New(uri, database string, timeout int) (*Storage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(database)

	storage := &Storage{
		client:   client,
		database: db,
		ctx:      context.Background(),
		cancel:   cancel,
	}

	// Create indexes
	if err := storage.createIndexes(); err != nil {
		storage.Close()
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return storage, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.client != nil {
		return s.client.Disconnect(context.Background())
	}
	return nil
}

// createIndexes creates necessary database indexes
func (s *Storage) createIndexes() error {
	// Games collection
	games := s.database.Collection("games")
	_, err := games.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "steam_app_id", Value: 1}},
		Options: options.Index().SetSparse(true).SetName("steam_app_id_idx"),
	})
	if err != nil {
		return err
	}

	// Sessions collection
	sessions := s.database.Collection("sessions")
	_, err = sessions.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "game_id", Value: 1}, {Key: "start_time", Value: -1}},
		Options: options.Index().SetName("game_id_start_time_idx"),
	})
	if err != nil {
		return err
	}

	// Events collection
	events := s.database.Collection("events")
	_, err = events.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "game_id", Value: 1}, {Key: "timestamp", Value: -1}},
			Options: options.Index().SetName("game_id_timestamp_idx"),
		},
		{
			Keys:    bson.D{{Key: "synced", Value: 1}, {Key: "timestamp", Value: 1}},
			Options: options.Index().SetName("synced_timestamp_idx"),
		},
	})
	if err != nil {
		return err
	}

	// Achievement progress collection
	progress := s.database.Collection("achievement_progress")
	_, err = progress.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "game_id", Value: 1}, {Key: "achievement_id", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("game_id_achievement_id_idx"),
	})
	if err != nil {
		return err
	}

	return nil
}

// SaveGame saves or updates a game
func (s *Storage) SaveGame(game *models.Game) error {
	collection := s.database.Collection("games")

	filter := bson.M{"_id": game.ID}
	update := bson.M{"$set": game}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(s.ctx, filter, update, opts)
	return err
}

// GetGame retrieves a game by ID
func (s *Storage) GetGame(gameID string) (*models.Game, error) {
	collection := s.database.Collection("games")

	var game models.Game
	err := collection.FindOne(s.ctx, bson.M{"_id": gameID}).Decode(&game)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &game, nil
}

// GetGameBySteamAppID retrieves a game by Steam App ID
func (s *Storage) GetGameBySteamAppID(appID string) (*models.Game, error) {
	collection := s.database.Collection("games")

	var game models.Game
	err := collection.FindOne(s.ctx, bson.M{"steam_app_id": appID}).Decode(&game)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &game, nil
}

// SaveSession saves or updates a session
func (s *Storage) SaveSession(session *models.Session) error {
	collection := s.database.Collection("sessions")

	filter := bson.M{"_id": session.ID}
	update := bson.M{"$set": session}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(s.ctx, filter, update, opts)
	return err
}

// GetActiveSession retrieves the active session for a game
func (s *Storage) GetActiveSession(gameID string) (*models.Session, error) {
	collection := s.database.Collection("sessions")

	var session models.Session
	err := collection.FindOne(s.ctx, bson.M{
		"game_id":   gameID,
		"is_active": true,
	}).Decode(&session)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// SaveAchievement saves or updates an achievement definition
func (s *Storage) SaveAchievement(achievement *models.Achievement) error {
	collection := s.database.Collection("achievements")

	filter := bson.M{"_id": achievement.ID}
	update := bson.M{"$set": achievement}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(s.ctx, filter, update, opts)
	return err
}

// GetAchievementsByGame retrieves all achievements for a game
func (s *Storage) GetAchievementsByGame(gameID string) ([]models.Achievement, error) {
	collection := s.database.Collection("achievements")

	cursor, err := collection.Find(s.ctx, bson.M{"game_id": gameID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var achievements []models.Achievement
	if err := cursor.All(s.ctx, &achievements); err != nil {
		return nil, err
	}
	return achievements, nil
}

// SaveAchievementProgress saves or updates achievement progress
func (s *Storage) SaveAchievementProgress(progress *models.AchievementProgress) error {
	collection := s.database.Collection("achievement_progress")

	filter := bson.M{"_id": progress.ID}
	update := bson.M{"$set": progress}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(s.ctx, filter, update, opts)
	return err
}

// GetAchievementProgress retrieves achievement progress
func (s *Storage) GetAchievementProgress(gameID, achievementID string) (*models.AchievementProgress, error) {
	collection := s.database.Collection("achievement_progress")

	var progress models.AchievementProgress
	err := collection.FindOne(s.ctx, bson.M{
		"game_id":        gameID,
		"achievement_id": achievementID,
	}).Decode(&progress)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &progress, nil
}

// GetAllProgress retrieves all achievement progress for a game
func (s *Storage) GetAllProgress(gameID string) ([]models.AchievementProgress, error) {
	collection := s.database.Collection("achievement_progress")

	cursor, err := collection.Find(s.ctx, bson.M{"game_id": gameID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var progress []models.AchievementProgress
	if err := cursor.All(s.ctx, &progress); err != nil {
		return nil, err
	}
	return progress, nil
}

// SaveStatistic saves or updates a statistic
func (s *Storage) SaveStatistic(stat *models.Statistic) error {
	collection := s.database.Collection("statistics")

	filter := bson.M{"_id": stat.ID}
	update := bson.M{"$set": stat}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(s.ctx, filter, update, opts)
	return err
}

// GetStatistic retrieves a statistic
func (s *Storage) GetStatistic(gameID, statName string) (*models.Statistic, error) {
	collection := s.database.Collection("statistics")

	var stat models.Statistic
	err := collection.FindOne(s.ctx, bson.M{
		"game_id": gameID,
		"name":    statName,
	}).Decode(&stat)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &stat, nil
}

// SaveEvent saves an event
func (s *Storage) SaveEvent(event *models.Event) error {
	collection := s.database.Collection("events")

	if event.ID == "" {
		event.ID = primitive.NewObjectID().Hex()
	}

	_, err := collection.InsertOne(s.ctx, event)
	return err
}

// GetUnsyncedEvents retrieves all unsynced events
func (s *Storage) GetUnsyncedEvents(limit int) ([]models.Event, error) {
	collection := s.database.Collection("events")

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := collection.Find(s.ctx, bson.M{"synced": false}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var events []models.Event
	if err := cursor.All(s.ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// MarkEventSynced marks an event as synced
func (s *Storage) MarkEventSynced(eventID string) error {
	collection := s.database.Collection("events")

	update := bson.M{
		"$set": bson.M{
			"synced":    true,
			"synced_at": time.Now(),
		},
	}
	_, err := collection.UpdateOne(s.ctx, bson.M{"_id": eventID}, update)
	return err
}

// UpsertAchievementDefs upserts multiple achievement definitions
func (s *Storage) UpsertAchievementDefs(achievements []models.Achievement) error {
	collection := s.database.Collection("achievements")

	for _, achievement := range achievements {
		filter := bson.M{"_id": achievement.ID}
		update := bson.M{"$set": achievement}
		opts := options.Update().SetUpsert(true)

		if _, err := collection.UpdateOne(s.ctx, filter, update, opts); err != nil {
			return fmt.Errorf("failed to upsert achievement %s: %w", achievement.ID, err)
		}
	}

	return nil
}

// GetAchievementsBySteamAppID retrieves achievements by Steam App ID
func (s *Storage) GetAchievementsBySteamAppID(appID string) ([]models.Achievement, error) {
	collection := s.database.Collection("achievements")

	cursor, err := collection.Find(s.ctx, bson.M{"steam_app_id": appID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.ctx)

	var achievements []models.Achievement
	if err := cursor.All(s.ctx, &achievements); err != nil {
		return nil, err
	}
	return achievements, nil
}
