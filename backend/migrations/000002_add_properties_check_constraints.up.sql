-- Add a column to mark when a property is featured; NULL means not featured
ALTER TABLE properties
ADD COLUMN featured_at timestamp with time zone;

-- Ensure area is a positive number
ALTER TABLE properties
ADD CONSTRAINT properties_area_check CHECK (area > 0);

-- Ensure bedrooms is a positive number
ALTER TABLE properties
ADD CONSTRAINT properties_bedrooms_check CHECK (bedrooms > 0);

-- Ensure bathrooms is zero or more
ALTER TABLE properties
ADD CONSTRAINT properties_bathrooms_check CHECK (bathrooms >= 0);

-- Ensure floor is zero or more
ALTER TABLE properties
ADD CONSTRAINT properties_floor_check CHECK (floor >= 0);

-- Ensure price is positive
ALTER TABLE properties
ADD CONSTRAINT properties_price_check CHECK (price > 0);

-- Ensure year built is between 1800 and the current year
ALTER TABLE properties
ADD CONSTRAINT properties_yearbuilt_check CHECK (year_built BETWEEN 1800 AND date_part('year', now()));

-- Ensure features array has between 1 and 10 items
ALTER TABLE properties
ADD CONSTRAINT properties_features_length_check CHECK (array_length(features, 1) BETWEEN 1 AND 10);
