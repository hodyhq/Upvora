-- Web search for Vora: admins connect Serper.dev or a self-hosted SearXNG
-- instance so the agent can ground ideas in current web results.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS ai_web_search_enabled  BOOLEAN     NOT NULL DEFAULT FALSE;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS ai_web_search_provider VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS ai_web_search_api_key  TEXT        NOT NULL DEFAULT '';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS ai_web_search_base_url TEXT        NOT NULL DEFAULT '';

-- Per-product opt-in (default off): web search only runs for agents that enable it.
ALTER TABLE ai_agents ADD COLUMN IF NOT EXISTS web_search_enabled BOOLEAN NOT NULL DEFAULT FALSE;
