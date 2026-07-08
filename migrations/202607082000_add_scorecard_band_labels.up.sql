-- Scorecard v2: band stage names become tenant-editable. Five stages — the
-- four thresholded bands plus the bottom band (scored but below Low), which
-- previously existed only as a hardcoded "Not Recommended" label.

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS scorecard_band_strong_label VARCHAR(60) NOT NULL DEFAULT 'Strong Candidate',
    ADD COLUMN IF NOT EXISTS scorecard_band_good_label   VARCHAR(60) NOT NULL DEFAULT 'Good Candidate',
    ADD COLUMN IF NOT EXISTS scorecard_band_refine_label VARCHAR(60) NOT NULL DEFAULT 'Needs Refinement',
    ADD COLUMN IF NOT EXISTS scorecard_band_low_label    VARCHAR(60) NOT NULL DEFAULT 'Low Priority',
    ADD COLUMN IF NOT EXISTS scorecard_band_none_label   VARCHAR(60) NOT NULL DEFAULT 'Not Recommended';
