CREATE SEQUENCE appointment_ids;

CREATE TABLE appointments (
    id            text PRIMARY KEY,
    vin           text NOT NULL,
    dealership_id text NOT NULL,
    scheduled_at  timestamptz NOT NULL,
    service       text NOT NULL,
    notes         text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX appointments_vin ON appointments (vin, scheduled_at);
CREATE INDEX appointments_slot ON appointments (dealership_id, scheduled_at);
