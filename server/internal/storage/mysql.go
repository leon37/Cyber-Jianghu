package storage

import (
	"Cyber-Jianghu/server/internal/config"
	"Cyber-Jianghu/server/internal/models"
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MySQLStore struct {
	db *gorm.DB
}

func NewMySQLStore(cfg config.MySQLConfig) (*MySQLStore, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Auto migrate tables (will be expanded in later phases)
	if err := db.AutoMigrate(); err != nil {
		return nil, err
	}

	return &MySQLStore{db: db}, nil
}

func (s *MySQLStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *MySQLStore) GetDB() *gorm.DB {
	return s.db
}

// Transaction helper
func (s *MySQLStore) WithTx(fn func(*gorm.DB) error) error {
	return s.db.Transaction(fn)
}

// SaveStory saves a story to the database
func (s *MySQLStore) SaveStory(ctx context.Context, story *models.Story) error {
	return s.db.WithContext(ctx).Save(story).Error
}

// GetStory retrieves a story by ID
func (s *MySQLStore) GetStory(ctx context.Context, storyID string) (*models.Story, error) {
	var story models.Story
	err := s.db.WithContext(ctx).Where("id = ?", storyID).First(&story).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Story not found is not an error
		}
		return nil, err
	}
	return &story, nil
}

// UpdateStory updates a story in the database
func (s *MySQLStore) UpdateStory(ctx context.Context, storyID string, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&models.Story{}).Where("id = ?", storyID).Updates(updates).Error
}

// DeleteStory deletes a story (soft delete)
func (s *MySQLStore) DeleteStory(ctx context.Context, storyID string) error {
	return s.db.WithContext(ctx).Delete(&models.Story{}, "id = ?", storyID).Error
}

// SaveDecision saves a story decision
func (s *MySQLStore) SaveDecision(ctx context.Context, decision *models.StoryDecision) error {
	return s.db.WithContext(ctx).Save(decision).Error
}

// GetDecisions retrieves decisions for a story
func (s *MySQLStore) GetDecisions(ctx context.Context, storyID string, limit int) ([]models.StoryDecision, error) {
	var decisions []models.StoryDecision
	query := s.db.WithContext(ctx).Where("story_id = ?", storyID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&decisions).Error
	return decisions, err
}

// SaveStoryMemory saves a story memory point
func (s *MySQLStore) SaveStoryMemory(ctx context.Context, memory *models.StoryMemory) error {
	return s.db.WithContext(ctx).Save(memory).Error
}

// GetMemories retrieves memories for a story
func (s *MySQLStore) GetMemories(ctx context.Context, storyID string, memoryType string, limit int) ([]models.StoryMemory, error) {
	var memories []models.StoryMemory
	query := s.db.WithContext(ctx).Where("story_id = ?", storyID)
	if memoryType != "" {
		query = query.Where("type = ?", memoryType)
	}
	query = query.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&memories).Error
	return memories, err
}

// CleanExpiredMemories removes expired memories
func (s *MySQLStore) CleanExpiredMemories(ctx context.Context) (int64, error) {
	result := s.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&models.StoryMemory{})
	return result.RowsAffected, result.Error
}

// GetStoriesByStatus retrieves stories by status
func (s *MySQLStore) GetStoriesByStatus(ctx context.Context, status string, limit int) ([]models.Story, error) {
	var stories []models.Story
	query := s.db.WithContext(ctx).Where("status = ?", status).Order("updated_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&stories).Error
	return stories, err
}

// GetRecentStories retrieves recent stories
func (s *MySQLStore) GetRecentStories(ctx context.Context, limit int) ([]models.Story, error) {
	var stories []models.Story
	query := s.db.WithContext(ctx).Order("updated_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&stories).Error
	return stories, err
}

// GetStoryCount returns the count of stories
func (s *MySQLStore) GetStoryCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&models.Story{}).Count(&count).Error
	return count, err
}

// GetMemoryCount returns the count of memories
func (s *MySQLStore) GetMemoryCount(ctx context.Context, storyID string) (int64, error) {
	var count int64
	query := s.db.WithContext(ctx).Model(&models.StoryMemory{}).Where("story_id = ?", storyID)
	if storyID == "" {
		query = s.db.WithContext(ctx).Model(&models.StoryMemory{})
	}
	err := query.Count(&count).Error
	return count, err
}
