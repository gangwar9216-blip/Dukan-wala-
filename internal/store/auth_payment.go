package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type OTPRecord struct {
	ID, Mobile, Hash string
	ExpiresAt        time.Time
	Attempts         int
}

func (s Store) SaveOTP(ctx context.Context, mobile, hash string, expires time.Time) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO otp_requests(id,mobile,otp_hash,expires_at) VALUES($1,$2,$3,$4)`, uuid.New(), mobile, hash, expires)
	return err
}
func (s Store) VerifyLatestOTP(ctx context.Context, mobile, hash string) (bool, error) {
	var id string
	var stored string
	var exp time.Time
	var attempts int
	err := s.DB.QueryRow(ctx, `SELECT id,otp_hash,expires_at,attempts FROM otp_requests WHERE mobile=$1 AND consumed_at IS NULL ORDER BY created_at DESC LIMIT 1`, mobile).Scan(&id, &stored, &exp, &attempts)
	if err != nil {
		return false, err
	}
	if attempts >= 5 || time.Now().After(exp) {
		return false, nil
	}
	if stored != hash {
		_, e := s.DB.Exec(ctx, `UPDATE otp_requests SET attempts=attempts+1 WHERE id=$1`, id)
		return false, e
	}
	_, err = s.DB.Exec(ctx, `UPDATE otp_requests SET consumed_at=now() WHERE id=$1`, id)
	return true, err
}
func (s Store) UserRole(ctx context.Context, id string) (string, error) {
	var r string
	err := s.DB.QueryRow(ctx, `SELECT role FROM users WHERE id=$1`, id).Scan(&r)
	return r, err
}
func (s Store) SetRole(ctx context.Context, id, role string) error {
	var savedRole string

	err := s.DB.QueryRow(
		ctx,
		`UPDATE users SET role=$1,updated_at=now() WHERE id=$2 RETURNING role`,
		role,
		id,
	).Scan(&savedRole)

	if err != nil {
		return err
	}

	if savedRole != role {
		return fmt.Errorf("role was not saved")
	}

	return nil
}
func (s Store) ActivatePro(ctx context.Context, uid string) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE users SET plan='PRO',updated_at=now() WHERE id=$1`, uid); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO user_plans(id,user_id,plan_code,status) VALUES($1,$2,'PRO','ACTIVE')`, uuid.New(), uid); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type PaymentOrder struct {
	ID, ProviderOrderID, PlanCode, Status, Currency string
	AmountPaise                                     int64
	CreatedAt                                       time.Time
}

func (s Store) SavePaymentOrder(ctx context.Context, uid, providerOrder, plan string, amount int64) (PaymentOrder, error) {
	var p PaymentOrder
	err := s.DB.QueryRow(ctx, `INSERT INTO payment_orders(id,user_id,provider_order_id,plan_code,amount_paise,currency,status) VALUES($1,$2,$3,$4,$5,'INR','CREATED') RETURNING id,provider_order_id,plan_code,amount_paise,currency,status,created_at`, uuid.New(), uid, providerOrder, plan, amount).Scan(&p.ID, &p.ProviderOrderID, &p.PlanCode, &p.AmountPaise, &p.Currency, &p.Status, &p.CreatedAt)
	return p, err
}
func (s Store) RecordPayment(ctx context.Context, orderID, paymentID, status string, amount int64, raw any) error {
	b, _ := json.Marshal(raw)
	_, err := s.DB.Exec(ctx, `INSERT INTO payment_transactions(id,payment_order_id,provider_payment_id,status,amount_paise,raw_event) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(provider_payment_id) DO NOTHING`, uuid.New(), orderID, paymentID, status, amount, b)
	return err
}
func (s Store) FindPaymentOrder(ctx context.Context, providerOrder string) (string, string, int64, error) {
	var internalID, uid string
	var amt int64
	err := s.DB.QueryRow(ctx, `SELECT id,user_id,amount_paise FROM payment_orders WHERE provider_order_id=$1`, providerOrder).Scan(&internalID, &uid, &amt)
	return internalID, uid, amt, err
}
func (s Store) SaveWebhook(ctx context.Context, provider, eventID string, payload any) (bool, error) {
	b, _ := json.Marshal(payload)
	_, err := s.DB.Exec(ctx, `INSERT INTO webhook_events(id,provider,event_id,payload) VALUES($1,$2,$3,$4) ON CONFLICT(event_id) DO NOTHING`, uuid.New(), provider, eventID, b)
	if err != nil {
		return false, err
	}
	return true, nil
}

var _ = errors.New
var _ = fmt.Sprintf
