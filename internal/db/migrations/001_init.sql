CREATE TABLE categories (
  id         INTEGER PRIMARY KEY,
  slug       TEXT NOT NULL UNIQUE,
  name       TEXT NOT NULL,
  icon       TEXT NOT NULL DEFAULT '',
  color      TEXT NOT NULL DEFAULT '#D9447F',
  sort_order INTEGER NOT NULL DEFAULT 0
);

-- Admins log in with their student ID, which is also their primary key.
CREATE TABLE admins (
  student_id    TEXT PRIMARY KEY COLLATE NOCASE,
  full_name     TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  active        INTEGER NOT NULL DEFAULT 1,
  created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE sessions (
  token_hash TEXT PRIMARY KEY,
  student_id TEXT NOT NULL REFERENCES admins(student_id) ON DELETE CASCADE,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  expires_at TEXT NOT NULL
);
CREATE INDEX sessions_admin ON sessions(student_id);

-- Every admin action. admin_id is a plain student ID (no foreign key) so the
-- history survives when an admin is removed.
CREATE TABLE audit_log (
  id        INTEGER PRIMARY KEY,
  at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
  admin_id  TEXT NOT NULL COLLATE NOCASE,
  action    TEXT NOT NULL,
  entity    TEXT NOT NULL DEFAULT '',
  entity_id TEXT NOT NULL DEFAULT '',
  details   TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX audit_log_admin ON audit_log(admin_id, id);
CREATE INDEX audit_log_entity ON audit_log(entity, entity_id, id);

CREATE TABLE places (
  id              INTEGER PRIMARY KEY,
  slug            TEXT NOT NULL UNIQUE,
  category_id     INTEGER NOT NULL REFERENCES categories(id),
  name            TEXT NOT NULL,
  name_th         TEXT NOT NULL DEFAULT '',
  summary         TEXT NOT NULL DEFAULT '',
  description_md  TEXT NOT NULL DEFAULT '',
  address         TEXT NOT NULL DEFAULT '',
  lat             REAL NOT NULL,
  lng             REAL NOT NULL,
  nearest_station TEXT NOT NULL DEFAULT '',
  station_line    TEXT NOT NULL DEFAULT '',
  walk_minutes    INTEGER,
  price_level     INTEGER CHECK (price_level BETWEEN 1 AND 4),
  dress_code      TEXT NOT NULL DEFAULT '',
  etiquette_tips  TEXT NOT NULL DEFAULT '',
  must_try        TEXT NOT NULL DEFAULT '',
  website         TEXT NOT NULL DEFAULT '',
  phone           TEXT NOT NULL DEFAULT '',
  google_maps_url TEXT NOT NULL DEFAULT '',
  status          TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
  featured        INTEGER NOT NULL DEFAULT 0,
  created_by      TEXT, -- student ID; see audit_log for the full history
  updated_by      TEXT,
  created_at      TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX places_category ON places(category_id, status);

-- Times are "HH:MM" in Asia/Bangkok. closes <= opens means the interval runs past midnight.
CREATE TABLE place_hours (
  place_id INTEGER NOT NULL REFERENCES places(id) ON DELETE CASCADE,
  weekday  INTEGER NOT NULL CHECK (weekday BETWEEN 0 AND 6), -- 0 = Sunday
  opens    TEXT NOT NULL,
  closes   TEXT NOT NULL
);
CREATE INDEX place_hours_place ON place_hours(place_id);

CREATE TABLE photos (
  id         INTEGER PRIMARY KEY,
  place_id   INTEGER NOT NULL REFERENCES places(id) ON DELETE CASCADE,
  file_key   TEXT NOT NULL,
  width      INTEGER NOT NULL,
  height     INTEGER NOT NULL,
  alt        TEXT NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX photos_place ON photos(place_id, sort_order);

CREATE TABLE suggestions (
  id            INTEGER PRIMARY KEY,
  name          TEXT NOT NULL,
  category_hint TEXT NOT NULL DEFAULT '',
  location_text TEXT NOT NULL DEFAULT '',
  note          TEXT NOT NULL DEFAULT '',
  contact       TEXT NOT NULL DEFAULT '',
  status        TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'accepted', 'rejected')),
  created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE pages (
  slug       TEXT PRIMARY KEY,
  title      TEXT NOT NULL,
  body_md    TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE VIRTUAL TABLE places_fts USING fts5(
  name, name_th, summary, description_md,
  content='places', content_rowid='id', tokenize='unicode61 remove_diacritics 2'
);
CREATE TRIGGER places_ai AFTER INSERT ON places BEGIN
  INSERT INTO places_fts(rowid, name, name_th, summary, description_md)
  VALUES (new.id, new.name, new.name_th, new.summary, new.description_md);
END;
CREATE TRIGGER places_ad AFTER DELETE ON places BEGIN
  INSERT INTO places_fts(places_fts, rowid, name, name_th, summary, description_md)
  VALUES ('delete', old.id, old.name, old.name_th, old.summary, old.description_md);
END;
CREATE TRIGGER places_au AFTER UPDATE ON places BEGIN
  INSERT INTO places_fts(places_fts, rowid, name, name_th, summary, description_md)
  VALUES ('delete', old.id, old.name, old.name_th, old.summary, old.description_md);
  INSERT INTO places_fts(rowid, name, name_th, summary, description_md)
  VALUES (new.id, new.name, new.name_th, new.summary, new.description_md);
END;
