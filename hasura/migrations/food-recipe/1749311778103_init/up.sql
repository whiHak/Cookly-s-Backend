SET check_function_bodies = false;
CREATE TYPE public.recipe_likes_result AS (
	likes_count integer
);
CREATE TYPE public.recipe_rating_result AS (
	rating integer
);
CREATE TABLE public.recipes (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    title character varying(255) NOT NULL,
    description text,
    preparation_time integer NOT NULL,
    user_id text,
    featured_image text NOT NULL,
    price real,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    difficulty text,
    servings integer
);
CREATE FUNCTION public.get_recipe_average_rating(recipe_row public.recipes) RETURNS TABLE(rating integer)
    LANGUAGE sql STABLE
    AS $$
    SELECT COALESCE(ROUND(AVG(r.rating)), 0)::INTEGER
    FROM recipe_ratings r
    WHERE r.recipe_id = recipe_row.id;
$$;
CREATE FUNCTION public.get_recipe_average_rating_composite(recipe_row public.recipes) RETURNS public.recipe_rating_result
    LANGUAGE sql STABLE
    AS $$
    SELECT ROW(
        COALESCE(ROUND(AVG(rating)), 0)::INTEGER
    )::recipe_rating_result
    FROM recipe_ratings
    WHERE recipe_id = recipe_row.id;
$$;
CREATE FUNCTION public.get_recipe_likes_count(recipe_row public.recipes) RETURNS TABLE(likes_count integer)
    LANGUAGE sql STABLE
    AS $$
    SELECT COUNT(*)::INTEGER
    FROM recipe_likes l
    WHERE l.recipe_id = recipe_row.id;
$$;
CREATE FUNCTION public.get_recipe_likes_count_composite(recipe_row public.recipes) RETURNS public.recipe_likes_result
    LANGUAGE sql STABLE
    AS $$
    SELECT ROW(
        COUNT(*)
    )::recipe_likes_result
    FROM recipe_likes
    WHERE recipe_id = recipe_row.id;
$$;
CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;
CREATE TABLE public.categories (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    image_url text,
    created_at timestamp with time zone DEFAULT now()
);
CREATE TABLE public.ingredients (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    name text
);
CREATE TABLE public.recipe_bookmarks (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    recipe_id text,
    user_id text,
    created_at timestamp with time zone DEFAULT now()
);
CREATE TABLE public.recipe_categories (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    recipe_id text,
    category_id text,
    created_at timestamp with time zone DEFAULT now()
);
CREATE TABLE public.recipe_comments (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    recipe_id text,
    user_id text,
    content text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);
CREATE TABLE public.recipe_images (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    recipe_id text,
    image_url text NOT NULL,
    is_featured boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now()
);
CREATE TABLE public.recipe_ingredients (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    recipe_id text,
    ingredient_id text,
    quantity character varying(255) NOT NULL,
    unit character varying(50),
    created_at timestamp with time zone DEFAULT now()
);
CREATE TABLE public.recipe_likes (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    recipe_id text,
    user_id text,
    created_at timestamp with time zone DEFAULT now()
);
CREATE TABLE public.recipe_purchases (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    recipe_id text,
    user_id text,
    amount numeric(10,2) NOT NULL,
    transaction_id character varying(255) NOT NULL,
    status character varying(50) NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);
CREATE TABLE public.recipe_ratings (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    recipe_id text,
    user_id text,
    rating integer NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT recipe_ratings_rating_check CHECK (((rating >= 1) AND (rating <= 5)))
);
CREATE TABLE public.recipe_steps (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    recipe_id text,
    step_number integer NOT NULL,
    description text NOT NULL,
    image_url text,
    created_at timestamp with time zone DEFAULT now()
);
CREATE TABLE public.users (
    id text DEFAULT (gen_random_uuid())::text NOT NULL,
    username character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    password_hash character varying(255) NOT NULL,
    full_name character varying(255),
    bio text,
    profile_picture text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);
ALTER TABLE ONLY public.categories
    ADD CONSTRAINT categories_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ingredients
    ADD CONSTRAINT ingredients_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recipe_bookmarks
    ADD CONSTRAINT recipe_bookmarks_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recipe_bookmarks
    ADD CONSTRAINT recipe_bookmarks_recipe_id_user_id_key UNIQUE (recipe_id, user_id);
ALTER TABLE ONLY public.recipe_categories
    ADD CONSTRAINT recipe_categories_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recipe_categories
    ADD CONSTRAINT recipe_categories_recipe_id_category_id_key UNIQUE (recipe_id, category_id);
ALTER TABLE ONLY public.recipe_comments
    ADD CONSTRAINT recipe_comments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recipe_images
    ADD CONSTRAINT recipe_images_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recipe_ingredients
    ADD CONSTRAINT recipe_ingredients_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recipe_ingredients
    ADD CONSTRAINT recipe_ingredients_recipe_id_ingredient_id_key UNIQUE (recipe_id, ingredient_id);
ALTER TABLE ONLY public.recipe_likes
    ADD CONSTRAINT recipe_likes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recipe_likes
    ADD CONSTRAINT recipe_likes_recipe_id_user_id_key UNIQUE (recipe_id, user_id);
ALTER TABLE ONLY public.recipe_purchases
    ADD CONSTRAINT recipe_purchases_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recipe_ratings
    ADD CONSTRAINT recipe_ratings_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recipe_steps
    ADD CONSTRAINT recipe_steps_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recipe_steps
    ADD CONSTRAINT recipe_steps_recipe_id_step_number_key UNIQUE (recipe_id, step_number);
ALTER TABLE ONLY public.recipes
    ADD CONSTRAINT recipes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);
CREATE TRIGGER update_recipe_comments_updated_at BEFORE UPDATE ON public.recipe_comments FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
CREATE TRIGGER update_recipes_updated_at BEFORE UPDATE ON public.recipes FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();
ALTER TABLE ONLY public.recipe_bookmarks
    ADD CONSTRAINT recipe_bookmarks_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES public.recipes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_bookmarks
    ADD CONSTRAINT recipe_bookmarks_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_categories
    ADD CONSTRAINT recipe_categories_category_id_fkey2 FOREIGN KEY (category_id) REFERENCES public.categories(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_categories
    ADD CONSTRAINT recipe_categories_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES public.recipes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_comments
    ADD CONSTRAINT recipe_comments_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES public.recipes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_comments
    ADD CONSTRAINT recipe_comments_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_images
    ADD CONSTRAINT recipe_images_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES public.recipes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_ingredients
    ADD CONSTRAINT recipe_ingredients_ingredient_id_fkey FOREIGN KEY (ingredient_id) REFERENCES public.ingredients(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_ingredients
    ADD CONSTRAINT recipe_ingredients_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES public.recipes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_likes
    ADD CONSTRAINT recipe_likes_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES public.recipes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_likes
    ADD CONSTRAINT recipe_likes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_purchases
    ADD CONSTRAINT recipe_purchases_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES public.recipes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_purchases
    ADD CONSTRAINT recipe_purchases_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_ratings
    ADD CONSTRAINT recipe_ratings_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES public.recipes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_ratings
    ADD CONSTRAINT recipe_ratings_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_steps
    ADD CONSTRAINT recipe_steps_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES public.recipes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipes
    ADD CONSTRAINT recipes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;
