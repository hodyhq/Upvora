# Updating Fider — Pulling a New Upstream Release

> [!info] What this doc is
> A complete, dummy-proof procedure for when the original Fider developer ([getfider/fider](https://github.com/getfider/fider)) releases a new version and we want our HCM-branded instance to pick it up.

Related: [[HCM-Fider-Theme]] — what our fork actually changes.

---

## 0. Mental model (read this first)

We have one git repo with two branches:

- **`main`** — mirrors the upstream `getfider/fider` repo. We never edit code here.
- **`hcm-theme`** — `main` + our brand customizations. **Everything we ship comes from this branch.**

Pulling a new Fider release works in three stages:

1. **Update `main`** to match upstream Fider.
2. **Merge `main` into `hcm-theme`**, resolving any conflicts in the files [[HCM-Fider-Theme#3.2 Files we modified (in the Fider source tree)|we modified]].
3. **Rebuild the Docker image and deploy to VM 201.**

That's it. The rest of this document is just the buttons to press.

> [!tip] Average time investment
> If upstream didn't touch any of the files we modified: ~10 minutes including the Docker build.
> If they did: ~30–60 minutes to resolve conflicts and verify.

---

## 1. One-time setup (skip if already done)

Do these once per laptop. After this is set up, future updates skip straight to Section 2.

### 1.1 Tools you need installed

- **Git** (any recent version)
- **Docker Desktop**, running, logged into `git.hcm.adev`
- A text editor (VS Code, Cursor, Notepad++ — anything)

### 1.2 Clone the repo (if you don't have it locally)

```bash
git clone https://git.hcm.adev/hody/fider
cd fider
git checkout hcm-theme
```

### 1.3 Add upstream as a remote (if it isn't already)

```bash
git remote -v
```

You should see two remotes:
```
origin    https://git.hcm.adev/hody/fider (fetch)
origin    https://git.hcm.adev/hody/fider (push)
upstream  https://github.com/getfider/fider (fetch)
upstream  https://github.com/getfider/fider (push)
```

If `upstream` is missing, add it:
```bash
git remote add upstream https://github.com/getfider/fider
```

### 1.4 Log Docker into the HCM registry (once per laptop)

```bash
docker login git.hcm.adev
```
Use your Gitea/Forgejo username and either your password or a personal access token.

---

## 2. The update procedure

### 2.1 Make sure your working tree is clean

```bash
git status
```

If anything's uncommitted: commit it, stash it, or discard it. Don't start an update with dirty changes.

### 2.2 Pull the latest from upstream Fider

```bash
git fetch upstream
git fetch origin
```

Look at what's new:

```bash
git log --oneline main..upstream/main | head -30
```

That shows you the commits Fider added since your last sync. If the list is empty, you're already up to date — stop here.

> [!tip] What to look for in the upstream commits
> Scan the messages. If you see anything touching:
> - `public/assets/styles/`
> - `public/pages/Home/Home.page.tsx`
> - `public/components/VoteCounter.tsx`
> - `public/components/ShowPostResponse.tsx`
>
> …expect to resolve a merge conflict in step 2.4.

### 2.3 Fast-forward `main`

```bash
git checkout main
git merge --ff-only upstream/main
git push origin main
```

> [!warning] If `--ff-only` fails
> That means someone committed directly to `main` (which they shouldn't). Open `git log main..upstream/main` and `git log upstream/main..main` to see what's where, then decide whether to reset `main` to `upstream/main` (preferred, since `main` is supposed to be a mirror) or do a real merge.

### 2.4 Merge `main` into `hcm-theme`

```bash
git checkout hcm-theme
git merge main
```

Three things can happen:

#### A. Clean merge (lucky path)

You'll see a merge commit message in your editor. Save and close it. Move to step 2.5.

#### B. Merge conflicts (likely if upstream touched our files)

Git will print:
```
CONFLICT (content): Merge conflict in <file>
Automatic merge failed; fix conflicts and then commit the result.
```

The files most likely to conflict are listed in [[HCM-Fider-Theme#3.2 Files we modified (in the Fider source tree)|HCM-Fider-Theme § 3.2]]. Open each one in your editor. Conflict markers look like:

```
<<<<<<< HEAD
(our hcm-theme version)
=======
(upstream's new version)
>>>>>>> main
```

For each conflict, pick the right side — usually **keep the new upstream code structure** but **re-apply our changes on top of it**. Our changes are summarized below:

| File | What we changed |
| --- | --- |
| `public/assets/styles/index.scss` | Last line: `@import "hcm-theme.scss";` |
| `public/pages/Home/Home.page.tsx` | (1) Welcome message is wrapped in `<div className="p-home__welcome-card">`. (2) The submit button's inner `<span>` reads literally `Submit Your Idea` instead of `{fider.session.tenant.invitation \|\| defaultInvitation}`. (3) The button's `<HStack>` uses `spacing={2} align="center" justify="center"`. |
| `public/components/VoteCounter.tsx` | `<span className="c-vote-counter__count">…</span>` is rendered as a **sibling** of the `<button>`, not inside it. Two fragments (`vote` and `disabled`) both use `<>…</>` wrappers with the span after the button. |
| `public/components/ShowPostResponse.tsx` | New `extractSubstage()` helper function near the top. `ResponseLozenge` computes a `substage` variable from `props.response?.text` and conditionally renders `<span className="c-status-substage"> · {substage}</span>` inside the lozenge's main span. |
| `public/components/common/PoweredByFider.tsx` | Version parse changed from `.split("-")[0]` to `.replace(/-[0-9a-f]{7,}$/, "")` so the `-hcm.N` suffix survives. If upstream rewrites how the version string is shown, re-apply the regex form. |

> [!tip] Use the patch as a reference
> `hcm-theme.patch` in the repo root holds the diff that originally added the SCSS theme + `index.scss` import. It won't apply cleanly after rebases, but it's a quick way to remind yourself what the original change looked like.

After resolving each conflict, stage the file:

```bash
git add <file>
```

When all conflicts are resolved:

```bash
git commit
```

(Don't pass `-m` — let git use the prepared merge commit message.)

#### C. The build later fails

Don't panic. Section 5 covers diagnosis.

### 2.5 Quick sanity check before building

Verify our changes are still present:

```bash
grep "hcm-theme.scss"          public/assets/styles/index.scss
grep "Submit Your Idea"        public/pages/Home/Home.page.tsx
grep "c-status-substage"       public/components/ShowPostResponse.tsx
grep "p-home__welcome-card"    public/pages/Home/Home.page.tsx
grep "0-9a-f]{7,}"             public/components/common/PoweredByFider.tsx
```

All five should print at least one line. If any prints nothing, you lost that edit in the merge — open the file and add it back. See [[HCM-Fider-Theme#3.2 Files we modified (in the Fider source tree)|the file-change reference]] for what each looked like.

### 2.6 Decide the new version number

We follow a `vX.Y.Z-hcm.N` scheme where `vX.Y.Z` is the **upstream Fider tag** you just merged and `N` is our iteration counter against it.

```bash
# Confirm which upstream tag your main is sitting on
git describe --tags --abbrev=0 main           # e.g. "v0.36.0"

# Look at the most recent HCM tag for this base
git tag --list "v0.36.0-hcm*" --sort=-v:refname | head -3

# If the list is empty → your tag is v0.36.0-hcm.1
# If you see v0.36.0-hcm.2 → your tag is v0.36.0-hcm.3
```

Hold onto two values for the next steps:

```bash
NEW_VERSION="v0.36.0-hcm.1"                   # adjust to match the above
COMMITHASH=$(git rev-parse --short hcm-theme) # 8-char hash of current hcm-theme HEAD
```

### 2.7 Tag the commit

```bash
git tag "$NEW_VERSION"
```

> [!important] Tag BEFORE the build.
> The COMMITHASH baked into the binary should match the tagged commit. Doing it in this order keeps the displayed version and the git history aligned.

### 2.8 Build the new image (with version baked in)

```bash
set -o pipefail
docker build \
  --build-arg VERSION="$NEW_VERSION" \
  --build-arg COMMITHASH="$COMMITHASH" \
  -t "git.hcm.adev/hody/fider:stable" \
  -t "git.hcm.adev/hody/fider:$NEW_VERSION" \
  . 2>&1 | tee /tmp/fider-build.log
```

> [!danger] Use `set -o pipefail`
> Without it, if `docker build` fails the pipeline still exits 0 because `tee` succeeded — you'll think the build worked and push an outdated image. With `pipefail`, the pipeline's exit code reflects the *first* failing command.

> [!note] Why pass both `VERSION` and `COMMITHASH`?
> The Fider Go binary reads these via `-ldflags` (see Makefile line 5–6). Without them, the version shown in the footer / API and on `./fider --version` defaults to literal `"dev"` — that's why our pre-versioning builds all reported `dev`.

This takes 5–15 minutes the first time after an upstream update (npm + Go caches get invalidated). Subsequent builds with no source changes finish in ~30 seconds.

If it fails, jump to [[#5. When things go wrong]].

### 2.9 Push the image (both tags)

```bash
docker push "git.hcm.adev/hody/fider:stable"
docker push "git.hcm.adev/hody/fider:$NEW_VERSION"
```

Watch the output. You want to see new layer hashes followed by `Pushed`. If you only see `Layer already exists`, the build didn't actually rebuild anything (probably because step 2.8 silently failed and we didn't catch it).

Note the digest printed at the end — something like `sha256:abcd…`. That's the new image you just pushed.

The `:stable` tag is what VM 201 pulls automatically. The `:vX.Y.Z-hcm.N` tag is your immutable rollback point.

### 2.10 Push your branch and tag

```bash
git push origin hcm-theme
git push origin "$NEW_VERSION"
```

### 2.11 Publish the Gitea release

Gitea separates **tags** (markers on a commit) from **releases** (titled, notable entries on the repo's Releases page). Step 2.10 pushed the tag — now turn it into a release.

#### Option A: Web UI (easiest, no scripting)

1. Open https://git.hcm.adev/hody/fider/releases/new
2. **Tag**: pick `v0.35.0-hcm.1` (or whatever `$NEW_VERSION` you chose) from the dropdown — *don't create a new one*
3. **Title**: `v0.35.0-hcm.1 — HCM brand on Fider v0.35.0`
4. **Description**: paste a short summary. A reusable template:
   ```markdown
   HCM build on upstream Fider <upstream-version>.

   **Image:** `git.hcm.adev/hody/fider:<NEW_VERSION>` (also pushed as `:stable`)
   **Digest:** `sha256:<from-docker-push-output>`

   ## What changed since the previous release
   - <bullet>
   - <bullet>

   ## Deploy
   ```bash
   cd /opt/fider && sudo docker compose pull && sudo docker compose up -d
   ```
   ```
5. Click **Publish Release**.

#### Option B: API (scriptable, one command)

If your Windows Credential Manager already has `git.hcm.adev` saved (because `git push` works without a prompt), you can publish a release without opening the browser:

```bash
CREDS=$(printf 'protocol=https\nhost=git.hcm.adev\n\n' | git credential fill 2>/dev/null)
USER=$(echo "$CREDS" | sed -n 's/^username=//p')
PASS=$(echo "$CREDS" | sed -n 's/^password=//p')

cat > /tmp/release.json <<EOF
{
  "tag_name": "$NEW_VERSION",
  "name": "$NEW_VERSION — HCM brand on Fider <upstream-version>",
  "body": "Short release summary in markdown.\\n\\nImage: \\\`git.hcm.adev/hody/fider:$NEW_VERSION\\\`",
  "draft": false,
  "prerelease": false
}
EOF

curl -sk -u "$USER:$PASS" -H "Content-Type: application/json" \
  -X POST --data-binary @/tmp/release.json \
  https://git.hcm.adev/api/v1/repos/hody/fider/releases
```

The response includes the `html_url` of the published release.

### 2.12 Deploy to VM 201

SSH to VM 201, then:

```bash
cd /opt/fider
sudo docker compose pull
sudo docker compose up -d
sudo docker compose ps
```

You want the `fider` service status to say `running`. Visit the site in a browser and hard-refresh (Ctrl-F5) to bust the asset cache.

---

## 3. Verification checklist (the "did it actually work?" list)

Go through this after every deploy. It takes 60 seconds and will save you from shipping a broken site.

- [ ] Home page loads without an error banner.
- [ ] Theme toggle works (try light → dark → light).
- [ ] In **light mode**, the "Submit Your Idea" button shows a green border and green text. Hover fills green with white text.
- [ ] In **dark mode**, the "Submit Your Idea" button shows a bright-green border and bright-green text. Hover fills green with white text.
- [ ] Clicking the button opens the share-feedback modal.
- [ ] An idea card on the home page shows: title, body, vote count (green), and a status pill if anything but Open.
- [ ] Clicking a card opens the post detail page.
- [ ] The post title on the detail page is **smaller** than the body, not a giant banner.
- [ ] If you have a Plane-driven post in "Started", the status pill reads `Started · <substage>` and the substage stays mixed-case.
- [ ] The bottom action bar (Copy link / Edit / Respond / Comment Feed / Delete) — hover any one. Should outline + fill brand green (red for Delete).
- [ ] No console errors in DevTools.
- [ ] In the footer (or on `/api/v1/_version`), the Fider version reads `vX.Y.Z-hcm.N` — **not `dev`**. If it says `dev`, the build skipped the `--build-arg VERSION=` and `COMMITHASH=` flags; rebuild and redeploy.

If anything fails: roll back per [[#6. Rolling back]].

---

## 4. Provisioning a fresh VM (rare path)

You're on a new VM that has Docker installed but no `/opt/fider`. One-time setup:

```bash
sudo mkdir -p /opt/fider
sudo nano /opt/fider/docker-compose.yml
```

Paste a compose file pointing at our image:

```yaml
services:
  fider:
    image: git.hcm.adev/hody/fider:stable
    ports:
      - "3000:3000"
    environment:
      # …all the env vars your Fider needs (DATABASE_URL, JWT_SECRET, etc.)
    restart: unless-stopped
```

Then:

```bash
sudo docker login git.hcm.adev
sudo docker compose -f /opt/fider/docker-compose.yml pull
sudo docker compose -f /opt/fider/docker-compose.yml up -d
```

Alternatively, `scripts/update-fider.sh` in this repo does the whole bootstrap (clones the repo, builds locally on the VM, rewrites compose). Slower but self-contained.

---

## 5. When things go wrong

### 5.1 Merge conflict you can't figure out

1. Abort the merge: `git merge --abort`.
2. You're back where you started, safe.
3. Open the conflicting file in upstream:
   ```bash
   git show upstream/main:<path-to-file>
   ```
4. Look at our diff between `main` and `hcm-theme` for that file:
   ```bash
   git diff main hcm-theme -- <path-to-file>
   ```
5. Mentally combine them, then re-run the merge and edit by hand.

### 5.2 `docker build` fails with a TypeScript / webpack error

Read the error carefully — it'll point at a file and line number. Common causes after a rebase:

- **HStack/VStack prop mismatch.** Their type accepts `spacing: 0 | 1 | 2 | 4 | 6 | 8` only (no 3 or 5). Adjust.
- **Removed component.** Upstream renamed or deleted a component we imported in one of our modified files. Update the import.
- **Missing image asset.** Upstream renamed an icon SVG. Update the import path.

### 5.3 Build "succeeds" but push says `Already exists` for every layer

You hit the `tee` pipe-failure bug (see [[#2.8 Build the new image (with version baked in)]] warning). Rerun the build with `set -o pipefail` first and look at the *real* exit code.

### 5.4 `docker compose pull` on VM 201 says `unauthorized`

The VM's Docker daemon hasn't logged into the registry. SSH in and run:
```bash
sudo docker login git.hcm.adev
```

### 5.5 New image deployed but the site still looks old

Hard-refresh the browser (Ctrl-F5 / Cmd-Shift-R). Fider serves hashed asset URLs but the browser may cache the HTML for a few minutes.

If a hard-refresh doesn't help:
```bash
sudo docker compose ps
sudo docker compose logs --tail=100 fider
```

Confirm the running image matches the digest you just pushed:
```bash
sudo docker compose images fider
```

### 5.6 New release totally breaks something

Roll back. See next section.

---

## 5b. Phase 2 + v0.36.0 cutover (v0.36.0-hcm.1, one-time)

> [!warning] This section only applies to the **first** cutover that ships custom statuses end-to-end + the upstream v0.36.0 roadmap-v2 release. It runs DDL migrations that drop the legacy `posts.status` int column and `statuses.legacy_enum`, plus adds tenant deletion and description-template columns. Once it lands, the regular procedure in §2 covers everything; this section can be deleted.

**What's different from a normal release.**

- HCM Phase 2 migrations `202606231200`..`202606231500`: new `statuses` table, new `posts.status_slug` column with backfill, defensive collapse of the HCM-only `PostReview=7` enum, then a DROP of `posts.status` and `statuses.legacy_enum`, plus `statuses.show_on_roadmap` (admin opt-in for each lane on the Roadmap page).
- Upstream v0.36.0 migrations `202606101200` + `202606121200` were tagged before our hcm-beta ones; **migration `202606241400`** catches them up with `IF NOT EXISTS` guards so the tenant gets `description_template` + `scheduled_deletion_at` + `deletion_requested_by` + `deletion_cancel_key`.
- Once those migrations apply, you cannot start an older image again — the new code is the only thing that knows about `status_slug` and the new tenant columns. **You must back up the DB before pulling the new image** so rollback is possible.
- Webhook payload gains `post_status_slug`, `post_status_kind`, `post_status_label`, `post_old_status_slug`, `post_old_status_label`. The legacy `post_status`/`post_old_status` keys are gone. If Plane (or any other receiver) maps on the legacy key, update it before cutover.
- v0.36.0 adds /roadmap (auto-unlocked for self-hosted), better tag admin, and a self-service Danger Zone for tenant deletion. Danger Zone is gated to multi-host mode upstream, so it stays hidden on ideas.hcm.adev.

**Pre-cutover checklist** — run in order:

1. **Back up the prod DB on VM 204.** Compose includes the postgres service; dump the live volume to a host file:
   ```bash
   ssh -i ~/.ssh/compass_deploy root@10.10.40.200 'ssh root@<vm-204-ip> "cd /opt/fider && docker compose exec -T db pg_dump -U postgres fider | gzip > /root/fider-pre-phase2-$(date +%Y%m%d-%H%M).sql.gz"'
   ssh -i ~/.ssh/compass_deploy root@10.10.40.200 'ssh root@<vm-204-ip> ls -lh /root/fider-pre-phase2-*.sql.gz'
   ```
2. **Snapshot the VM in Proxmox** so the whole disk is reversible if the migration corrupts anything unforeseen:
   ```
   Proxmox web UI → VM 204 → Snapshots → Take Snapshot → name: pre-phase2-<date>
   ```
   Or `qm snapshot 204 pre-phase2-$(date +%Y%m%d-%H%M) --vmstate 1` from the host.
3. **Set `WEBHOOK_ALLOW_PRIVATE_IPS=true`** in `/opt/fider/docker-compose.yml`'s `fider` service env block (Plane lives on a private network; PR #0 makes the SSRF block opt-out). Restart will pick this up when we recompose below.
4. **Update the Plane webhook** to consume `post_status_slug` instead of `post_status` (or duplicate the mapping). The legacy keys are removed in v0.35.0-hcm.3.
5. **Confirm beta has run the same image stable** for at least 24h with no regressions. Beta = ideas-beta.hcm.adev, VM 214, running `git.hcm.adev/hody/fider:beta`.

**Build and tag the release image** (on laptop, on `hcm-theme` after merging `hcm-beta` in):

```bash
git checkout hcm-theme
git merge hcm-beta                                # bring Phase 2 commits in
NEW_VERSION="v0.36.0-hcm.1"
COMMITHASH=$(git rev-parse --short hcm-theme)
git tag "$NEW_VERSION"
set -o pipefail
docker build --build-arg VERSION="$NEW_VERSION" --build-arg COMMITHASH="$COMMITHASH" \
  -t "git.hcm.adev/hody/fider:stable" -t "git.hcm.adev/hody/fider:$NEW_VERSION" \
  . 2>&1 | tee /tmp/build.log
docker push "git.hcm.adev/hody/fider:stable"
docker push "git.hcm.adev/hody/fider:$NEW_VERSION"
git push origin hcm-theme && git push origin "$NEW_VERSION"
```

**Cutover on VM 204** (requires explicit "go on prod" approval in chat):

**Step C1 — turn on maintenance mode (optional but recommended).** Locks the site behind Fider's built-in 503 page while the migration runs (~30 s on a paged-out VM, instant on warm cache). Append to `services.fider.environment` in `/opt/fider/docker-compose.yml`:

```yaml
      MAINTENANCE: "true"
      MAINTENANCE_MESSAGE: "Brief maintenance — back at 1:00 PM EDT."
      MAINTENANCE_UNTIL: "2026-06-24T17:00:00Z"   # ISO-8601 in UTC; 17:00Z = 1pm EDT
```

Then recreate the container — this picks up the env change without pulling the new image yet:

```bash
ssh -i ~/.ssh/compass_deploy root@10.10.40.200 'ssh root@<vm-204-ip> "cd /opt/fider && docker compose up -d && sleep 4 && curl -sS https://ideas.hcm.adev/ -o /dev/null -w \"http %{http_code}\n\""'
# expect: http 503
```

**Step C2 — pull the new image and apply migrations.**

```bash
ssh -i ~/.ssh/compass_deploy root@10.10.40.200 'ssh root@<vm-204-ip> "cd /opt/fider && docker compose pull && docker compose up -d && sleep 5 && docker logs fider-fider-1 2>&1 | tail -40"'
```

Watch the logs for `running migration 202606241500` (the last new one) then `server started`. The catch-up migration `202606241400` and the show-on-roadmap migration `202606241500` are the two extras vs. a plain `:beta` pull. The first request after restart will JIT-compile the new SSR bundle (typically <30 s).

**Step C3 — verify smoke test (still in maintenance — admins not bypassed).** Until maintenance is off you cannot hit the home page through a browser, so verify via the host shell:

```bash
ssh -i ~/.ssh/compass_deploy root@10.10.40.200 'ssh root@<vm-204-ip> "docker exec fider-fider-1 wget -qO- http://localhost:3000/_health && docker logs fider-fider-1 2>&1 | tail -5"'
# expect: OK
```

`_health` is the only route that doesn't go through the maintenance middleware. Container running + `OK` body + log line `server started` is enough to take maintenance off.

**Step C4 — disable maintenance and verify the live site.** Strip the three vars and recreate:

```bash
ssh -i ~/.ssh/compass_deploy root@10.10.40.200 'ssh root@<vm-204-ip> "sed -i \"/MAINTENANCE:/d;/MAINTENANCE_MESSAGE:/d;/MAINTENANCE_UNTIL:/d\" /opt/fider/docker-compose.yml && cd /opt/fider && docker compose up -d && sleep 4 && curl -sS https://ideas.hcm.adev/ -o /dev/null -w \"http %{http_code}\n\""'
# expect: http 200
```

**Verify** — `https://ideas.hcm.adev/` returns 200, hover the version footer to check the `v0.36.0-hcm.1` tag, hit `/admin/statuses` and confirm the six built-ins seeded, `/roadmap` renders the Planned/Started/Completed lanes, and Plane receives a `post_status_slug` payload on the next status change.

**Rollback path** (if migration breaks or smoke test fails):

```bash
# 1. Stop the new container
ssh -i ~/.ssh/compass_deploy root@10.10.40.200 'ssh root@<vm-204-ip> "cd /opt/fider && docker compose down"'
# 2. Restore the DB dump from step 1 of the pre-cutover checklist
ssh -i ~/.ssh/compass_deploy root@10.10.40.200 'ssh root@<vm-204-ip> "cd /opt/fider && docker compose up -d db && sleep 5 && gunzip -c /root/fider-pre-phase2-<timestamp>.sql.gz | docker compose exec -T db psql -U postgres fider"'
# 3. Retag :stable to the previous v0.35.0-hcm.2 image and redeploy
docker pull git.hcm.adev/hody/fider:v0.35.0-hcm.2
docker tag  git.hcm.adev/hody/fider:v0.35.0-hcm.2 git.hcm.adev/hody/fider:stable
docker push git.hcm.adev/hody/fider:stable
ssh -i ~/.ssh/compass_deploy root@10.10.40.200 'ssh root@<vm-204-ip> "cd /opt/fider && docker compose pull && docker compose up -d"'
```

If the Proxmox snapshot is more recent than any data you'd lose, restoring the snapshot is the fastest path — VM goes back to its pre-cutover state in ~30 seconds.

---

## 6. Rolling back

> [!important] If you tagged the previous release, rollback is a one-liner on the VM.

If your previous release was tagged `hcm-v0.3.0`:

```bash
# On your laptop, retag :stable to point at the old commit
git checkout hcm-v0.3.0
docker build -t git.hcm.adev/hody/fider:stable .
docker push git.hcm.adev/hody/fider:stable
```

Then on VM 201:
```bash
sudo docker compose pull
sudo docker compose up -d
```

If you also kept the previous image locally on your laptop, you can skip the rebuild:
```bash
docker tag <previous-digest-or-old-tag> git.hcm.adev/hody/fider:stable
docker push git.hcm.adev/hody/fider:stable
```

---

## 7. Cheat sheet (paste-this version)

For when you've done this twice and don't want to read the long version. Set `NEW_VERSION` first.

```bash
# ── On laptop, on hcm-theme branch, clean working tree ──────────────
git fetch upstream && git fetch origin
git checkout main && git merge --ff-only upstream/main && git push origin main
git checkout hcm-theme && git merge main              # resolve conflicts, commit

# Sanity check that our edits survived the merge
grep -q "hcm-theme.scss"      public/assets/styles/index.scss        || echo "LOST IMPORT"
grep -q "Submit Your Idea"    public/pages/Home/Home.page.tsx        || echo "LOST BUTTON TEXT"
grep -q "c-status-substage"   public/components/ShowPostResponse.tsx || echo "LOST SUBSTAGE"
grep -q "p-home__welcome-card" public/pages/Home/Home.page.tsx       || echo "LOST CARD WRAPPER"
grep -q "0-9a-f]{7,}"          public/components/common/PoweredByFider.tsx || echo "LOST VERSION REGEX"

# Decide version (vX.Y.Z = upstream tag, .N = our iteration)
NEW_VERSION="v0.36.0-hcm.1"                            # edit me
COMMITHASH=$(git rev-parse --short hcm-theme)

git tag "$NEW_VERSION"

# Build + push (both tags)
set -o pipefail
docker build --build-arg VERSION="$NEW_VERSION" --build-arg COMMITHASH="$COMMITHASH" \
  -t "git.hcm.adev/hody/fider:stable" -t "git.hcm.adev/hody/fider:$NEW_VERSION" \
  . 2>&1 | tee /tmp/build.log
docker push "git.hcm.adev/hody/fider:stable"
docker push "git.hcm.adev/hody/fider:$NEW_VERSION"
git push origin hcm-theme
git push origin "$NEW_VERSION"

# Publish the Gitea release (uses saved git credentials)
CREDS=$(printf 'protocol=https\nhost=git.hcm.adev\n\n' | git credential fill 2>/dev/null)
USER=$(echo "$CREDS" | sed -n 's/^username=//p'); PASS=$(echo "$CREDS" | sed -n 's/^password=//p')
cat > /tmp/release.json <<EOF
{"tag_name":"$NEW_VERSION","name":"$NEW_VERSION","body":"HCM build on upstream Fider.","draft":false,"prerelease":false}
EOF
curl -sk -u "$USER:$PASS" -H "Content-Type: application/json" \
  -X POST --data-binary @/tmp/release.json \
  https://git.hcm.adev/api/v1/repos/hody/fider/releases

# ── On VM 201 ──────────────────────────────────────────────────────
cd /opt/fider && sudo docker compose pull && sudo docker compose up -d && sudo docker compose ps
```

Then walk through [[#3. Verification checklist (the "did it actually work?" list)|the verification checklist]].

---

## 8. Glossary

- **Upstream**: the original Fider repo at `github.com/getfider/fider`.
- **Origin**: our fork at `git.hcm.adev/hody/fider`.
- **`main`**: our local mirror of upstream Fider. We never edit it.
- **`hcm-theme`**: the deployable branch. Everything HCM ships from here.
- **`hcm-vX.Y.Z`**: a tag marking a deployable release. Tag whenever you ship.
- **HCM image**: `git.hcm.adev/hody/fider:stable`. The Docker image VM 201 pulls.
- **Custom CSS field**: Fider's built-in admin setting at Site Settings → Advanced Settings → Custom CSS. Accepts paste-ready CSS (`hcm-theme.css`) as a no-rebuild fallback for selector-level theming.
