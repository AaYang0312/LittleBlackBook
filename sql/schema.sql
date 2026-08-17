CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(128) NOT NULL,
    nickname VARCHAR(64) NOT NULL DEFAULT '',
    avatar_url VARCHAR(255) NOT NULL DEFAULT '',
    bio VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS notes (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    title VARCHAR(128) NOT NULL,
    content TEXT,
    cover_url VARCHAR(255) NOT NULL DEFAULT '',
    images JSON,
    like_count BIGINT NOT NULL DEFAULT 0,
    collect_count BIGINT NOT NULL DEFAULT 0,
    comment_count BIGINT NOT NULL DEFAULT 0,
    status TINYINT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_created (user_id, created_at)
);

CREATE TABLE IF NOT EXISTS likes (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    note_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_note (user_id, note_id)
);

CREATE TABLE IF NOT EXISTS collects (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    note_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_note (user_id, note_id)
);

CREATE TABLE IF NOT EXISTS comments (
    id BIGINT PRIMARY KEY,
    note_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    content VARCHAR(512) NOT NULL,
    parent_id BIGINT NOT NULL DEFAULT 0,
    reply_count BIGINT NOT NULL DEFAULT 0,
    reply_to BIGINT NOT NULL DEFAULT 0,
    status TINYINT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_note_created (note_id, created_at),
    INDEX idx_parent (parent_id, id)
);

CREATE TABLE IF NOT EXISTS follows (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    follower_id BIGINT NOT NULL,
    followee_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_pair (follower_id, followee_id),
    INDEX idx_followee (followee_id)
);