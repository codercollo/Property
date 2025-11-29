-- Check that Area is positive
ALTER TABLE properties
ADD CONSTRAINT properties_area_check CHECK (area > 0);

-- Check that Bedrooms is positive
ALTER TABLE properties
ADD CONSTRAINT properties_bedrooms_check CHECK (bedrooms > 0);

-- Check that Bathrooms is zero or more
ALTER TABLE properties
ADD CONSTRAINT properties_bathrooms_check CHECK (bathrooms >= 0);

-- Check that Floor is zero or more
ALTER TABLE properties
ADD CONSTRAINT properties_floor_check CHECK (floor >= 0);

-- Check that Price is positive
ALTER TABLE properties
ADD CONSTRAINT properties_price_check CHECK (price > 0);

-- Check that YearBuilt is between 1800 and current year
ALTER TABLE properties
ADD CONSTRAINT properties_yearbuilt_check CHECK (year_built BETWEEN 1800 AND date_part('year', now()));

-- Check features array length (1 to 10)
ALTER TABLE properties
ADD CONSTRAINT properties_features_length_check CHECK (array_length(features, 1) BETWEEN 1 AND 10);
