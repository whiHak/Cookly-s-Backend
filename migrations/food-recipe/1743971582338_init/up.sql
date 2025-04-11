-- Create users table
CREATE TABLE users (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    bio TEXT,
    profile_picture TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create categories table
CREATE TABLE categories (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create reciepe_categories table
CREATE TABLE recipe_categories (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    recipe_id TEXT REFERENCES recipes(id) ON DELETE CASCADE,
    category_id TEXT REFERENCES categories(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(recipe_id, category_id)
);


-- Create recipes table
CREATE TABLE recipes (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    difficulty TEXT,
    servings INTEGER,
    preparation_time INTEGER NOT NULL, -- in minutes
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    featured_image TEXT NOT NULL,
    price INTEGER NOT NULL -- for recipe purchase feature
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create recipe_images table
CREATE TABLE recipe_images (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    recipe_id TEXT REFERENCES recipes(id) ON DELETE CASCADE,
    image_url TEXT NOT NULL,
    is_featured BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create recipe_steps table
CREATE TABLE recipe_steps (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    recipe_id TEXT REFERENCES recipes(id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL,
    description TEXT NOT NULL,
    image_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(recipe_id, step_number)
);

-- Create ingredients table
CREATE TABLE ingredients (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create recipe_ingredients table (junction table)
CREATE TABLE recipe_ingredients (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    recipe_id TEXT REFERENCES recipes(id) ON DELETE CASCADE,
    ingredient_id TEXT REFERENCES ingredients(id) ON DELETE CASCADE,
    quantity VARCHAR(255) NOT NULL,
    unit VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(recipe_id, ingredient_id)
);

-- Create likes table
CREATE TABLE recipe_likes (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    recipe_id TEXT REFERENCES recipes(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(recipe_id, user_id)
);

-- Create bookmarks table
CREATE TABLE recipe_bookmarks (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    recipe_id TEXT REFERENCES recipes(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(recipe_id, user_id)
);

-- Create comments table
CREATE TABLE recipe_comments (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    recipe_id TEXT REFERENCES recipes(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create ratings table
CREATE TABLE recipe_ratings (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    recipe_id TEXT REFERENCES recipes(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    created_at TIMESTAMPTZ DEFAULT NOW(),
);

-- Create purchases table
CREATE TABLE recipe_purchases (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    recipe_id TEXT REFERENCES recipes(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(10,2) NOT NULL,
    transaction_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create computed columns function for average rating
CREATE OR REPLACE FUNCTION get_recipe_average_rating(recipe_row recipes)
RETURNS DECIMAL AS $$
    SELECT COALESCE(AVG(rating)::DECIMAL(3,2), 0)
    FROM recipe_ratings
    WHERE recipe_id = recipe_row.id;
$$ LANGUAGE SQL STABLE;

-- Create computed columns function for total likes
CREATE OR REPLACE FUNCTION get_recipe_likes_count(recipe_row recipes)
RETURNS INTEGER AS $$
    SELECT COUNT(*)
    FROM recipe_likes
    WHERE recipe_id = recipe_row.id;
$$ LANGUAGE SQL STABLE;

-- Create trigger for updating timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add triggers for tables that need updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_recipes_updated_at
    BEFORE UPDATE ON recipes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_recipe_comments_updated_at
    BEFORE UPDATE ON recipe_comments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column(); 