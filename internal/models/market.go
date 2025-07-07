package models

import "time"

// MarketData represents stock market data
type MarketData struct {
	ID        int64     `json:"id" db:"id"`
	Symbol    string    `json:"symbol" db:"symbol" binding:"required"`
	Date      time.Time `json:"date" db:"date" binding:"required"`
	Open      float64   `json:"open" db:"open" binding:"required,min=0"`
	High      float64   `json:"high" db:"high" binding:"required,min=0"`
	Low       float64   `json:"low" db:"low" binding:"required,min=0"`
	Close     float64   `json:"close" db:"close" binding:"required,min=0"`
	Volume    int64     `json:"volume" db:"volume" binding:"required,min=0"`
	Source    string    `json:"source" db:"source" binding:"required,oneof=yahoo mirae manual"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// BulkCreateRequest represents a request to create multiple market data records
type BulkCreateRequest struct {
	Data []MarketData `json:"data" binding:"required,dive"`
}

// YahooQuote represents data from Yahoo Finance API
type YahooQuote struct {
	Symbol   string    `json:"symbol"`
	Date     time.Time `json:"date"`
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Close    float64   `json:"close"`
	Volume   int64     `json:"volume"`
	AdjClose float64   `json:"adjClose"`
}

// CSVUploadResponse represents the response for CSV upload
type CSVUploadResponse struct {
	Message      string   `json:"message"`
	RowsImported int      `json:"rows_imported"`
	RowsSkipped  int      `json:"rows_skipped"`
	Errors       []string `json:"errors,omitempty"`
}
