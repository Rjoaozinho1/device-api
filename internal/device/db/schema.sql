CREATE TYPE device_state AS ENUM ('available', 'in-use', 'inactive');

CREATE TABLE IF NOT EXISTS device (
    ID UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    Name TEXT NOT NULL CHECK (length(Name) > 0),
    Brand TEXT NOT NULL CHECK (length(Brand) > 0),
    State device_state NOT NULL DEFAULT 'available',
    CreationTime TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_device_brand ON device(Brand);
CREATE INDEX idx_device_state ON device(State);