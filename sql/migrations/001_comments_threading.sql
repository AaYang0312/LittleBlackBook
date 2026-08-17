-- 老库升级：评论楼中楼两级回复（parent_id / reply_count / reply_to + idx_parent 索引）。
-- 本文件可重复执行（幂等）。MySQL 8.0 不支持 ADD COLUMN IF NOT EXISTS /
-- ADD INDEX IF NOT EXISTS（MariaDB 语法，MySQL 8.0.46 实测报 ERROR 1064），
-- 因此通过 information_schema 检查列/索引是否存在，仅在缺失时动态执行 ALTER。

-- 列: parent_id
SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND COLUMN_NAME = 'parent_id');
SET @s := IF(@c = 0, 'ALTER TABLE comments ADD COLUMN parent_id BIGINT NOT NULL DEFAULT 0', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- 列: reply_count
SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND COLUMN_NAME = 'reply_count');
SET @s := IF(@c = 0, 'ALTER TABLE comments ADD COLUMN reply_count BIGINT NOT NULL DEFAULT 0', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- 列: reply_to
SET @c := (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND COLUMN_NAME = 'reply_to');
SET @s := IF(@c = 0, 'ALTER TABLE comments ADD COLUMN reply_to BIGINT NOT NULL DEFAULT 0', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- 索引: idx_parent (parent_id, id)
SET @c := (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'comments' AND INDEX_NAME = 'idx_parent');
SET @s := IF(@c = 0, 'ALTER TABLE comments ADD INDEX idx_parent (parent_id, id)', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
