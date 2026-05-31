PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE words (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    text TEXT NOT NULL,
    definition TEXT NOT NULL,
    cefr TEXT NOT NULL
);

CREATE TABLE rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    text TEXT NOT NULL,
    cefr TEXT NOT NULL
);

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL
);

CREATE TABLE user_progress (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_table TEXT NOT NULL,
    target_id    INTEGER NOT NULL,
    score        REAL NOT NULL DEFAULT 0,
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

