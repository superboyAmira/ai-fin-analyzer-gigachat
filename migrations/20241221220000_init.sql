
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('receipt', 'statement', 'screenshot')),
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    file_url VARCHAR(500) NOT NULL,
    extracted_text TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_user_id ON documents(user_id);
CREATE INDEX idx_documents_created_at ON documents(created_at DESC);

-- Transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(15, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    description VARCHAR(500),
    category VARCHAR(20) NOT NULL CHECK (category IN ('food', 'transport', 'utilities', 'shopping', 'entertainment', 'healthcare', 'education', 'other')),
    llm_description TEXT,
    date TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_document_id ON transactions(document_id);
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_category ON transactions(category);
CREATE INDEX idx_transactions_date ON transactions(date DESC);

-- Recommendations table
CREATE TABLE IF NOT EXISTS recommendations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    potential_savings DECIMAL(15, 2) DEFAULT 0,
    source VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recommendations_transaction_id ON recommendations(transaction_id);
CREATE INDEX idx_recommendations_user_id ON recommendations(user_id);
CREATE INDEX idx_recommendations_potential_savings ON recommendations(potential_savings DESC);

-- Knowledge base table
CREATE TABLE IF NOT EXISTS knowledge_base (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(20) NOT NULL CHECK (type IN ('bank_tariff', 'gov_tariff', 'education')),
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    embedding REAL[], -- For pgvector: vector(1536) or similar
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_knowledge_base_type ON knowledge_base(type);
CREATE INDEX idx_knowledge_base_title ON knowledge_base USING gin(to_tsvector('russian', title));
CREATE INDEX idx_knowledge_base_content ON knowledge_base USING gin(to_tsvector('russian', content));
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS recommendations;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS knowledge_base;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd