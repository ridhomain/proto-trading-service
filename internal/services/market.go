package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ridhomain/proto-trading-service/internal/database"
	"github.com/ridhomain/proto-trading-service/internal/models"
	"github.com/ridhomain/proto-trading-service/pkg/logger"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type MarketService struct {
	db     *database.DB
	logger *zap.Logger
}

func NewMarketService(db *database.DB) *MarketService {
	return &MarketService{
		db:     db,
		logger: logger.With(zap.String("service", "market")),
	}
}

// GetBySymbol retrieves market data for a symbol
func (s *MarketService) GetBySymbol(ctx context.Context, symbol string, limit int) ([]models.MarketData, error) {
	query := `
		SELECT id, symbol, date, open, high, low, close, volume, source, created_at 
		FROM market_data 
		WHERE symbol = $1 
		ORDER BY date DESC 
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, symbol, limit)
	if err != nil {
		s.logger.Error("Failed to get market data by symbol",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	var results []models.MarketData
	for rows.Next() {
		var md models.MarketData
		err := rows.Scan(
			&md.ID, &md.Symbol, &md.Date, &md.Open, &md.High,
			&md.Low, &md.Close, &md.Volume, &md.Source, &md.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, md)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return results, nil
}

// GetBySymbolAndDateRange retrieves market data within a date range
func (s *MarketService) GetBySymbolAndDateRange(ctx context.Context, symbol string, startDate, endDate time.Time) ([]models.MarketData, error) {
	query := `
		SELECT id, symbol, date, open, high, low, close, volume, source, created_at 
		FROM market_data 
		WHERE symbol = $1 AND date >= $2 AND date <= $3
		ORDER BY date ASC
	`

	rows, err := s.db.Query(ctx, query, symbol, startDate, endDate)
	if err != nil {
		s.logger.Error("Failed to get market data by date range",
			zap.String("symbol", symbol),
			zap.Time("start_date", startDate),
			zap.Time("end_date", endDate),
			zap.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	// Use pgx.CollectRows for cleaner code
	results, err := pgx.CollectRows(rows, pgx.RowToStructByPos[models.MarketData])
	if err != nil {
		return nil, fmt.Errorf("failed to collect rows: %w", err)
	}

	return results, nil
}

// Create inserts new market data
func (s *MarketService) Create(ctx context.Context, data models.MarketData) (*models.MarketData, error) {
	query := `
		INSERT INTO market_data (symbol, date, open, high, low, close, volume, source) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
		RETURNING id, created_at
	`

	err := s.db.QueryRow(ctx, query,
		data.Symbol, data.Date, data.Open, data.High,
		data.Low, data.Close, data.Volume, data.Source,
	).Scan(&data.ID, &data.CreatedAt)

	if err != nil {
		s.logger.Error("Failed to create market data",
			zap.String("symbol", data.Symbol),
			zap.Error(err),
		)
		return nil, err
	}

	return &data, nil
}

// BulkCreate efficiently inserts multiple market data records using COPY
func (s *MarketService) BulkCreate(ctx context.Context, dataList []models.MarketData) error {
	if len(dataList) == 0 {
		return nil
	}

	// Prepare data for COPY
	rows := make([][]interface{}, len(dataList))
	for i, data := range dataList {
		rows[i] = []interface{}{
			data.Symbol,
			data.Date,
			data.Open,
			data.High,
			data.Low,
			data.Close,
			data.Volume,
			data.Source,
		}
	}

	// Use COPY for bulk insert - much faster than individual INSERTs
	copyCount, err := s.db.CopyFrom(
		ctx,
		pgx.Identifier{"market_data"},
		[]string{"symbol", "date", "open", "high", "low", "close", "volume", "source"},
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		s.logger.Error("Failed to bulk create market data",
			zap.Int("count", len(dataList)),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("Bulk created market data",
		zap.Int64("inserted", copyCount),
		zap.Int("requested", len(dataList)),
	)

	return nil
}

// BulkCreateWithConflict inserts with conflict handling
func (s *MarketService) BulkCreateWithConflict(ctx context.Context, dataList []models.MarketData) error {
	if len(dataList) == 0 {
		return nil
	}

	// Use transaction with batch for conflict handling
	err := s.db.Transaction(ctx, func(tx pgx.Tx) error {
		batch := &pgx.Batch{}

		query := `
			INSERT INTO market_data (symbol, date, open, high, low, close, volume, source) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
			ON CONFLICT (symbol, date, source) DO UPDATE SET
				open = EXCLUDED.open,
				high = EXCLUDED.high,
				low = EXCLUDED.low,
				close = EXCLUDED.close,
				volume = EXCLUDED.volume
		`

		for _, data := range dataList {
			batch.Queue(query,
				data.Symbol, data.Date, data.Open, data.High,
				data.Low, data.Close, data.Volume, data.Source,
			)
		}

		br := tx.SendBatch(ctx, batch)
		defer br.Close()

		// Execute all queries
		for i := 0; i < batch.Len(); i++ {
			if _, err := br.Exec(); err != nil {
				return fmt.Errorf("failed to execute batch item %d: %w", i, err)
			}
		}

		return nil
	})

	if err != nil {
		s.logger.Error("Failed to bulk create with conflict handling",
			zap.Int("count", len(dataList)),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// Delete removes market data by symbol
func (s *MarketService) Delete(ctx context.Context, symbol string) error {
	query := `DELETE FROM market_data WHERE symbol = $1`

	cmdTag, err := s.db.Exec(ctx, query, symbol)
	if err != nil {
		s.logger.Error("Failed to delete market data",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("Deleted market data",
		zap.String("symbol", symbol),
		zap.Int64("rows_affected", cmdTag.RowsAffected()),
	)

	return nil
}

// GetLatestBySymbol gets the most recent data point for a symbol
func (s *MarketService) GetLatestBySymbol(ctx context.Context, symbol string) (*models.MarketData, error) {
	query := `
		SELECT id, symbol, date, open, high, low, close, volume, source, created_at 
		FROM market_data 
		WHERE symbol = $1 
		ORDER BY date DESC 
		LIMIT 1
	`

	var result models.MarketData
	err := s.db.QueryRow(ctx, query, symbol).Scan(
		&result.ID, &result.Symbol, &result.Date, &result.Open, &result.High,
		&result.Low, &result.Close, &result.Volume, &result.Source, &result.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		s.logger.Error("Failed to get latest market data",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		return nil, err
	}

	return &result, nil
}

// GetSymbols returns all unique symbols in the database
func (s *MarketService) GetSymbols(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT symbol FROM market_data ORDER BY symbol`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		s.logger.Error("Failed to get symbols", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}

	return symbols, nil
}

// HealthCheck verifies the service is working
func (s *MarketService) HealthCheck(ctx context.Context) error {
	return s.db.HealthCheck(ctx)
}
