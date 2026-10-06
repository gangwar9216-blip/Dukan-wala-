# Clean Render deployment

This backend is intentionally packaged so the backend files live at the repository root.
Do **not** put them inside another `backend/` or `.backend/` folder.

After uploading the contents of this package to the GitHub repository root, the repo should look like:

```text
go.mod
cmd/
internal/
migrations/
deploy/
Dockerfile
.env.example
README.md
RENDER_DEPLOY.md
docker-compose.yml
```

Keep the existing `dashboard/` folder in the GitHub repository.

## Render Web Service

- Root Directory: **blank**
- Build Command:

```bash
go mod tidy && go build -o app ./cmd/server
```

- Start Command:

```bash
./app
```

The server automatically runs the embedded PostgreSQL migrations at startup.

## Environment variables

Keep/use the existing Render `DATABASE_URL`.

Also set:

```text
CORS_ORIGINS=https://healthcare-admin-dashboard.onrender.com
JWT_SECRET=<strong-random-secret>
APP_ENV=production
OTP_DEV_MODE=true
```

`OTP_DEV_MODE=true` is only for testing. Before production, connect a real OTP/SMS provider and set it to `false`.

Razorpay variables can be added when production payments are configured:

```text
RAZORPAY_KEY_ID=
RAZORPAY_KEY_SECRET=
RAZORPAY_WEBHOOK_SECRET=
```

## First tests after deploy

1. `GET /health` should return JSON with `status: ok`.
2. `POST /api/v1/auth/request-otp` should work.
3. `POST /api/v1/auth/verify-otp` should return a JWT in development OTP mode.
4. Authenticated `GET /api/v1/plans` should return the Basic/Pro plans.
