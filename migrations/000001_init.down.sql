-- Удаление триггеров
DROP TRIGGER IF EXISTS update_secrets_updated_at ON secrets;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Удаление функции
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Удаление индексов
DROP INDEX IF EXISTS idx_refresh_tokens_expires;
DROP INDEX IF EXISTS idx_refresh_tokens_user;
DROP INDEX IF EXISTS idx_secrets_user_deleted;
DROP INDEX IF EXISTS idx_secrets_user_updated;
DROP INDEX IF EXISTS idx_secrets_user_id;

-- Удаление таблиц
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS secrets;
DROP TABLE IF EXISTS users;
