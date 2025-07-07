-- User preferences table to store trading-specific user data
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id VARCHAR(255) PRIMARY KEY,  -- Kratos identity ID
    email VARCHAR(255) NOT NULL,
    default_source VARCHAR(50) DEFAULT 'yahoo',
    selected_symbols TEXT[] DEFAULT '{}',
    watchlist TEXT[] DEFAULT '{}', 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add trigger to update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_user_preferences_updated_at 
BEFORE UPDATE ON user_preferences 
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();

-- Index for faster lookups
CREATE INDEX idx_user_preferences_email ON user_preferences(email);