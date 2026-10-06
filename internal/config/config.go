package config

import "os"

type Config struct {
	Port, DatabaseURL, JWTSecret, Env                       string
	RazorpayKeyID, RazorpayKeySecret, RazorpayWebhookSecret string
	OTPDevMode                                              bool
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	jwt := os.Getenv("JWT_SECRET")
	if jwt == "" {
		jwt = "change-me"
	}
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	otpDev := os.Getenv("OTP_DEV_MODE") != "false"
	return Config{
		Port: port, DatabaseURL: os.Getenv("DATABASE_URL"), JWTSecret: jwt, Env: env,
		RazorpayKeyID: os.Getenv("RAZORPAY_KEY_ID"), RazorpayKeySecret: os.Getenv("RAZORPAY_KEY_SECRET"),
		RazorpayWebhookSecret: os.Getenv("RAZORPAY_WEBHOOK_SECRET"), OTPDevMode: otpDev,
	}
}
