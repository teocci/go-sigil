// Package db manages SQLite connection setup and schema migrations.
package db

// V1DDL is the complete schema for version 1.
// Applied atomically when PRAGMA user_version = 0.
const V1DDL = `
CREATE TABLE files (
  path         TEXT PRIMARY KEY,
  blob_sha     TEXT,
  mtime        TEXT,
  size         INTEGER,
  last_indexed TEXT NOT NULL
);

CREATE TABLE symbols (
  id                  TEXT PRIMARY KEY,
  kind                TEXT NOT NULL,
  name                TEXT NOT NULL,
  qualified_name      TEXT NOT NULL,
  language            TEXT NOT NULL,
  file                TEXT NOT NULL,
  package_root        TEXT,
  byte_start          INTEGER,
  byte_end            INTEGER,
  line_start          INTEGER NOT NULL,
  line_end            INTEGER NOT NULL,
  signature           TEXT,
  summary             TEXT,
  tags                TEXT,
  parent_id           TEXT,
  children            TEXT,
  depth               INTEGER NOT NULL DEFAULT 0,
  imports             TEXT,
  content_hash        TEXT,
  possible_unresolved INTEGER NOT NULL DEFAULT 0,
  untracked           INTEGER NOT NULL DEFAULT 0,
  embedding           BLOB,
  indexed_at          TEXT NOT NULL
);

CREATE TABLE call_edges (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  caller_id       TEXT NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
  callee_id       TEXT,
  raw_expression  TEXT,
  confidence      TEXT NOT NULL DEFAULT 'static'
);

-- Deduplicate resolved edges (callee known)
CREATE UNIQUE INDEX idx_edges_resolved ON call_edges(caller_id, callee_id)
  WHERE callee_id IS NOT NULL;
-- Deduplicate unresolved edges (raw expression only)
CREATE UNIQUE INDEX idx_edges_unresolved ON call_edges(caller_id, raw_expression)
  WHERE callee_id IS NULL;

CREATE VIRTUAL TABLE symbols_fts USING fts5(
  name,
  qualified_name,
  signature,
  summary,
  content=symbols,
  content_rowid=rowid
);

CREATE TABLE savings_log (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id   TEXT NOT NULL,
  tool_name    TEXT NOT NULL,
  call_at      TEXT NOT NULL,
  timing_ms    REAL,
  tokens_saved INTEGER
);

CREATE INDEX idx_symbols_name   ON symbols(name);
CREATE INDEX idx_symbols_file   ON symbols(file);
CREATE INDEX idx_symbols_kind   ON symbols(kind);
CREATE INDEX idx_symbols_parent ON symbols(parent_id);
CREATE INDEX idx_symbols_pkg    ON symbols(package_root);
CREATE INDEX idx_edges_caller   ON call_edges(caller_id);
CREATE INDEX idx_edges_callee   ON call_edges(callee_id);
CREATE INDEX idx_edges_conf     ON call_edges(confidence);
CREATE INDEX idx_savings_sess   ON savings_log(session_id);

CREATE TRIGGER symbols_ai AFTER INSERT ON symbols BEGIN
  INSERT INTO symbols_fts(rowid, name, qualified_name, signature, summary)
  VALUES (new.rowid, new.name, new.qualified_name, new.signature, new.summary);
END;

CREATE TRIGGER symbols_ad AFTER DELETE ON symbols BEGIN
  INSERT INTO symbols_fts(symbols_fts, rowid, name, qualified_name, signature, summary)
  VALUES ('delete', old.rowid, old.name, old.qualified_name, old.signature, old.summary);
END;

CREATE TRIGGER symbols_au AFTER UPDATE ON symbols BEGIN
  INSERT INTO symbols_fts(symbols_fts, rowid, name, qualified_name, signature, summary)
  VALUES ('delete', old.rowid, old.name, old.qualified_name, old.signature, old.summary);
  INSERT INTO symbols_fts(rowid, name, qualified_name, signature, summary)
  VALUES (new.rowid, new.name, new.qualified_name, new.signature, new.summary);
END;
`
