-- CREATE TABLE IF NOT EXISTS EMAIL_NOTIFICATIONS (
--     id INTEGER PRIMARY KEY AUTOINCREMENT,
--     email TEXT NOT NULL,
--     received_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
--     historyID STRING NOT NULL
-- );

CREATE TABLE IF NOT EXISTS INGEST_CURSOR (
    source             TEXT PRIMARY KEY,
    history_id         INTEGER NOT NULL,
    received_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS NORMALIZED_SUBMISSIONS (
    id STRING PRIMARY KEY,
    source TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL,
    author TEXT NOT NULL,
    contact_info TEXT NOT NULL,
    attribution_tags TEXT NOT NULL,
    image_paths TEXT NOT NULL, -- stored as a JSON array of image paths
    caption TEXT NOT NULL
);
