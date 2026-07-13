<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="etc/readme/header-dark.png">
    <img src="etc/readme/header-light.png" width="560" alt="Upvora">
  </picture>
</p>

<h1 align="center">Open-source customer feedback, feature voting &amp; product roadmap software</h1>

<p align="center">
  <b>Self-hosted. Multi-product. AGPL.</b> A fork of <a href="https://github.com/getfider/fider">Fider</a> that kept the engine and rebuilt the experience.
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#upvora-and-fider">Upvora and Fider</a> •
  <a href="#getting-started-self-hosted">Getting started</a> •
  <a href="#development">Development</a> •
  <a href="#license--attribution">License</a>
</p>

![The Upvora feedback board](etc/readme/board-dark.png)

**Upvora** is a feedback portal you run yourself: your users post ideas and feature requests, vote and discuss, and follow every decision through a public roadmap — while your team prioritizes with a built-in scorecard and keeps its internal deliberation private. One Docker image, one Postgres database, your domain, your data.

If you're evaluating tools like Canny, UserVoice, Fider, Featurebase, or Astuto and want something **open-source and self-hosted** — or you're a Fider user curious what a heavily-extended fork looks like — this is for you.

## Features

**Feedback board** — ideas with voting, rich-text discussion, similar-post search, tags, and configurable statuses (define your own workflow, colors, and which statuses appear on the home page). Trending / most-wanted / most-discussed views, powerful filtering, optional post moderation.

**Multi-product** — run several products in one portal. Products are a lens, not a wall: one member list, one tag set, one workflow, but every idea belongs to a product, every product gets its own public board at `/p/your-product`, and every view — board, roadmap, scorecard — filters to any combination of products. Ideas without a product land in a General bucket.

**Public roadmap** — a lane view of your statuses with search, tag and product filters, inline voting, and drag-to-restatus for your team. Show your users what's planned, started, and shipped without leaving the portal.

**Prioritization scorecard** — a committee-style scoring workspace your visitors never see: define your own dimensions and fields, score candidate ideas, and watch the weighted ring tell you what to build next. Linked to the board, so scores and internal notes stay attached to the idea.

**Internal notes & comments** — a team-only layer on every idea. Internal comments live in the public discussion thread but are visible only to your team, excluded from public counts, and never trigger notifications. Each idea also carries one shared internal note, synced live between the idea page and the scorecard.

**Theme studio** — brand color and per-function accent colors (buttons, votes, links, header) applied as design tokens across light and dark mode, plus per-tenant custom CSS for everything else. Light, dark, or follow-the-system by default — your choice.

**One-click updates** — a System panel under the admin menu shows your installed version against the latest GitHub release and, with the bundled updater sidecar, updates your instance from the browser — no SSH required.

**Everything Fider ships** — passwordless email sign-in, OAuth2/OIDC single sign-on, email + web notifications, webhooks, a REST API with per-user API keys, CSV export, invitations, GDPR-friendly privacy controls, and 30+ languages.

## Upvora and Fider

Upvora is a friendly fork of [Fider](https://fider.io) (Go + React, AGPL-3.0). The Fider engine — its data model, API, auth, and operational simplicity — is excellent, and Upvora deliberately stays close enough to keep merging upstream improvements.

What the fork adds on top:

| | Fider | Upvora |
|---|---|---|
| Feedback board, voting, discussion | ✅ | ✅ (redesigned UI, dark-first) |
| Custom statuses & workflow | fixed set | ✅ fully configurable |
| Multiple products per portal | — | ✅ boards, filters, `/p/slug` pages |
| Public roadmap view | — | ✅ lanes, filters, inline voting |
| Prioritization scorecard | — | ✅ custom fields & dimensions |
| Internal notes & team-only comments | — | ✅ |
| Theming | custom CSS | ✅ token-based colors + custom CSS |
| In-app updates | — | ✅ System panel + updater sidecar |

If you want the smallest possible tool with the largest community, use Fider — it's great. If you want the features above out of the box, use Upvora.

## Getting started (self-hosted)

Upvora ships as a single Docker image with a Postgres database.

```yaml
# docker-compose.yml
services:
  db:
    image: postgres:17
    environment:
      POSTGRES_USER: upvora
      POSTGRES_PASSWORD: change-me
      POSTGRES_DB: upvora
    volumes: [db-data:/var/lib/postgresql/data]

  app:
    image: your-registry.example.com/upvora:latest   # build from this repo (see below)
    depends_on: [db]
    ports: ["3000:3000"]
    environment:
      BASE_URL: https://feedback.yourdomain.com
      DATABASE_URL: postgres://upvora:change-me@db:5432/upvora?sslmode=disable
      JWT_SECRET: generate-a-long-random-secret
      EMAIL_NOREPLY: noreply@yourdomain.com
      EMAIL_SMTP_HOST: smtp.yourprovider.com
      EMAIL_SMTP_PORT: "587"
      EMAIL_SMTP_USERNAME: your-smtp-user
      EMAIL_SMTP_PASSWORD: your-smtp-password
      EMAIL_SMTP_ENABLE_STARTTLS: "true"

volumes:
  db-data:
```

```bash
docker build -t your-registry.example.com/upvora:latest .
docker compose up -d
```

The image runs database migrations on start and serves on port `3000`. Put it behind a TLS-terminating reverse proxy (Traefik, Caddy, nginx) and open `BASE_URL`.

**Optional — one-click updates:** add the updater sidecar from [`etc/updater/updater.sh`](etc/updater/updater.sh) (instructions in the file header) and the System panel in the admin settings gains an **Update now** button. The sidecar holds the Docker socket so the app container never has to; the app only writes an empty trigger file.

## Development

Requires Node 22, Go 1.25, and Docker (for Postgres).

```bash
make watch          # hot-reload server + UI
make lint           # golangci-lint + eslint
make test           # server + UI unit tests
```

CI runs the same checks on every merge request into `main`.

## License & attribution

Upvora is licensed under the **GNU Affero General Public License v3.0** — see [`LICENSE`](LICENSE). It is a derivative work of [getfider/fider](https://github.com/getfider/fider) (also AGPL-3.0); the original Fider copyright notices are retained. If you run a modified copy as a network service, the AGPL requires you to make your source available to its users.

Fider is a registered project of its respective maintainers; Upvora is an independent project and is not affiliated with or endorsed by the Fider team.
