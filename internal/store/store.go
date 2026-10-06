package store

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"healthcare-backend/internal/models"
	"time"
)

type Store struct{ DB *pgxpool.Pool }

func (s Store) UpsertUser(ctx context.Context, mobile, name, location string) (models.User, error) {
	var u models.User
	err := s.DB.QueryRow(ctx, `INSERT INTO users(id,mobile,name,location,plan,status) VALUES($1,$2,$3,$4,'BASIC','ACTIVE') ON CONFLICT(mobile) DO UPDATE SET name=EXCLUDED.name,location=EXCLUDED.location RETURNING id,mobile,COALESCE(name,''),COALESCE(location,''),plan,status,created_at`, uuid.New(), mobile, name, location).Scan(&u.ID, &u.Mobile, &u.Name, &u.Location, &u.Plan, &u.Status, &u.CreatedAt)
	return u, err
}
func (s Store) GetUser(ctx context.Context, id string) (models.User, error) {
	var u models.User
	e := s.DB.QueryRow(ctx, `SELECT id,mobile,COALESCE(name,''),COALESCE(location,''),plan,status,created_at FROM users WHERE id=$1`, id).Scan(&u.ID, &u.Mobile, &u.Name, &u.Location, &u.Plan, &u.Status, &u.CreatedAt)
	return u, e
}
func (s Store) Partners(ctx context.Context, typ, city string) ([]models.Partner, error) {
	q := `SELECT id,name,type,city,phone,address,COALESCE(benefits,''),status FROM partners WHERE status='ACTIVE'`
	args := []any{}
	n := 1
	if typ != "" {
		q += ` AND type=$` + itoa(n)
		args = append(args, typ)
		n++
	}
	if city != "" {
		q += ` AND city ILIKE $` + itoa(n)
		args = append(args, "%"+city+"%")
		n++
	}
	q += ` ORDER BY name LIMIT 50`
	rows, e := s.DB.Query(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []models.Partner{}
	for rows.Next() {
		var p models.Partner
		if e = rows.Scan(&p.ID, &p.Name, &p.Type, &p.City, &p.Phone, &p.Address, &p.Benefits, &p.Status); e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s Store) CreateHelp(ctx context.Context, userID, details string) (models.MedicalHelpRequest, error) {
	var r models.MedicalHelpRequest
	e := s.DB.QueryRow(ctx, `INSERT INTO medical_help_requests(id,user_id,details,status) VALUES($1,$2,$3,'SUBMITTED') RETURNING id,user_id,details,status,created_at`, uuid.New(), userID, details).Scan(&r.ID, &r.UserID, &r.Details, &r.Status, &r.CreatedAt)
	return r, e
}
func (s Store) Offers(ctx context.Context) ([]models.Offer, error) {
	rows, e := s.DB.Query(ctx, `SELECT id,title,description,location,starts_at,ends_at,status FROM offers_updates WHERE status='PUBLISHED' ORDER BY created_at DESC LIMIT 20`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []models.Offer{}
	for rows.Next() {
		var o models.Offer
		if e = rows.Scan(&o.ID, &o.Title, &o.Description, &o.Location, &o.StartsAt, &o.EndsAt, &o.Status); e != nil {
			return nil, e
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
func itoa(i int) string {
	if i == 1 {
		return "1"
	}
	return "2"
}

var _ = errors.New
var _ = time.Now
var _ = pgx.ErrNoRows

func (s Store) GetPlans(ctx context.Context) ([]models.Plan, error) {
	rows, err := s.DB.Query(ctx, `SELECT code,name,price_paise,status FROM plans WHERE status='ACTIVE' ORDER BY price_paise`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Plan{}
	for rows.Next() {
		var p models.Plan
		if err := rows.Scan(&p.Code, &p.Name, &p.PricePaise, &p.Status); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s Store) GetCard(ctx context.Context, uid string) (models.HealthcareCard, error) {
	var c models.HealthcareCard
	err := s.DB.QueryRow(ctx, `SELECT id,user_id,card_number,status,created_at FROM healthcare_cards WHERE user_id=$1`, uid).Scan(&c.ID, &c.UserID, &c.CardNumber, &c.Status, &c.CreatedAt)
	if err == nil {
		return c, nil
	}
	cardNo := "HC-" + uuid.NewString()[:8]
	err = s.DB.QueryRow(ctx, `INSERT INTO healthcare_cards(id,user_id,card_number,status) VALUES($1,$2,$3,'ACTIVE') ON CONFLICT(user_id) DO UPDATE SET status='ACTIVE' RETURNING id,user_id,card_number,status,created_at`, uuid.New(), uid, cardNo).Scan(&c.ID, &c.UserID, &c.CardNumber, &c.Status, &c.CreatedAt)
	return c, err
}

func (s Store) CreateAssistance(ctx context.Context, uid, requirement string) (models.AssistanceRecord, error) {
	var a models.AssistanceRecord
	err := s.DB.QueryRow(ctx, `INSERT INTO assistance_records(id,user_id,requirement,status) VALUES($1,$2,$3,'OPEN') RETURNING id,user_id,requirement,status,created_at,updated_at`, uuid.New(), uid, requirement).Scan(&a.ID, &a.UserID, &a.Requirement, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (s Store) Notifications(ctx context.Context, uid string) ([]models.Notification, error) {
	rows, err := s.DB.Query(ctx, `SELECT id,title,body,type,read_at,created_at FROM notifications WHERE user_id=$1 OR user_id IS NULL ORDER BY created_at DESC LIMIT 50`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Notification{}
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.Title, &n.Body, &n.Type, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s Store) ListMyHelp(ctx context.Context, uid string) ([]models.MedicalHelpRequest, error) {
	rows, err := s.DB.Query(ctx, `SELECT id,user_id,details,status,created_at FROM medical_help_requests WHERE user_id=$1 ORDER BY created_at DESC LIMIT 50`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.MedicalHelpRequest{}
	for rows.Next() {
		var x models.MedicalHelpRequest
		if err := rows.Scan(&x.ID, &x.UserID, &x.Details, &x.Status, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s Store) RegisterDeviceToken(ctx context.Context, uid, token, platform string) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO device_tokens(id,user_id,token,platform,active,updated_at) VALUES($1,$2,$3,$4,TRUE,now()) ON CONFLICT(token) DO UPDATE SET user_id=EXCLUDED.user_id,platform=EXCLUDED.platform,active=TRUE,updated_at=now()`, uuid.New(), uid, token, platform)
	return err
}
func (s Store) AdminHelp(ctx context.Context, status string) ([]models.MedicalHelpRequest, error) {
	q := `SELECT id,user_id,details,status,created_at FROM medical_help_requests`
	args := []any{}
	if status != "" {
		q += ` WHERE status=$1`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT 100`
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.MedicalHelpRequest{}
	for rows.Next() {
		var x models.MedicalHelpRequest
		if err := rows.Scan(&x.ID, &x.UserID, &x.Details, &x.Status, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s Store) AdminUpdateHelp(ctx context.Context, actor, id, status, note string) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var uid string
	if err = tx.QueryRow(ctx, `UPDATE medical_help_requests SET status=$1 WHERE id=$2 RETURNING user_id`, status, id).Scan(&uid); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO medical_help_events(id,request_id,actor_user_id,status,internal_note) VALUES($1,$2,$3,$4,$5)`, uuid.New(), id, actor, status, note); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO notifications(id,user_id,title,body,type) VALUES($1,$2,$3,$4,'MEDICAL_HELP')`, uuid.New(), uid, "Medical Help update", "Your Medical Help request status is now "+status); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s Store) AdminPartners(ctx context.Context, status string) ([]models.Partner, error) {
	q := `SELECT id,name,type,city,phone,address,COALESCE(benefits,''),status FROM partners`
	args := []any{}
	if status != "" {
		q += ` WHERE status=$1`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT 100`
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Partner{}
	for rows.Next() {
		var x models.Partner
		if err := rows.Scan(&x.ID, &x.Name, &x.Type, &x.City, &x.Phone, &x.Address, &x.Benefits, &x.Status); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s Store) AdminSetPartnerStatus(ctx context.Context, id, status string) error {
	_, err := s.DB.Exec(ctx, `UPDATE partners SET status=$1 WHERE id=$2`, status, id)
	return err
}

func (s Store) ProcessRazorpayWebhook(ctx context.Context, payload map[string]any) error {
	payment, _ := payload["payload"].(map[string]any)
	_ = payment
	// Razorpay webhook payloads vary by event; keep matching conservative.
	p, _ := payload["payload"].(map[string]interface{})
	if p == nil {
		return nil
	}
	payObj, _ := p["payment"].(map[string]interface{})
	ent, _ := payObj["entity"].(map[string]interface{})
	orderID, _ := ent["order_id"].(string)
	paymentID, _ := ent["id"].(string)
	if orderID == "" || paymentID == "" {
		return nil
	}
	internalID, uid, amount, err := s.FindPaymentOrder(ctx, orderID)
	if err != nil || amount != 50000 {
		return err
	}
	if err = s.RecordPayment(ctx, internalID, paymentID, "CAPTURED", amount, payload); err != nil {
		return err
	}
	return s.ActivatePro(ctx, uid)
}
