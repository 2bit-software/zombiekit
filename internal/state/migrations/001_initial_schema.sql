CREATE TABLE IF NOT EXISTS jobs (
    ticket_id     TEXT PRIMARY KEY,
    worktree_path TEXT NOT NULL,
    cmux_session  TEXT NOT NULL,
    pr_number     INTEGER,
    status        TEXT NOT NULL DEFAULT 'queued',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS comment_watermarks (
    pr_number                 INTEGER PRIMARY KEY,
    last_processed_comment_id INTEGER NOT NULL,
    updated_at                TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS concurrency_slots (
    project_id   TEXT PRIMARY KEY,
    active_count INTEGER NOT NULL DEFAULT 0,
    slot_limit   INTEGER NOT NULL DEFAULT 1
);
