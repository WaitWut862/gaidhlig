PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

-- WORDS

CREATE TABLE words (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    lemma TEXT NOT NULL
);

CREATE TABLE word_associations (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    word_id      INTEGER NOT NULL REFERENCES words(id) ON DELETE CASCADE,
    target_table TEXT NOT NULL,
    target_id    INTEGER NOT NULL
);

CREATE TABLE spellings (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    target_table TEXT NOT NULL,
    target_id    INTEGER NOT NULL,
    form         TEXT NOT NULL
);

CREATE TABLE pronunciations (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    word_id INTEGER NOT NULL REFERENCES words(id) ON DELETE CASCADE,
    ipa     TEXT NOT NULL
);

CREATE TABLE definitions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    target_table TEXT NOT NULL,
    target_id    INTEGER NOT NULL,
    text         TEXT NOT NULL,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

-- RULES

CREATE TABLE rules (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    text TEXT NOT NULL
);

CREATE TABLE rule_associations (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id      INTEGER NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    target_table TEXT NOT NULL,
    target_id    INTEGER NOT NULL
);

-- METADATA

CREATE TABLE tags (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    category TEXT NOT NULL,
    label    TEXT NOT NULL,
    UNIQUE(category, label)
);

CREATE TABLE tag_associations (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    tag_id       INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    target_table TEXT NOT NULL,
    target_id    INTEGER NOT NULL
);

-- RESOURCES

CREATE TABLE sentences (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    hash       TEXT NOT NULL UNIQUE,
    text       TEXT NOT NULL,
    raw_conllu TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE sentence_analyses (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    sentence_id         INTEGER NOT NULL REFERENCES sentences(id) ON DELETE CASCADE,
    token               TEXT NOT NULL,
    lemma_text          TEXT,
    word_id             INTEGER REFERENCES words(id) ON DELETE SET NULL,
    position            INTEGER NOT NULL,
    upos                TEXT,
    xpos                TEXT,
    feats               TEXT,
    dependency_head     INTEGER,
    dependency_relation TEXT,
    misc                TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE expressions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    text       TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- USER

CREATE TABLE users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    username   TEXT NOT NULL UNIQUE,
);

CREATE TABLE user_progress (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_table TEXT NOT NULL,
    target_id    INTEGER NOT NULL,
    score        REAL NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
