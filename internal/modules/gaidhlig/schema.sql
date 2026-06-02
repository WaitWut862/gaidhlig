PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

-- WORDS
CREATE TABLE lemmas (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    word        TEXT NOT NULL,
    pos         TEXT,
    gender      TEXT,
    ipa         TEXT,
    etymology   TEXT,
    source      TEXT NOT NULL DEFAULT 'kaikki',
    verified    INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE senses (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    lemma_id    INTEGER NOT NULL REFERENCES lemmas(id) ON DELETE CASCADE,
    gloss       TEXT NOT NULL,
    tags        TEXT,
    topics      TEXT,
    example_gd  TEXT,
    example_en  TEXT
);

CREATE TABLE forms (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    lemma_id    INTEGER NOT NULL REFERENCES lemmas(id) ON DELETE CASCADE,
    form        TEXT NOT NULL,
    tags        TEXT
);

CREATE TABLE synonyms (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    lemma_id    INTEGER NOT NULL REFERENCES lemmas(id) ON DELETE CASCADE,
    sense_id    INTEGER REFERENCES senses(id) ON DELETE SET NULL,
    word        TEXT NOT NULL
);

CREATE TABLE derived (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    lemma_id    INTEGER NOT NULL REFERENCES lemmas(id) ON DELETE CASCADE,
    word        TEXT NOT NULL,
    english     TEXT
);

-- INDEXES
CREATE INDEX idx_lemmas_word ON lemmas(word);
CREATE INDEX idx_forms_form ON forms(form);
CREATE INDEX idx_senses_lemma ON senses(lemma_id);

-- GRAMMAR
CREATE TABLE rules (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT NOT NULL,
    text        TEXT NOT NULL,
    category    TEXT,
    difficulty  TEXT
);

CREATE TABLE rule_associations (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id      INTEGER NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    target_table TEXT NOT NULL,
    target_id    INTEGER NOT NULL
);

-- SENTENCES
CREATE TABLE sentences (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    hash       TEXT NOT NULL UNIQUE,
    text_gd    TEXT NOT NULL,
    text_en    TEXT,
    source     TEXT,
    difficulty TEXT,
    raw_conllu TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE sentence_analyses (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    sentence_id         INTEGER NOT NULL REFERENCES sentences(id) ON DELETE CASCADE,
    token               TEXT NOT NULL,
    lemma_text          TEXT,
    lemma_id            INTEGER REFERENCES lemmas(id) ON DELETE SET NULL,
    position            INTEGER NOT NULL,
    upos                TEXT,
    xpos                TEXT,
    feats               TEXT,
    dependency_head     INTEGER,
    dependency_relation TEXT,
    misc                TEXT,
    created_at          TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_sentence_analyses_sentence ON sentence_analyses(sentence_id);
CREATE INDEX idx_sentence_analyses_lemma ON sentence_analyses(lemma_id);

-- EXPRESSIONS
CREATE TABLE expressions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    text_gd    TEXT NOT NULL,
    text_en    TEXT NOT NULL,
    literal_en TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE expression_lemmas (
    expression_id INTEGER NOT NULL REFERENCES expressions(id) ON DELETE CASCADE,
    lemma_id      INTEGER NOT NULL REFERENCES lemmas(id) ON DELETE CASCADE,
    PRIMARY KEY (expression_id, lemma_id)
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
    target_id    INTEGER NOT NULL,
    note         TEXT
);

-- USERS
CREATE TABLE users (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    username   TEXT NOT NULL UNIQUE
);

CREATE TABLE user_progress (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_table TEXT NOT NULL,
    target_id    INTEGER NOT NULL,
    score        REAL NOT NULL DEFAULT 0 CHECK (score >= 0 AND score <= 100),
    seen_count   INTEGER NOT NULL DEFAULT 0,
    last_seen_at TEXT,
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, target_table, target_id)
);

CREATE INDEX idx_user_progress_user ON user_progress(user_id);
