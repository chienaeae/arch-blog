-- Create posts table
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(200) NOT NULL,
    slug VARCHAR(250) UNIQUE NOT NULL,
    content TEXT NOT NULL, -- HTML content
    excerpt VARCHAR(500),  -- Plain text excerpt
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Data integrity constraints
    CONSTRAINT check_published_at_when_published 
        CHECK ((status = 'published' AND published_at IS NOT NULL) OR 
               (status != 'published')),
    
    CONSTRAINT check_title_not_empty 
        CHECK (LENGTH(TRIM(title)) > 0),
    
    CONSTRAINT check_slug_format 
        CHECK (slug ~ '^[a-z0-9-]+$'),
    
    CONSTRAINT check_content_not_empty 
        CHECK (LENGTH(TRIM(content)) > 0)
);

-- Create indexes for posts
CREATE INDEX idx_posts_author_id ON posts(author_id);
CREATE INDEX idx_posts_status ON posts(status);
CREATE INDEX idx_posts_published_at ON posts(published_at DESC) WHERE status = 'published';
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX idx_posts_slug ON posts(slug);

-- Create themes table
CREATE TABLE themes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(150) UNIQUE NOT NULL,
    description VARCHAR(1000),
    curator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Data integrity constraints
    CONSTRAINT check_theme_name_not_empty 
        CHECK (LENGTH(TRIM(name)) > 0),
    
    CONSTRAINT check_theme_slug_format 
        CHECK (slug ~ '^[a-z0-9-]+$')
);

-- Create indexes for themes
CREATE INDEX idx_themes_curator_id ON themes(curator_id);
CREATE INDEX idx_themes_is_active ON themes(is_active);
CREATE INDEX idx_themes_slug ON themes(slug);
CREATE INDEX idx_themes_created_at ON themes(created_at DESC);

-- Create theme_articles table (junction table for themes and posts)
CREATE TABLE theme_articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    theme_id UUID NOT NULL REFERENCES themes(id) ON DELETE CASCADE,
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position > 0),
    added_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure a post can only be in a theme once
    UNIQUE(theme_id, post_id),
    -- Ensure positions are unique within a theme
    UNIQUE(theme_id, position)
);

-- Create a function to check if a post is published
CREATE OR REPLACE FUNCTION check_post_is_published()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM posts 
        WHERE id = NEW.post_id 
        AND status = 'published'
    ) THEN
        RAISE EXCEPTION 'Only published posts can be added to themes';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to enforce the business rule
CREATE TRIGGER ensure_post_published_before_theme_add
    BEFORE INSERT OR UPDATE ON theme_articles
    FOR EACH ROW
    EXECUTE FUNCTION check_post_is_published();

-- Create indexes for theme_articles
CREATE INDEX idx_theme_articles_theme_id ON theme_articles(theme_id);
CREATE INDEX idx_theme_articles_post_id ON theme_articles(post_id);
CREATE INDEX idx_theme_articles_position ON theme_articles(theme_id, position);
CREATE INDEX idx_theme_articles_added_by ON theme_articles(added_by);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create updated_at triggers
CREATE TRIGGER update_posts_updated_at BEFORE UPDATE ON posts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_themes_updated_at BEFORE UPDATE ON themes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_theme_articles_updated_at BEFORE UPDATE ON theme_articles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE posts IS 'Blog posts with content and publication status';
COMMENT ON TABLE themes IS 'Curated collections of posts organized by topic';
COMMENT ON TABLE theme_articles IS 'Junction table linking posts to themes with ordering';

COMMENT ON COLUMN posts.status IS 'Publication status: draft, published, or archived';
COMMENT ON COLUMN posts.content IS 'HTML content of the post';
COMMENT ON COLUMN posts.excerpt IS 'Plain text summary of the post';

COMMENT ON COLUMN themes.curator_id IS 'User who created and manages this theme';
COMMENT ON COLUMN themes.is_active IS 'Whether the theme is currently active and visible';

COMMENT ON COLUMN theme_articles.position IS '1-based position of the article within the theme';
COMMENT ON COLUMN theme_articles.added_by IS 'User who added this post to the theme';
