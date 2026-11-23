-- Application-owned outbox used by the compatibility consumer. The broker
-- creates and manages its own history and notification tables separately.
CREATE TABLE IF NOT EXISTS compat_consumer_outbox (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    method text NOT NULL,
    payload jsonb NOT NULL,
    partition integer NOT NULL DEFAULT 0
);
