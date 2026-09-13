CREATE TABLE survey_annotations (
    submission_id INTEGER PRIMARY KEY REFERENCES survey_submissions(id) ON DELETE CASCADE,
    tags TEXT NOT NULL CHECK(json_valid(tags) AND json_type(tags) = 'array'),
    note TEXT NOT NULL CHECK(length(note) <= 2000),
    updated_at TEXT NOT NULL
) STRICT;
