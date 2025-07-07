package services

import (
	"context"
	"fmt"

	"github.com/ridhomain/proto-trading-service/internal/database"
	"github.com/ridhomain/proto-trading-service/pkg/logger"

	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

type UserPreferences struct {
	UserID          string   `json:"user_id" db:"user_id"`
	Email           string   `json:"email" db:"email"`
	DefaultSource   string   `json:"default_source" db:"default_source"`
	SelectedSymbols []string `json:"selected_symbols" db:"selected_symbols"`
	Watchlist       []string `json:"watchlist" db:"watchlist"`
	CreatedAt       string   `json:"created_at" db:"created_at"`
	UpdatedAt       string   `json:"updated_at" db:"updated_at"`
}

type UserService struct {
	db     *database.DB
	logger *zap.Logger
}

func NewUserService(db *database.DB) *UserService {
	return &UserService{
		db:     db,
		logger: logger.With(zap.String("service", "user")),
	}
}

// GetOrCreatePreferences gets user preferences or creates default ones
func (s *UserService) GetOrCreatePreferences(ctx context.Context, userID, email string) (*UserPreferences, error) {
	// Try to get existing preferences
	prefs, err := s.GetPreferences(ctx, userID)
	if err == nil && prefs != nil {
		return prefs, nil
	}

	// Create default preferences
	if err == pgx.ErrNoRows || prefs == nil {
		defaultPrefs := &UserPreferences{
			UserID:          userID,
			Email:           email,
			DefaultSource:   "yahoo",
			SelectedSymbols: []string{"BBCA.JK", "BBRI.JK", "TLKM.JK"},
			Watchlist:       []string{"BBCA.JK", "BBRI.JK", "TLKM.JK", "ASII.JK"},
		}

		err = s.CreatePreferences(ctx, defaultPrefs)
		if err != nil {
			return nil, fmt.Errorf("failed to create preferences: %w", err)
		}

		return defaultPrefs, nil
	}

	return nil, err
}

// GetPreferences retrieves user preferences
func (s *UserService) GetPreferences(ctx context.Context, userID string) (*UserPreferences, error) {
	query := `
		SELECT user_id, email, default_source, selected_symbols, watchlist, created_at, updated_at
		FROM user_preferences
		WHERE user_id = $1
	`

	var prefs UserPreferences
	err := s.db.QueryRow(ctx, query, userID).Scan(
		&prefs.UserID,
		&prefs.Email,
		&prefs.DefaultSource,
		pq.Array(&prefs.SelectedSymbols),
		pq.Array(&prefs.Watchlist),
		&prefs.CreatedAt,
		&prefs.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}
		s.logger.Error("Failed to get user preferences",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}

	return &prefs, nil
}

// CreatePreferences creates new user preferences
func (s *UserService) CreatePreferences(ctx context.Context, prefs *UserPreferences) error {
	query := `
		INSERT INTO user_preferences (user_id, email, default_source, selected_symbols, watchlist)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			email = EXCLUDED.email,
			updated_at = CURRENT_TIMESTAMP
		RETURNING created_at, updated_at
	`

	err := s.db.QueryRow(ctx, query,
		prefs.UserID,
		prefs.Email,
		prefs.DefaultSource,
		pq.Array(prefs.SelectedSymbols),
		pq.Array(prefs.Watchlist),
	).Scan(&prefs.CreatedAt, &prefs.UpdatedAt)

	if err != nil {
		s.logger.Error("Failed to create user preferences",
			zap.String("user_id", prefs.UserID),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// UpdatePreferences updates user preferences
func (s *UserService) UpdatePreferences(ctx context.Context, userID string, updates map[string]interface{}) error {
	// Build dynamic update query
	query := "UPDATE user_preferences SET "
	args := []interface{}{}
	argCount := 1

	for key, value := range updates {
		if argCount > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", key, argCount)
		args = append(args, value)
		argCount++
	}

	query += fmt.Sprintf(" WHERE user_id = $%d", argCount)
	args = append(args, userID)

	_, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		s.logger.Error("Failed to update user preferences",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// AddToWatchlist adds a symbol to user's watchlist
func (s *UserService) AddToWatchlist(ctx context.Context, userID, symbol string) error {
	query := `
		UPDATE user_preferences 
		SET watchlist = array_append(watchlist, $2)
		WHERE user_id = $1 AND NOT ($2 = ANY(watchlist))
	`

	_, err := s.db.Exec(ctx, query, userID, symbol)
	if err != nil {
		s.logger.Error("Failed to add to watchlist",
			zap.String("user_id", userID),
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// RemoveFromWatchlist removes a symbol from user's watchlist
func (s *UserService) RemoveFromWatchlist(ctx context.Context, userID, symbol string) error {
	query := `
		UPDATE user_preferences 
		SET watchlist = array_remove(watchlist, $2)
		WHERE user_id = $1
	`

	_, err := s.db.Exec(ctx, query, userID, symbol)
	if err != nil {
		s.logger.Error("Failed to remove from watchlist",
			zap.String("user_id", userID),
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		return err
	}

	return nil
}
