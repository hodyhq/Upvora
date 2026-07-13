-- Universal default scorecard catalogue (fork v0.36.1.2.41).
-- 2026-07-13 (v0.36.1.3.1): demo-field INSERTs are now guarded so they only
-- seed tenants with NO custom (non-system) fields — upgrading an instance
-- whose admin already built a scorecard must not add demo questions.
-- Already-migrated databases are unaffected (the migration does not re-run).
-- 1) Reword the 8 locked scoring dimensions so they read company-agnostic.
--    Keys and weights are stable - existing scorecard values keep working.
-- 2) Seed a set of non-system default fields that exercise every field type
--    (text, multiline, note, date, number, url, choice, user) across every body group,
--    so a fresh scorecard demonstrates the whole form. Admins can edit,
--    deactivate, or delete them (is_system = FALSE).

UPDATE scorecard_fields SET label = 'Strategic alignment', question = 'Does this advance a stated company goal or priority?'            WHERE key = 'score_strategic'      AND is_system;
UPDATE scorecard_fields SET label = 'Business value',      question = 'Will this measurably improve revenue, cost, quality, or speed?'  WHERE key = 'score_business_value' AND is_system;
UPDATE scorecard_fields SET label = 'Ownership clarity',   question = 'Is there a clear owner accountable for the outcome?'             WHERE key = 'score_ownership'      AND is_system;
UPDATE scorecard_fields SET label = 'Workflow clarity',    question = 'Do we understand how day-to-day work changes once this ships?'   WHERE key = 'score_workflow'       AND is_system;
UPDATE scorecard_fields SET label = 'Readiness',           question = 'Are the inputs - data, content, integrations - available and reliable?' WHERE key = 'score_data_readiness' AND is_system;
UPDATE scorecard_fields SET label = 'Risk manageability',  question = 'Can security, privacy, and compliance risks be managed?'         WHERE key = 'score_risk'           AND is_system;
UPDATE scorecard_fields SET label = 'Adoption likelihood', question = 'Will people actually use this without heavy change management?' WHERE key = 'score_adoption'       AND is_system;
UPDATE scorecard_fields SET label = 'Supportability',      question = 'Can we support and maintain this for the long haul?'             WHERE key = 'score_supportability' AND is_system;

INSERT INTO scorecard_fields (tenant_id, key, label, group_key, type, weight, question, sort_order, is_system)
SELECT id, 'requesting_team',   'Requesting team',   'intake',         'text',   0, 'Which team or department is asking for this?',                          110, FALSE FROM tenants WHERE NOT EXISTS (SELECT 1 FROM scorecard_fields f WHERE f.tenant_id = tenants.id AND NOT f.is_system)
UNION ALL
SELECT id, 'requested_by',      'Requested by',      'intake',         'user',   0, 'Who raised or sponsors this request?',                                  120, FALSE FROM tenants WHERE NOT EXISTS (SELECT 1 FROM scorecard_fields f WHERE f.tenant_id = tenants.id AND NOT f.is_system)
UNION ALL
SELECT id, 'problem_statement', 'Problem statement', 'context',        'note',   0, 'What problem does this solve, and what happens if we do nothing?',      210, FALSE FROM tenants WHERE NOT EXISTS (SELECT 1 FROM scorecard_fields f WHERE f.tenant_id = tenants.id AND NOT f.is_system)
UNION ALL
SELECT id, 'reference_link',    'Reference link',    'context',        'url',    0, 'Link to a doc, mockup, or example that explains the idea.',             220, FALSE FROM tenants WHERE NOT EXISTS (SELECT 1 FROM scorecard_fields f WHERE f.tenant_id = tenants.id AND NOT f.is_system)
UNION ALL
SELECT id, 'process_affected',  'Process affected',  'workflow',       'multiline', 0, 'Which process or workflow would this change, and how is it handled today?', 310, FALSE FROM tenants WHERE NOT EXISTS (SELECT 1 FROM scorecard_fields f WHERE f.tenant_id = tenants.id AND NOT f.is_system)
UNION ALL
SELECT id, 'needed_by',         'Needed by',         'workflow',       'date',   0, 'When does this need to be in place to matter?',                         320, FALSE FROM tenants WHERE NOT EXISTS (SELECT 1 FROM scorecard_fields f WHERE f.tenant_id = tenants.id AND NOT f.is_system)
UNION ALL
SELECT id, 'business_owner',    'Business owner',    'ownership',      'user',   0, 'Who owns this after it ships?',                                         410, FALSE FROM tenants WHERE NOT EXISTS (SELECT 1 FROM scorecard_fields f WHERE f.tenant_id = tenants.id AND NOT f.is_system)
UNION ALL
SELECT id, 'people_impacted',   'People impacted',   'classification', 'number', 0, 'Roughly how many people would use or benefit from this?',              520, FALSE FROM tenants WHERE NOT EXISTS (SELECT 1 FROM scorecard_fields f WHERE f.tenant_id = tenants.id AND NOT f.is_system)
UNION ALL
SELECT id, 'decision_notes',    'Decision notes',    'decision',       'note',   0, 'Rationale, conditions, and next steps for the decision.',              910, FALSE FROM tenants WHERE NOT EXISTS (SELECT 1 FROM scorecard_fields f WHERE f.tenant_id = tenants.id AND NOT f.is_system)
UNION ALL
SELECT id, 'next_review',       'Next review',       'decision',       'date',   0, 'When should this decision be revisited?',                               920, FALSE FROM tenants WHERE NOT EXISTS (SELECT 1 FROM scorecard_fields f WHERE f.tenant_id = tenants.id AND NOT f.is_system)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Effort estimate is a choice field - seeded separately for its choices JSON.
INSERT INTO scorecard_fields (tenant_id, key, label, group_key, type, choices, weight, question, sort_order, is_system)
SELECT id, 'effort_estimate', 'Effort estimate', 'classification', 'choice', '[
  {"value": "Small",       "color": "mint"},
  {"value": "Medium",      "color": "gold"},
  {"value": "Large",       "color": "salmon"},
  {"value": "Extra Large", "color": "coral"}
]'::jsonb, 0, 'How big is this likely to be?', 510, FALSE
FROM tenants
ON CONFLICT (tenant_id, key) DO NOTHING;
