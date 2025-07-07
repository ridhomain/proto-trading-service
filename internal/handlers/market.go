package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ridhomain/proto-trading-service/internal/middleware"
	"github.com/ridhomain/proto-trading-service/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MarketDataResponse represents the response for market data queries
type MarketDataResponse struct {
	Symbol string              `json:"symbol"`
	Count  int                 `json:"count"`
	Data   []models.MarketData `json:"data"`
}

// GetMarketData retrieves market data with query parameters
func (h *Handler) GetMarketData(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "symbol parameter is required",
		})
		return
	}

	// Parse limit with default
	limit := 30
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	// Get user preferences for default source
	userID := middleware.GetUserID(c)
	ctx := c.Request.Context()

	// Get user's preferred data source
	source := c.Query("source")
	if source == "" && userID != "" {
		prefs, err := h.userService.GetPreferences(ctx, userID)
		if err == nil && prefs != nil {
			source = prefs.DefaultSource
		}
	}

	data, err := h.marketService.GetBySymbol(ctx, symbol, limit)
	if err != nil {
		h.logger.Error("Failed to fetch market data",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to fetch data",
		})
		return
	}

	c.JSON(http.StatusOK, MarketDataResponse{
		Symbol: symbol,
		Count:  len(data),
		Data:   data,
	})
}

// GetMarketDataBySymbol retrieves market data for a specific symbol
func (h *Handler) GetMarketDataBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")

	// Parse date range if provided
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	ctx := c.Request.Context()

	if startDateStr != "" && endDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Invalid start_date format. Use YYYY-MM-DD",
			})
			return
		}

		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Invalid end_date format. Use YYYY-MM-DD",
			})
			return
		}

		data, err := h.marketService.GetBySymbolAndDateRange(ctx, symbol, startDate, endDate)
		if err != nil {
			h.logger.Error("Failed to fetch market data by date range",
				zap.String("symbol", symbol),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "Failed to fetch data",
			})
			return
		}

		c.JSON(http.StatusOK, MarketDataResponse{
			Symbol: symbol,
			Count:  len(data),
			Data:   data,
		})
		return
	}

	// Default: get latest 30 days
	data, err := h.marketService.GetBySymbol(ctx, symbol, 30)
	if err != nil {
		h.logger.Error("Failed to fetch market data",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to fetch data",
		})
		return
	}

	c.JSON(http.StatusOK, MarketDataResponse{
		Symbol: symbol,
		Count:  len(data),
		Data:   data,
	})
}

// CreateMarketData creates a new market data entry
func (h *Handler) CreateMarketData(c *gin.Context) {
	var data models.MarketData

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	result, err := h.marketService.Create(ctx, data)
	if err != nil {
		h.logger.Error("Failed to create market data",
			zap.String("symbol", data.Symbol),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to create data",
		})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// BulkCreateMarketData creates multiple market data entries
func (h *Handler) BulkCreateMarketData(c *gin.Context) {
	var req models.BulkCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	err := h.marketService.BulkCreateWithConflict(ctx, req.Data)
	if err != nil {
		h.logger.Error("Failed to bulk create market data",
			zap.Int("count", len(req.Data)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to bulk create data",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Data created successfully",
		"count":   len(req.Data),
	})
}

// FetchYahooData fetches data from Yahoo Finance (mock for now)
func (h *Handler) FetchYahooData(c *gin.Context) {
	symbol := c.Param("symbol")

	// Optional query parameters
	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	h.logger.Info("Fetching Yahoo Finance data",
		zap.String("symbol", symbol),
		zap.Int("days", days),
	)

	// TODO: Implement actual Yahoo Finance API call
	// For now, generate mock data
	endDate := time.Now()
	mockData := make([]models.MarketData, days)

	for i := 0; i < days; i++ {
		date := endDate.AddDate(0, 0, -i)
		mockData[i] = models.MarketData{
			Symbol: symbol,
			Date:   date,
			Open:   8500 + float64(i*10),
			High:   8600 + float64(i*10),
			Low:    8400 + float64(i*10),
			Close:  8550 + float64(i*10),
			Volume: 12500000 + int64(i*100000),
			Source: "yahoo",
		}
	}

	ctx := c.Request.Context()
	err := h.marketService.BulkCreate(ctx, mockData)
	if err != nil {
		h.logger.Error("Failed to save Yahoo data",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to save data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data fetched successfully",
		"symbol":  symbol,
		"count":   len(mockData),
		"source":  "yahoo",
	})
}

// DeleteMarketData deletes market data for a symbol
func (h *Handler) DeleteMarketData(c *gin.Context) {
	symbol := c.Param("symbol")

	ctx := c.Request.Context()
	err := h.marketService.Delete(ctx, symbol)
	if err != nil {
		h.logger.Error("Failed to delete market data",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to delete data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data deleted successfully",
		"symbol":  symbol,
	})
}

// UploadCSV handles CSV file uploads
func (h *Handler) UploadCSV(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "No file uploaded",
		})
		return
	}
	defer file.Close()

	h.logger.Info("Processing CSV upload",
		zap.String("filename", header.Filename),
		zap.Int64("size", header.Size),
	)

	// Parse CSV
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to parse CSV",
			Message: err.Error(),
		})
		return
	}

	if len(records) < 2 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "CSV file is empty or has no data rows",
		})
		return
	}

	// Process records (skip header)
	var marketData []models.MarketData
	var errors []string

	for i, record := range records[1:] {
		if len(record) < 7 {
			errors = append(errors, fmt.Sprintf("Row %d: insufficient columns", i+2))
			continue
		}

		// Parse date
		date, err := time.Parse("2006-01-02", record[1])
		if err != nil {
			errors = append(errors, fmt.Sprintf("Row %d: invalid date format", i+2))
			continue
		}

		// Parse numeric values
		open, _ := strconv.ParseFloat(record[2], 64)
		high, _ := strconv.ParseFloat(record[3], 64)
		low, _ := strconv.ParseFloat(record[4], 64)
		close, _ := strconv.ParseFloat(record[5], 64)
		volume, _ := strconv.ParseInt(record[6], 10, 64)

		marketData = append(marketData, models.MarketData{
			Symbol: record[0],
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
			Source: "mirae",
		})
	}

	// Bulk insert
	ctx := c.Request.Context()
	if len(marketData) > 0 {
		err = h.marketService.BulkCreateWithConflict(ctx, marketData)
		if err != nil {
			h.logger.Error("Failed to import CSV data",
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "Failed to import data",
			})
			return
		}
	}

	response := models.CSVUploadResponse{
		Message:      "CSV processed successfully",
		RowsImported: len(marketData),
		RowsSkipped:  len(records) - 1 - len(marketData),
		Errors:       errors,
	}

	c.JSON(http.StatusOK, response)
}
