ALTER TABLE waitingrooms
ADD COLUMN active_session_ttl_seconds INTEGER CHECK (active_session_ttl_seconds > 0),
ADD COLUMN waiting_session_ttl_seconds INTEGER CHECK (waiting_session_ttl_seconds > 0);