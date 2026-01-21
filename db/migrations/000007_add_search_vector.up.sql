-- Add search_vector column
ALTER TABLE products ADD COLUMN search_vector tsvector;

-- Create GIN index for fast search
CREATE INDEX idx_products_search_vector ON products USING GIN(search_vector);

-- Create a function to update the search_vector
CREATE FUNCTION products_search_vector_update() RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', coalesce(NEW.name, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(NEW.description, '')), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- Create a trigger to update the search_vector on insert or update
CREATE TRIGGER products_search_vector_update
BEFORE INSERT OR UPDATE ON products
FOR EACH ROW
EXECUTE FUNCTION products_search_vector_update();

-- Backfill existing data
UPDATE products SET search_vector =
    setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(description, '')), 'B');
