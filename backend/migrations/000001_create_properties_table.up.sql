-- create_properties_table.up.sql
CREATE TABLE IF NOT EXISTS properties (
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    title text NOT NULL,
    year_built integer,
    area integer,
    bedrooms integer,
    bathrooms integer,
    floor integer,
    price numeric(12,2),
    location text NOT NULL,
    property_type text NOT NULL,
    features text[],
    images text[],
    version integer NOT NULL DEFAULT 1
);
