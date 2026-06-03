ALTER TABLE waitingrooms
ADD COLUMN polling_interval_seconds INTEGER CHECK (polling_interval_seconds > 0);
