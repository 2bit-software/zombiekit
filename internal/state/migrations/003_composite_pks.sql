-- Migration 003: Composite primary keys for multi-project support.
-- Clean break: drops existing data. Restart any in-flight agents after upgrade.

DROP TABLE IF EXISTS jobs;

CREATE TABLE jobs (
    project_id    TEXT    NOT NULL,
    ticket_id     TEXT    NOT NULL,
    worktree_path TEXT    NOT NULL,
    cmux_session  TEXT    NOT NULL,
    pr_number     INTEGER,
    status        TEXT    NOT NULL DEFAULT 'queued',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id, ticket_id)
);

CREATE INDEX idx_jobs_status ON jobs (status);
CREATE INDEX idx_jobs_pr_number ON jobs (pr_number);
CREATE INDEX idx_jobs_project_status ON jobs (project_id, status);

DROP TABLE IF EXISTS comment_watermarks;

CREATE TABLE comment_watermarks (
    project_id                TEXT    NOT NULL,
    pr_number                 INTEGER NOT NULL,
    last_processed_comment_id INTEGER NOT NULL,
    updated_at                TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id, pr_number)
);
