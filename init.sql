-- Создание отдельного пользователя для приложения (опционально)
CREATE USER api_user WITH PASSWORD 'api_password';
ALTER USER api_user SET client_encoding TO 'utf8';
ALTER USER api_user SET default_transaction_isolation TO 'read committed';
ALTER USER api_user SET timezone TO 'UTC';

-- Предоставление прав
GRANT ALL PRIVILEGES ON DATABASE simple_api TO api_user;
GRANT CREATE ON DATABASE simple_api TO api_user;

-- Создание расширений (если нужно)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";