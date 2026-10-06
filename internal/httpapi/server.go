package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"healthcare-backend/internal/auth"
	"healthcare-backend/internal/models"
	"healthcare-backend/internal/store"
)

type Server struct {
	Store                                                   store.Store
	Auth                                                    auth.Manager
	RazorpayKeyID, RazorpayKeySecret, RazorpayWebhookSecret string
	OTPDevMode                                              bool
}

func (s Server) Router() http.Handler {
	r := chi.NewRouter()
    r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, corsMiddleware, securityHeaders, func(next http.Handler) http.Handler {
    return bodyLimit(1<<20, next)
})
	generalLimiter := newIPLimiter(120, time.Minute)
	authLimiter := newIPLimiter(10, time.Minute)
	r.Use(func(next http.Handler) http.Handler { return routeRateLimit(generalLimiter, next) })
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	r.Route("/api/v1", func(r chi.Router) {
		r.With(func(next http.Handler) http.Handler { return routeRateLimit(authLimiter, next) }).Post("/auth/request-otp", s.requestOTP)
		r.With(func(next http.Handler) http.Handler { return routeRateLimit(authLimiter, next) }).Post("/auth/verify-otp", s.verifyOTP)
		r.Post("/payments/razorpay/webhook", s.razorpayWebhook)
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/me", s.me)
			r.Get("/partners", s.partners)
			r.Get("/offers", s.offers)
			r.Get("/plans", s.plans)
			r.Get("/card", s.card)
			r.Get("/notifications", s.notifications)
			r.Post("/medical-help", s.createHelp)
			r.Post("/healthcare-assistance", s.createAssistance)
			r.Get("/medical-help", s.listMyHelp)
			r.Post("/devices/fcm-token", s.registerFCMToken)
			r.Post("/payments/pro/order", s.createProOrder)
			r.Post("/payments/pro/verify", s.verifyProPayment)
		})
		r.Group(func(r chi.Router) {
			r.Use(s.requireAdmin)
			r.Get("/admin/stats", s.adminStats)
			r.Get("/admin/users", s.adminUsers)
			r.Post("/admin/users/{id}/role", s.adminRole)
			r.Get("/admin/medical-help", s.adminHelp)
			r.Patch("/admin/medical-help/{id}", s.adminUpdateHelp)
			r.Get("/admin/partners", s.adminPartners)
			r.Patch("/admin/partners/{id}/status", s.adminPartnerStatus)
			r.Post("/admin/partners", s.adminCreatePartner)
			r.Post("/admin/offers", s.adminCreateOffer)
			r.Patch("/admin/offers/{id}", s.adminUpdateOffer)
			r.Get("/admin/payments", s.adminPayments)
		})
	})
	return r
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type ctxKey string

const userKey ctxKey = "userID"

func userID(r *http.Request) string {
	v := r.Context().Value(userKey)
	if x, ok := v.(string); ok {
		return x
	}
	return ""
}
func (s Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			write(w, 401, map[string]string{"error": "missing bearer token"})
			return
		}
		id, e := s.Auth.Parse(strings.TrimPrefix(h, "Bearer "))
		if e != nil {
			write(w, 401, map[string]string{"error": "invalid token"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, id)))
	})
}
func (s Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := userID(r)
		role, e := s.Store.UserRole(r.Context(), id)
		if e != nil || role != "ADMIN" {
			write(w, 403, map[string]string{"error": "admin access required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s Server) me(w http.ResponseWriter, r *http.Request) {
	u, e := s.Store.GetUser(r.Context(), userID(r))
	if e != nil {
		write(w, 404, map[string]string{"error": "user not found"})
		return
	}
	write(w, 200, u)
}
func (s Server) partners(w http.ResponseWriter, r *http.Request) {
	out, e := s.Store.Partners(r.Context(), r.URL.Query().Get("type"), r.URL.Query().Get("city"))
	if e != nil {
		write(w, 500, map[string]string{"error": "could not load partners"})
		return
	}
	write(w, 200, out)
}
func (s Server) offers(w http.ResponseWriter, r *http.Request) {
	out, e := s.Store.Offers(r.Context())
	if e != nil {
		write(w, 500, map[string]string{"error": "could not load offers"})
		return
	}
	write(w, 200, out)
}
func (s Server) plans(w http.ResponseWriter, r *http.Request) {
	out, e := s.Store.GetPlans(r.Context())
	if e != nil {
		write(w, 500, map[string]string{"error": "could not load plans"})
		return
	}
	write(w, 200, out)
}
func (s Server) card(w http.ResponseWriter, r *http.Request) {
	out, e := s.Store.GetCard(r.Context(), userID(r))
	if e != nil {
		write(w, 500, map[string]string{"error": "could not load healthcare card"})
		return
	}
	write(w, 200, out)
}
func (s Server) notifications(w http.ResponseWriter, r *http.Request) {
	out, e := s.Store.Notifications(r.Context(), userID(r))
	if e != nil {
		write(w, 500, map[string]string{"error": "could not load notifications"})
		return
	}
	write(w, 200, out)
}
func (s Server) createHelp(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Details string `json:"details"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Details) == "" {
		write(w, 400, map[string]string{"error": "details are required"})
		return
	}
	out, e := s.Store.CreateHelp(r.Context(), userID(r), strings.TrimSpace(in.Details))
	if e != nil {
		write(w, 500, map[string]string{"error": "could not create request"})
		return
	}
	write(w, 201, out)
}
func (s Server) createAssistance(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Requirement string `json:"requirement"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Requirement) == "" {
		write(w, 400, map[string]string{"error": "requirement is required"})
		return
	}
	out, e := s.Store.CreateAssistance(r.Context(), userID(r), strings.TrimSpace(in.Requirement))
	if e != nil {
		write(w, 500, map[string]string{"error": "could not create assistance record"})
		return
	}
	write(w, 201, out)
}
func (s Server) requestOTP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Mobile string `json:"mobile"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Mobile) == "" {
		write(w, 400, map[string]string{"error": "mobile is required"})
		return
	}
	otp, e := auth.GenerateOTP()
	if e != nil {
		write(w, 500, map[string]string{"error": "could not generate otp"})
		return
	}
	if e = s.Store.SaveOTP(r.Context(), strings.TrimSpace(in.Mobile), auth.HashOTP(otp), time.Now().Add(5*time.Minute)); e != nil {
		write(w, 500, map[string]string{"error": "could not save otp"})
		return
	}
	resp := map[string]any{"message": "OTP sent"}
	if s.OTPDevMode {
		resp["dev_otp"] = otp
	}
	write(w, 200, resp)
}
func (s Server) verifyOTP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Mobile, OTP, Name, Location string `json:"mobile"`
	}
	_ = in
	var body map[string]string
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		write(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	mobile := strings.TrimSpace(body["mobile"])
	otp := strings.TrimSpace(body["otp"])
	if mobile == "" || otp == "" {
		write(w, 400, map[string]string{"error": "mobile and otp are required"})
		return
	}
	ok, e := s.Store.VerifyLatestOTP(r.Context(), mobile, auth.HashOTP(otp))
	if e != nil || !ok {
		write(w, 401, map[string]string{"error": "invalid or expired otp"})
		return
	}
	u, e := s.Store.UpsertUser(r.Context(), mobile, body["name"], body["location"])
	if e != nil {
		write(w, 500, map[string]string{"error": "could not save user"})
		return
	}
	adminMobile := strings.TrimSpace(os.Getenv("ADMIN_MOBILE"))

cleanMobile := strings.TrimSpace(mobile)
cleanAdminMobile := strings.TrimSpace(adminMobile)

cleanMobile = strings.TrimPrefix(cleanMobile, "+91")
cleanAdminMobile = strings.TrimPrefix(cleanAdminMobile, "+91")

cleanMobile = strings.ReplaceAll(cleanMobile, " ", "")
cleanAdminMobile = strings.ReplaceAll(cleanAdminMobile, " ", "")

if cleanAdminMobile != "" && cleanMobile == cleanAdminMobile {
    if e := s.Store.SetRole(r.Context(), u.ID, "ADMIN"); e != nil {
        write(w, 500, map[string]string{"error": "could not assign admin role"})
        return
    }
}
	tok, e := s.Auth.Issue(u.ID)
	if e != nil {
		write(w, 500, map[string]string{"error": "could not create token"})
		return
	}
	write(w, 200, map[string]any{"token": tok, "user": u})
}
func (s Server) createProOrder(w http.ResponseWriter, r *http.Request) {
	if s.RazorpayKeyID == "" || s.RazorpayKeySecret == "" {
		write(w, 503, map[string]string{"error": "payment gateway is not configured"})
		return
	}
	req, _ := http.NewRequestWithContext(r.Context(), "POST", "https://api.razorpay.com/v1/orders", strings.NewReader(`{"amount":50000,"currency":"INR","receipt":"pro-`+userID(r)+`"}`))
	req.SetBasicAuth(s.RazorpayKeyID, s.RazorpayKeySecret)
	req.Header.Set("Content-Type", "application/json")
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		write(w, 502, map[string]string{"error": "payment gateway unavailable"})
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		write(w, 502, map[string]string{"error": "could not create payment order"})
		return
	}
	var v struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(b, &v) != nil || v.ID == "" {
		write(w, 502, map[string]string{"error": "invalid gateway response"})
		return
	}
	p, e := s.Store.SavePaymentOrder(r.Context(), userID(r), v.ID, "PRO", 50000)
	if e != nil {
		write(w, 500, map[string]string{"error": "could not save payment order"})
		return
	}
	write(w, 201, map[string]any{"key_id": s.RazorpayKeyID, "order": p})
}
func verifyHMAC(secret, payload, signature string) bool {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	return hmac.Equal([]byte(hex.EncodeToString(h.Sum(nil))), []byte(signature))
}
func (s Server) verifyProPayment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OrderID, PaymentID, Signature string `json:"order_id"`
	}
	_ = in
	var b map[string]string
	if json.NewDecoder(r.Body).Decode(&b) != nil {
		write(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	orderID, paymentID, sig := b["order_id"], b["payment_id"], b["signature"]
	if orderID == "" || paymentID == "" || sig == "" || s.RazorpayKeySecret == "" {
		write(w, 400, map[string]string{"error": "payment fields are required"})
		return
	}
	if !verifyHMAC(s.RazorpayKeySecret, orderID+"|"+paymentID, sig) {
		write(w, 400, map[string]string{"error": "invalid payment signature"})
		return
	}
	internalOrderID, uid, amt, e := s.Store.FindPaymentOrder(r.Context(), orderID)
	if e != nil || uid != userID(r) || amt != 50000 {
		write(w, 400, map[string]string{"error": "invalid payment order"})
		return
	}
	if e = s.Store.RecordPayment(r.Context(), internalOrderID, paymentID, "CAPTURED", amt, b); e != nil {
		write(w, 500, map[string]string{"error": "could not record payment"})
		return
	}
	if e = s.Store.ActivatePro(r.Context(), uid); e != nil {
		write(w, 500, map[string]string{"error": "payment recorded but plan activation failed"})
		return
	}
	write(w, 200, map[string]string{"status": "PRO_ACTIVE"})
}
func (s Server) razorpayWebhook(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	sig := r.Header.Get("X-Razorpay-Signature")
	if s.RazorpayWebhookSecret == "" || !verifyHMAC(s.RazorpayWebhookSecret, string(body), sig) {
		write(w, 401, map[string]string{"error": "invalid webhook signature"})
		return
	}
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		write(w, 400, map[string]string{"error": "invalid webhook"})
		return
	}
	eventID := r.Header.Get("X-Razorpay-Event-Id")
	if eventID == "" {
		eventID = fmt.Sprintf("%x", sha256.Sum256(body))
	}
	newEvent, e := s.Store.SaveWebhook(r.Context(), "RAZORPAY", eventID, payload)
	if e != nil {
		write(w, 500, map[string]string{"error": "could not save webhook"})
		return
	}
	if !newEvent {
		write(w, 200, map[string]string{"status": "already_processed"})
		return
	}
	// The webhook is authoritative for production activation. Never activate Pro
	// from an Android-only success callback. Handle captured payment events.
	event, _ := payload["event"].(string)
	if event == "payment.captured" || event == "order.paid" {
		// Payment/order identifiers are extracted by the store helper. If the
		// payload cannot be matched, it remains recorded for admin reconciliation.
		_ = s.Store.ProcessRazorpayWebhook(r.Context(), payload)
	}
	write(w, 200, map[string]string{"status": "received"})
}

func (s Server) adminCreatePartner(w http.ResponseWriter, r *http.Request) {
	var in struct{ Name, Type, City, Phone, Address, Benefits string }
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Type) == "" {
		write(w, 400, map[string]string{"error": "name and type are required"})
		return
	}
	var id string
	err := s.Store.DB.QueryRow(r.Context(), `INSERT INTO partners(id,name,type,city,phone,address,benefits,status) VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,'PENDING') RETURNING id`, in.Name, in.Type, in.City, in.Phone, in.Address, in.Benefits).Scan(&id)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not create partner"})
		return
	}
	write(w, 201, map[string]string{"id": id, "status": "PENDING"})
}
func (s Server) adminCreateOffer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title, Description, Location string
		StartsAt, EndsAt             *time.Time
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Description) == "" {
		write(w, 400, map[string]string{"error": "title and description are required"})
		return
	}
	var id string
	err := s.Store.DB.QueryRow(r.Context(), `INSERT INTO offers_updates(id,title,description,location,starts_at,ends_at,status) VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,'DRAFT') RETURNING id`, in.Title, in.Description, in.Location, in.StartsAt, in.EndsAt).Scan(&id)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not create update"})
		return
	}
	write(w, 201, map[string]string{"id": id, "status": "DRAFT"})
}
func (s Server) adminUpdateOffer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title, Description, Location, Status string
		StartsAt, EndsAt                     *time.Time
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		write(w, 400, map[string]string{"error": "invalid body"})
		return
	}
	allowed := map[string]bool{"DRAFT": true, "PUBLISHED": true, "ARCHIVED": true}
	if in.Status != "" && !allowed[in.Status] {
		write(w, 400, map[string]string{"error": "invalid status"})
		return
	}
	_, err := s.Store.DB.Exec(r.Context(), `UPDATE offers_updates SET title=COALESCE(NULLIF($1,''),title),description=COALESCE(NULLIF($2,''),description),location=COALESCE(NULLIF($3,''),location),starts_at=COALESCE($4,starts_at),ends_at=COALESCE($5,ends_at),status=COALESCE(NULLIF($6,''),status) WHERE id=$7`, in.Title, in.Description, in.Location, in.StartsAt, in.EndsAt, in.Status, chi.URLParam(r, "id"))
	if err != nil {
		write(w, 500, map[string]string{"error": "could not update update"})
		return
	}
	write(w, 200, map[string]string{"status": "updated"})
}
func (s Server) adminPayments(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.DB.Query(r.Context(), `SELECT po.id,po.user_id,po.provider_order_id,po.plan_code,po.amount_paise,po.currency,po.status,po.created_at,COALESCE(pt.status,'') FROM payment_orders po LEFT JOIN LATERAL (SELECT status FROM payment_transactions WHERE payment_order_id=po.id ORDER BY created_at DESC LIMIT 1) pt ON TRUE ORDER BY po.created_at DESC LIMIT 100`)
	if err != nil {
		write(w, 500, map[string]string{"error": "could not load payments"})
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, uid, porder, plan, currency, status, tx string
		var amount int64
		var created time.Time
		if err = rows.Scan(&id, &uid, &porder, &plan, &amount, &currency, &status, &created, &tx); err != nil {
			write(w, 500, map[string]string{"error": "could not read payments"})
			return
		}
		out = append(out, map[string]any{"id": id, "user_id": uid, "provider_order_id": porder, "plan": plan, "amount_paise": amount, "currency": currency, "order_status": status, "transaction_status": tx, "created_at": created})
	}
	write(w, 200, out)
}

func (s Server) adminUsers(w http.ResponseWriter, r *http.Request) {
	rows, e := s.Store.DB.Query(r.Context(), `SELECT id,mobile,COALESCE(name,''),COALESCE(location,''),plan,status,role,created_at FROM users ORDER BY created_at DESC LIMIT 100`)
	if e != nil {
		write(w, 500, map[string]string{"error": "could not load users"})
		return
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, mobile, name, loc, plan, status, role string
		var created time.Time
		if e = rows.Scan(&id, &mobile, &name, &loc, &plan, &status, &role, &created); e != nil {
			write(w, 500, map[string]string{"error": "could not read users"})
			return
		}
		out = append(out, map[string]any{"id": id, "mobile": mobile, "name": name, "location": loc, "plan": plan, "status": status, "role": role, "created_at": created})
	}
	write(w, 200, out)
}
func (s Server) adminRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Role string `json:"role"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || !map[string]bool{"USER": true, "ADMIN": true, "PARTNER": true}[in.Role] {
		write(w, 400, map[string]string{"error": "invalid role"})
		return
	}
	if e := s.Store.SetRole(r.Context(), id, in.Role); e != nil {
		write(w, 500, map[string]string{"error": "could not update role"})
		return
	}
	write(w, 200, map[string]string{"status": "updated"})
}

var _ models.User

func (s Server) listMyHelp(w http.ResponseWriter, r *http.Request) {
	out, e := s.Store.ListMyHelp(r.Context(), userID(r))
	if e != nil {
		write(w, 500, map[string]string{"error": "could not load requests"})
		return
	}
	write(w, 200, out)
}
func (s Server) registerFCMToken(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Token) == "" {
		write(w, 400, map[string]string{"error": "token is required"})
		return
	}
	if in.Platform == "" {
		in.Platform = "ANDROID"
	}
	if e := s.Store.RegisterDeviceToken(r.Context(), userID(r), strings.TrimSpace(in.Token), in.Platform); e != nil {
		write(w, 500, map[string]string{"error": "could not register device token"})
		return
	}
	write(w, 200, map[string]string{"status": "registered"})
}
func (s Server) adminHelp(w http.ResponseWriter, r *http.Request) {
	out, e := s.Store.AdminHelp(r.Context(), r.URL.Query().Get("status"))
	if e != nil {
		write(w, 500, map[string]string{"error": "could not load medical help"})
		return
	}
	write(w, 200, out)
}
func (s Server) adminUpdateHelp(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status       string `json:"status"`
		InternalNote string `json:"internal_note"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Status) == "" {
		write(w, 400, map[string]string{"error": "status is required"})
		return
	}
	allowed := map[string]bool{"SUBMITTED": true, "UNDER_REVIEW": true, "CALLBACK": true, "COMPLETED": true, "CANCELLED": true}
	if !allowed[in.Status] {
		write(w, 400, map[string]string{"error": "invalid status"})
		return
	}
	if e := s.Store.AdminUpdateHelp(r.Context(), userID(r), chi.URLParam(r, "id"), in.Status, in.InternalNote); e != nil {
		write(w, 500, map[string]string{"error": "could not update request"})
		return
	}
	write(w, 200, map[string]string{"status": "updated"})
}
func (s Server) adminPartners(w http.ResponseWriter, r *http.Request) {
	out, e := s.Store.AdminPartners(r.Context(), r.URL.Query().Get("status"))
	if e != nil {
		write(w, 500, map[string]string{"error": "could not load partners"})
		return
	}
	write(w, 200, out)
}
func (s Server) adminPartnerStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Status) == "" {
		write(w, 400, map[string]string{"error": "status is required"})
		return
	}
	if e := s.Store.AdminSetPartnerStatus(r.Context(), chi.URLParam(r, "id"), in.Status); e != nil {
		write(w, 500, map[string]string{"error": "could not update partner"})
		return
	}
	write(w, 200, map[string]string{"status": "updated"})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := os.Getenv("CORS_ORIGINS")
		if allowed == "" {
			allowed = "*"
		}
		if allowed == "*" || origin != "" && containsCSV(allowed, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if allowed == "*" && origin == "" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func containsCSV(csv, value string) bool {
	for _, x := range strings.Split(csv, ",") {
		if strings.TrimSpace(x) == value {
			return true
		}
	}
	return false
}

func (s Server) adminStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var users, pro, help, partners, pendingPartners, published int
	if err := s.Store.DB.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		write(w, 500, map[string]string{"error": "could not load stats"})
		return
	}
	if err := s.Store.DB.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE plan='PRO'`).Scan(&pro); err != nil {
		write(w, 500, map[string]string{"error": "could not load stats"})
		return
	}
	if err := s.Store.DB.QueryRow(ctx, `SELECT COUNT(*) FROM medical_help_requests`).Scan(&help); err != nil {
		write(w, 500, map[string]string{"error": "could not load stats"})
		return
	}
	if err := s.Store.DB.QueryRow(ctx, `SELECT COUNT(*) FROM partners`).Scan(&partners); err != nil {
		write(w, 500, map[string]string{"error": "could not load stats"})
		return
	}
	if err := s.Store.DB.QueryRow(ctx, `SELECT COUNT(*) FROM partners WHERE status='PENDING'`).Scan(&pendingPartners); err != nil {
		write(w, 500, map[string]string{"error": "could not load stats"})
		return
	}
	if err := s.Store.DB.QueryRow(ctx, `SELECT COUNT(*) FROM offers_updates WHERE status='PUBLISHED'`).Scan(&published); err != nil {
		write(w, 500, map[string]string{"error": "could not load stats"})
		return
	}
	write(w, 200, map[string]int{"users": users, "pro_users": pro, "medical_help": help, "partners": partners, "pending_partners": pendingPartners, "published_updates": published})
}
