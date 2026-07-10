<p align="center">
  <img src="etc/upvora-readme-header.png" width="640" alt="Upvora">
</p>

<p align="center">
  <a href="https://github.com/hodyhq/Upvora">GitHub</a> •
  <a href="https://github.com/hodyhq/Upvora/blob/main/CONTRIBUTING.md">Contributing</a> •
  <a href="https://github.com/hodyhq/Upvora/blob/main/LICENSE">License</a>
</p>

# Upvora — a feedback portal for feature requests and suggestions

**Give your customers a voice and let them tell you what they need.** Upvora is a
self-hosted feedback and roadmap tool: collect ideas, let people vote and discuss,
prioritize with a built-in scorecard, and share what you're building.

Upvora is a Go + React application. It is an independent product built on top of
[Fider](https://fider.io) and extended with features Fider doesn't ship — custom
statuses, an admin-configurable committee **Scorecard**, a site banner, themeable
brand colors, and assorted UX refinements.

# Getting started (self-hosted)

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
    image: registry.example.com/upvora:latest   # your registry
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
docker compose up -d
```

The image runs database migrations on start and serves on port `3000`. Put it behind
a TLS-terminating reverse proxy (Traefik, Caddy, nginx) and open `BASE_URL`.

# Development

Requires Node 22, Go 1.25, and Docker (for Postgres).

```bash
make watch          # hot-reload server + UI
make lint           # golangci-lint + eslint
make test           # server + UI unit tests
```

CI runs the same checks on every merge request into `main`.

# License & attribution

Upvora is licensed under the **GNU Affero General Public License v3.0** — see
[`LICENSE`](LICENSE). It is a derivative work of
[getfider/fider](https://github.com/getfider/fider) (also AGPL-3.0); the original
Fider copyright notices are retained. If you run a modified copy as a network
service, the AGPL requires you to make your source available to its users.
