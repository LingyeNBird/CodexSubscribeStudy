CREATE INDEX survey_submissions_submitted_at ON survey_submissions(submitted_at, id);
CREATE INDEX survey_submissions_status_time ON survey_submissions(status_mask, submitted_at, id);
CREATE INDEX evidence_batches_received_hour ON evidence_batches(received_hour);

CREATE VIEW survey_answers AS
SELECT s.id AS submission_id, questions.key AS question, options.value AS option
FROM survey_submissions AS s,
     json_each(s.payload, '$.answers') AS questions,
     json_each(questions.value) AS options;

CREATE VIEW survey_details AS
SELECT s.id AS submission_id, details.key AS field, details.value AS value
FROM survey_submissions AS s, json_each(s.payload, '$.details') AS details;
