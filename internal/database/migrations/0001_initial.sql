CREATE TABLE evidence_identities (
    reporter_id BLOB PRIMARY KEY CHECK(length(reporter_id) = 32),
    highest_revision TEXT NOT NULL CHECK(length(highest_revision) BETWEEN 1 AND 20 AND highest_revision NOT GLOB '*[^0-9]*')
) STRICT;

CREATE TABLE evidence_batches (
    reporter_id BLOB NOT NULL REFERENCES evidence_identities(reporter_id),
    batch_id TEXT NOT NULL,
    revision TEXT NOT NULL CHECK(length(revision) BETWEEN 1 AND 20 AND revision NOT GLOB '*[^0-9]*'),
    digest TEXT NOT NULL,
    received_hour INTEGER NOT NULL,
    report_json TEXT NOT NULL CHECK(json_valid(report_json)),
    PRIMARY KEY (reporter_id, batch_id)
) STRICT;

CREATE TABLE survey_submissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    submitted_at TEXT,
    status_mask INTEGER NOT NULL CHECK(status_mask BETWEEN 0 AND 7),
    payload TEXT NOT NULL CHECK(json_valid(payload))
) STRICT;

CREATE TABLE survey_statistics (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    payload TEXT NOT NULL CHECK(json_valid(payload))
) STRICT;

CREATE TABLE legacy_imports (
    source_sha256 TEXT PRIMARY KEY,
    imported_at TEXT NOT NULL,
    survey_count INTEGER NOT NULL,
    identity_count INTEGER NOT NULL,
    batch_count INTEGER NOT NULL
) STRICT;
