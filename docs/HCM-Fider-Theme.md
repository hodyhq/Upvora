# HCM Fider Theme — What We Built

> [!info] One-line summary
> A branded skin for our internal Fider instance, built as a tracked fork that survives upstream updates via `git rebase` and ships as a self-built Docker image.

Related: [[Update-Fider]] — how to pull a new Fider release into this fork.

---

## 1. The big picture

We forked the open-source Fider feedback tool and applied an HCM-branded skin to it. The fork lives in our internal Forgejo/Gitea instance and is deployed as a custom Docker image to **VM 201**.

```
┌───────────────────────────────┐         ┌──────────────────────────┐
│  github.com/getfider/fider    │ upstream│  Original Fider source   │
│  (the "real" Fider project)   │ ───────▶│  (we pull from here)     │
└──────────────┬────────────────┘         └──────────────────────────┘
               │
               │ git fetch upstream && git merge / rebase
               ▼
┌────────────────────────────────────────────────────────────────────┐
│  git.hcm.adev/hody/fider  (this repo)                              │
│   • main       — mirrors upstream Fider                            │
│   • hcm-theme  — our branding + small TSX tweaks live here         │
│   • tags hcm-vX.Y.Z mark deployable HCM releases                   │
└─────────────────────────────────┬──────────────────────────────────┘
                                  │
                                  │ docker build + docker push
                                  ▼
┌────────────────────────────────────────────────────────────────────┐
│  Docker registry: git.hcm.adev/hody/fider:stable                   │
└─────────────────────────────────┬──────────────────────────────────┘
                                  │
                                  │ docker compose pull on VM 201
                                  ▼
┌────────────────────────────────────────────────────────────────────┐
│  Production Fider instance (VM 201, /opt/fider)                    │
└────────────────────────────────────────────────────────────────────┘
```

---

## 2. HCM brand palette

These are the only hex values you should ever copy into theme code. They're declared as SCSS variables at the top of `public/assets/styles/hcm-theme.scss`.

| Token              | Hex       | Used for                          |
| ------------------ | --------- | --------------------------------- |
| `$hcm-green-bright`| `#00F7A5` | Dark-mode accents, active states  |
| `$hcm-green-mid`   | `#008859` | Primary buttons, brand borders    |
| `$hcm-green-dark`  | `#004C2C` | Hover fills, deep fills           |
| `$hcm-dark-bg`     | `#111111` | Dark-mode page background         |
| `$hcm-dark-panel`  | `#161616` | Dark-mode card/panel background   |
| `$hcm-dark-border` | `#1E1E1E` | Dark-mode panel borders           |
| `$hcm-dark-input`  | `#0D1F17` | Dark-mode input surfaces          |
| `$hcm-light-bg`    | `#FAFAFA` | Light-mode page background        |
| `$hcm-light-panel` | `#FFFFFF` | Light-mode cards                  |
| `$hcm-light-border`| `#E5E5E5` | Light-mode panel borders          |

> [!warning] Don't sprinkle hex codes throughout the codebase.
> If you need a new shade, add it as a `$hcm-*` variable in `hcm-theme.scss` first, then reference it. The flat `hcm-theme.css` is the only place real hex values appear (because it's a paste-into-admin fallback that can't use SCSS variables).

---

## 3. What was changed and where

Everything HCM-specific is intentionally consolidated into a handful of files so rebases stay tractable.

### 3.1 Files we own (additions)

| File | Purpose |
| --- | --- |
| `public/assets/styles/hcm-theme.scss` | **Source of truth for the theme.** All brand colors, dark/light surface treatments, component overrides. Gets compiled into the Fider bundle via webpack. |
| `hcm-theme.css` | Flattened plain CSS mirror of the SCSS, paste-ready for **Site Settings → Advanced Settings → Custom CSS** as a no-rebuild fallback. Kept in sync by hand. |
| `hcm-theme.patch` | A `git diff` snapshot for re-applying the theme by hand if a rebase implodes. Mostly historical. |
| `scripts/update-fider.sh` | One-shot deploy script intended for VM 201. Pulls a tagged HCM release, builds the Docker image, rewrites `/opt/fider/docker-compose.yml`, and restarts. Not used day-to-day (we build locally and the VM just pulls), but useful as a backup deploy path. |
| `docs/HCM-Fider-Theme.md` | This file. |
| `docs/Update-Fider.md` | The merge-an-upstream-release procedure. |

### 3.2 Files we modified (in the Fider source tree)

| File | Why we touched it |
| --- | --- |
| `public/assets/styles/index.scss` | Added one `@import "hcm-theme.scss";` line at the end so our theme loads last and wins the cascade. |
| `public/pages/Home/Home.page.tsx` | Wrapped the welcome message in a new `.p-home__welcome-card` div (gives us a themed card target). Also changed the home page "submit" button so its label reads **"Submit Your Idea"** instead of repeating the modal's placeholder text. The placeholder is still passed to the modal where the user actually types. |
| `public/components/VoteCounter.tsx` | Moved `<span class="c-vote-counter__count">` *outside* the `<button>` so the count renders as a sibling beneath the 32×32 vote button (instead of overlapping inside the button). |
| `public/components/ShowPostResponse.tsx` | Added an `extractSubstage()` helper that reads the first non-empty line of `post.response.text` (where the Plane integration writes its substage label) and appends it inside the status lozenge as `"Started · In Beta Testing"`. Same colors / shape, just wider. |

That's all of it on the source-code side. **Five files** of edits plus the new theme file. Everything else is CSS rules added into `hcm-theme.scss`.

> [!tip] When you touch the theme, only edit `hcm-theme.scss` and then mirror the change into `hcm-theme.css` by hand. The flat file is the one you'd paste into the admin UI if the rebuilt image were ever unavailable.

---

## 4. The visual specs we implemented

### Header
- Dark `#111` in dark mode, default light header in light mode.
- HCM tenant logo lives in Site Settings, no source changes.

### Home page — welcome card (left column)
- Wrapped in a `.p-home__welcome-card` with 12px radius, 24px padding.
- Card background `#161616` dark / `#FFFFFF` light, 1px subtle border.

### Home page — "Submit Your Idea" CTA (top of right column)
- **Outlined ghost button.** Transparent at rest, 2px brand-green border, brand-green text + plus icon.
- **On hover:** fills brand-green with white text. Slight shadow lift.
- **On focus:** bright-green outline ring.
- The button face says "Submit Your Idea" (hardcoded) regardless of the admin's invitation text. The admin-set invitation still flows through to the `ShareFeedback` modal as the textarea placeholder where the user types.

### Home page — idea cards
- 8px radius, 12×14 padding, 8px between cards.
- Brand-green border on hover.
- Title `font-weight: 500, font-size: 14px`. Description `13px`, dimmed color.

### Home page — vote counter
- 32×32 square button containing only the up-arrow icon.
- Count number rendered as a separate sibling beneath the button, brand-green.
- Voted state fills the button with brand green.
- Detail-page variant (`size="large"`) is 48×48.

### Status lozenges (everywhere they appear)
- Reshaped to small 4px-radius pills, `font-size: 11px`, brand-green palette.
- **Substage support:** if `post.response.text` is set, its first line is appended inside the same pill as `Started · In Beta Testing`. 80% opacity on the substage portion, original casing preserved.

### Post detail page
- Same dark-or-light surfaces as the home page.
- Title scaled down to 18/24/28 across breakpoints (was 20/32/36).
- The "blue 5px accent" above the bottom actions row is replaced with a subtle 1px brand divider.
- **Bottom action bar** (Copy link / Edit / Respond / Comment Feed / Delete) — same outlined-ghost treatment as the home Submit CTA. Brand-green fill on hover (red for the Delete variant).
- **Follow / Following button** — outlined panel with brand-green hover.
- **Response detail card** (the box that shows the admin reply / Plane substage description) — explicit text colors so the markdown body never blends into the card background.

### Sort dropdown
- Stayed a `<Dropdown>` per the iteration decision. Handle (`.c-post-sort-btn`) just gets themed text colors. No structural change.

### Primary buttons globally
- Anywhere `.c-button--primary` appears in the app — brand-green background, white text, dark-green hover.

---

## 5. The deploy pipeline

> [!important] Day-to-day: build locally, push to registry, pull on VM.

```bash
# On your laptop, inside this repo, on hcm-theme:
docker build -t git.hcm.adev/hody/fider:stable .
docker push git.hcm.adev/hody/fider:stable
git push origin hcm-theme
```

```bash
# On VM 201 (/opt/fider):
sudo docker compose pull
sudo docker compose up -d
```

VM 201's `/opt/fider/docker-compose.yml` already points at `image: git.hcm.adev/hody/fider:stable`. If you ever re-provision the VM, run the [[Update-Fider#Provisioning a fresh VM|first-time bootstrap]] block.

> [!danger] Always use `set -o pipefail` (or check the docker build exit code separately) when piping the build to `tee` or `head`.
> Without it, a failing build returns the exit code of the *last* command in the pipeline, which is 0 — so the previous image gets re-pushed and you'll swear you deployed when you didn't. We hit this exact bug. Symptom: digest doesn't change after a successful-looking push.

---

## 6. Light / dark mode coverage

Every brand rule has been written to work in both modes. The mechanism Fider already uses is `body[data-theme="light"]` / `body[data-theme="dark"]` toggled by the theme switcher in the header.

- **Brand-color rules** (primary buttons, vote counter color, status pills, submit CTA) — declared at root scope so they apply in both modes.
- **Surface rules** (page background, card backgrounds, text colors, panel borders) — scoped under `body[data-theme="dark"]` so we leave Fider's clean light surfaces alone except where the brand demands a change.

Switch the theme in the header to verify both states whenever you touch a rule.

---

## 7. Things to know when editing

> [!note] CSS specificity gotcha
> Fider's per-page SCSS (e.g. `Home.page.scss`) is imported by the page's TSX file and gets bundled *after* the global stylesheet that contains our `hcm-theme.scss`. On equal selector specificity, those rules win. If a rule "looks right but doesn't apply," check whether a same-specificity selector exists in the original page SCSS — and either bump our selector's specificity (e.g. add a `body[data-theme="…"]` scope) or use `!important` on the override.

> [!note] Don't modify admin-controlled text
> The welcome message, invitation, terms, etc. are stored in the database (`fider.session.tenant.*`). Leave those strings alone in source. The only exception we made: the home submit button's *button face* was hardcoded to "Submit Your Idea" because using the admin's placeholder there made the page look like a typeable input when it isn't.

> [!note] Vote counter restructure
> The change in `VoteCounter.tsx` is the only "DOM reparenting" we did. It was unavoidable: CSS can't move a node out of its parent, and we need the count beneath the button (not inside it) for the layout to work. If you ever rebase and that file conflicts, the diff is dead simple — count goes from being a child of the button to being a sibling.

> [!note] Substage parser
> `extractSubstage()` in `ShowPostResponse.tsx` reads the first non-empty line of `post.response.text`. If your Plane workflow ever changes where it puts the substage label (e.g. as a markdown heading or with a different prefix), update that function. It also strips `*`, `_`, and `` ` `` so simple bold/italic markdown doesn't leak into the lozenge.

---

## 8. Quick reference

| Question | Answer |
| --- | --- |
| Where does the theme code live? | `public/assets/styles/hcm-theme.scss` (source) + `hcm-theme.css` (flat mirror). |
| What's the deploy branch? | `hcm-theme` on `origin` (= `https://git.hcm.adev/hody/fider`). |
| Where is the image? | `git.hcm.adev/hody/fider:stable`. |
| How do I update the VM with a new theme tweak? | Edit SCSS → build → push → `docker compose pull && up -d` on VM 201. |
| How do I pick up a new upstream Fider release? | See [[Update-Fider]]. |
| Where's the mockup that defined the look? | `screenshots/` in this repo, plus `instructions/instructions.md`. |
| Where do I paste CSS if I can't rebuild? | Site Settings → Advanced Settings → Custom CSS field. Paste contents of `hcm-theme.css`. Some changes (e.g. relocated DOM, restructured PostsSort) require the rebuilt image; selector-level rules work fine via Custom CSS alone. |
