CREATE TABLE IF NOT EXISTS waitingrooms(
    room_id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    max_active_users_count INTEGER NOT NULL CHECK (max_active_users_count > 0),
    origin_application TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DELETED')),
    owner_id TEXT NOT NULL
);