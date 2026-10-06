# Production deployment checklist

1. Provision Linux server and PostgreSQL.
2. Create production database/user.
3. Copy `.env.example` to `.env` and replace every placeholder/secret.
4. Build the Go binary on a controlled build machine/server.
5. Run database migrations before starting the service.
6. Install the systemd unit and start the backend.
7. Put Nginx in front of the Go API.
8. Configure HTTPS with a trusted certificate.
9. Set `CORS_ORIGINS` to the exact dashboard origins.
10. Configure Razorpay production credentials and webhook URL.
11. Configure real OTP/SMS provider and keep `OTP_DEV_MODE=false`.
12. Configure Firebase/FCM credentials for notifications.
13. Verify `/health` and database readiness.
14. Test login, Medical Help, partner data, Pro payment and webhook.
15. Configure daily PostgreSQL backups and test a restore.
16. Keep secrets out of Git and ZIP files.
