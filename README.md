# Proto Trading Service

A high-performance trading data service built with Go, designed to fetch and store Indonesian stock market data from multiple sources.

## Features

- 🚀 High-performance REST API built with Gin
- 💾 PostgreSQL with pgx for optimal performance
- 📊 Support for Yahoo Finance data
- 📁 CSV upload support for Mirae Securities data
- 🔍 Structured logging with Zap
- ⚡ Bulk data operations using PostgreSQL COPY
- 🔧 Production-ready configuration with Viper
- 🐳 Docker support for easy deployment

## Tech Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: PostgreSQL with pgx/v5
- **Configuration**: Viper
- **Logging**: Uber's Zap
- **Container**: Docker & Docker Compose

## Quick Start

1. **Clone the repository**
```bash
git clone https://github.com/ridhomain/proto-trading-service.git
cd proto-trading-service
```

2. **Start the infrastructure**
```bash
make docker-up
```

3. **Run database migrations**
```bash
make migrate
```

4. **Install dependencies**
```bash
go mod download
```

5. **Run the service**
```bash
make run
```

The service will start on `http://localhost:8080`

## API Endpoints

### Health Check
```bash
GET /health
GET /ready
```

### Market Data
```bash
# Get market data
GET /api/v1/market-data?symbol=BBCA.JK&limit=30

# Get by symbol with date range
GET /api/v1/market-data/BBCA.JK?start_date=2025-01-01&end_date=2025-01-07

# Create single entry
POST /api/v1/market-data
{
  "symbol": "BBCA.JK",
  "date": "2025-01-07T00:00:00Z",
  "open": 8500,
  "high": 8600,
  "low": 8450,
  "close": 8550,
  "volume": 12500000,
  "source": "yahoo"
}

# Bulk create
POST /api/v1/market-data/bulk
{
  "data": [
    {
      "symbol": "BBCA.JK",
      "date": "2025-01-07T00:00:00Z",
      "open": 8500,
      "high": 8600,
      "low": 8450,
      "close": 8550,
      "volume": 12500000,
      "source": "yahoo"
    }
  ]
}

# Fetch from Yahoo Finance (mock)
POST /api/v1/market-data/yahoo/BBCA.JK?days=7

# Delete by symbol
DELETE /api/v1/market-data/BBCA.JK
```

### CSV Upload
```bash
# Upload Mirae Securities CSV
POST /api/v1/upload/csv
Content-Type: multipart/form-data
file: <your-csv-file>
```

CSV format:
```csv
Symbol,Date,Open,High,Low,Close,Volume
BBCA.JK,2025-01-07,8500,8600,8450,8550,12500000
```

## Project Structure

```
proto-trading-service/
├── cmd/server/          # Application entry point
├── internal/            # Private application code
│   ├── config/         # Configuration management
│   ├── database/       # Database connection and helpers
│   ├── handlers/       # HTTP handlers
│   ├── middleware/     # HTTP middleware
│   ├── models/         # Data models
│   └── services/       # Business logic
├── pkg/                # Public packages
│   └── logger/         # Logging utilities
├── migrations/         # Database migrations
├── docker-compose.yml  # Docker services
├── Makefile           # Build commands
└── go.mod             # Go dependencies
```

## Development

### Running Tests
```bash
make test
```

### Building Binary
```bash
make build
./bin/server
```

### Environment Variables

See `.env` file for all available configuration options.

## Indonesian Stock Symbols (Yahoo Finance)

- BBCA.JK - Bank Central Asia
- BBRI.JK - Bank Rakyat Indonesia
- BMRI.JK - Bank Mandiri
- TLKM.JK - Telkom Indonesia
- ASII.JK - Astra International

## Performance

- Bulk insert of 10,000 records: ~100ms using PostgreSQL COPY
- Connection pooling with configurable limits
- Efficient memory usage with pgx native driver

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.