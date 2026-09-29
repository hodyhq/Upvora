-- Private Ideas: an idea only collaborators and administrators can see.
ALTER TABLE posts ADD COLUMN IF NOT EXISTS is_private BOOLEAN NOT NULL DEFAULT false;
