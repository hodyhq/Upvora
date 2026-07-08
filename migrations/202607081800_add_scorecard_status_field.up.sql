-- Scorecard v2 (fork v0.36.1.3): card status lives at the top of every card as
-- a system choice field. New group 'header' renders beside the title instead of
-- in the body groups. Choice entries gain an optional "bucket" that maps custom
-- statuses onto the dashboard views (new | review | executive).

ALTER TABLE scorecard_fields DROP CONSTRAINT IF EXISTS scorecard_fields_group_key_check;
ALTER TABLE scorecard_fields ADD CONSTRAINT scorecard_fields_group_key_check
    CHECK (group_key IN ('header', 'intake', 'context', 'workflow', 'ownership', 'classification', 'scoring', 'decision'));

-- Seed the status field for every existing tenant. Colors reference the band
-- chip palette keys the client already knows; bucket drives dashboard tabs.
INSERT INTO scorecard_fields (tenant_id, key, label, group_key, type, choices, sort_order, is_system)
SELECT id, 'status', 'Status', 'header', 'choice', '[
  {"value": "Submitted",            "color": "blue",   "bucket": "new"},
  {"value": "Under Review",         "color": "gold",   "bucket": "review"},
  {"value": "Needs Clarification",  "color": "salmon", "bucket": "review"},
  {"value": "Scored",               "color": "lavender", "bucket": "executive"},
  {"value": "Approved for Pilot",   "color": "mint",   "bucket": "executive"},
  {"value": "Pilot in Progress",    "color": "cyan",   "bucket": "executive"},
  {"value": "Approved for Rollout", "color": "purple", "bucket": "executive"},
  {"value": "Deferred",             "color": "gray",   "bucket": "executive"},
  {"value": "Rejected",             "color": "coral",  "bucket": "executive"},
  {"value": "Completed",            "color": "pink",   "bucket": "executive"}
]'::jsonb, 0, TRUE
FROM tenants
ON CONFLICT (tenant_id, key) DO NOTHING;
