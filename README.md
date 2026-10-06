# Healthcare Backend — Go Phase 3

Production-oriented foundation for the locked Healthcare App plan.

## Added in Phase 3
- OTP request/verification flow (development OTP returned only when OTP_DEV_MODE=true)
- JWT authentication
- User roles: USER / ADMIN / PARTNER
- Admin user listing and role management
- Digital Healthcare Card
- Basic/Pro plans
- Medical Help
- Healthcare Assistance
- Healthcare Network and Offers
- Razorpay Pro ₹500 order creation
- Razorpay payment signature verification
- Razorpay webhook signature verification + idempotent event storage
- PostgreSQL migrations

## Important
OTP is still provider-agnostic. For production, connect an SMS/OTP provider and set OTP_DEV_MODE=false.
Payment activation should rely on verified server-side payment events/webhooks. Review gateway webhook event handling before production launch.

## Run
1. Configure the environment variables from `.env.example`.
2. Start PostgreSQL with Docker Compose or use Render PostgreSQL.
3. The server automatically applies the embedded migrations on startup.
4. Run `go mod tidy` and `go test ./...` on a machine with Go dependency access.
5. Start with `go run ./cmd/server`.

## Main endpoints
- POST `/api/v1/auth/request-otp`
- POST `/api/v1/auth/verify-otp`
- GET `/api/v1/me`
- GET `/api/v1/plans`
- GET `/api/v1/card`
- GET `/api/v1/partners`
- GET `/api/v1/offers`
- GET `/api/v1/notifications`
- POST `/api/v1/medical-help`
- POST `/api/v1/healthcare-assistance`
- POST `/api/v1/payments/pro/order`
- POST `/api/v1/payments/pro/verify`
- POST `/api/v1/payments/razorpay/webhook`
- GET `/api/v1/admin/users`
- POST `/api/v1/admin/users/{id}/role`


## Phase 5 additions
- Medical Help list/status API for users
- Admin Medical Help status updates with internal notes/events
- FCM device-token registration
- Admin partner listing/status management
- Migration 004 adds device tokens, help events and partner ownership field

Production still requires real OTP/SMS provider, Firebase project credentials, Razorpay production credentials, secret management, TLS, backups and deployment testing.


## Phase 6 additions
- `GET /api/v1/admin/stats` for dashboard counters.
- CORS middleware controlled by `CORS_ORIGINS`.
- Existing admin/partner-related APIs remain available.
- Dashboard is designed to use the same Go API; do not expose admin endpoints without admin JWT.
