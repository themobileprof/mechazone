package ledger

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
)

type AccessRequest struct {
	ID            string     `json:"id"`
	ApplicantName string     `json:"applicant_name"`
	ContactEmail  string     `json:"contact_email"`
	ContactPhone  string     `json:"contact_phone"`
	ShopName      string     `json:"shop_name"`
	City          string     `json:"city"`
	Country       string     `json:"country"`
	Kind          string     `json:"kind"`
	Note          string     `json:"note"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	AlreadyQueued bool       `json:"already_queued,omitempty"`
}

type CreateAccessRequestInput struct {
	ApplicantName string `json:"applicant_name"`
	ContactEmail  string `json:"contact_email"`
	ContactPhone  string `json:"contact_phone"`
	ShopName      string `json:"shop_name"`
	City          string `json:"city"`
	Country       string `json:"country"`
	Kind          string `json:"kind"`
	Note          string `json:"note"`
	Website       string `json:"website"` // honeypot; must stay empty
}

func NormalizeAccessRequest(in CreateAccessRequestInput) (CreateAccessRequestInput, error) {
	out := CreateAccessRequestInput{
		ApplicantName: clip(in.ApplicantName, 120),
		ContactEmail:  strings.ToLower(clip(in.ContactEmail, 255)),
		ContactPhone:  clip(in.ContactPhone, 32),
		ShopName:      clip(in.ShopName, 160),
		City:          clip(in.City, 80),
		Country:       clip(in.Country, 80),
		Kind:          strings.ToLower(clip(in.Kind, 32)),
		Note:          clip(in.Note, 1000),
		Website:       strings.TrimSpace(in.Website),
	}
	if out.Country == "" {
		out.Country = "Nigeria"
	}
	if out.Kind == "" {
		out.Kind = "shop"
	}
	if out.Kind != "shop" && out.Kind != "freelancer" {
		return CreateAccessRequestInput{}, fmt.Errorf("kind must be shop or freelancer")
	}
	if out.ApplicantName == "" {
		return CreateAccessRequestInput{}, fmt.Errorf("name is required")
	}
	if out.City == "" {
		return CreateAccessRequestInput{}, fmt.Errorf("city is required")
	}
	if digits(out.ContactPhone) < 10 {
		return CreateAccessRequestInput{}, fmt.Errorf("WhatsApp number is required")
	}
	if _, err := mail.ParseAddress(out.ContactEmail); err != nil {
		return CreateAccessRequestInput{}, fmt.Errorf("a real email is required")
	}
	if out.Kind == "shop" && out.ShopName == "" {
		return CreateAccessRequestInput{}, fmt.Errorf("shop name is required")
	}
	return out, nil
}

func digits(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}

func clip(s string, n int) string {
	s = strings.TrimSpace(strings.Join(strings.Fields(s), " "))
	if len(s) <= n {
		return s
	}
	return strings.TrimRightFunc(s[:n], unicode.IsSpace)
}

func (s *Store) CreateAccessRequest(ctx context.Context, in CreateAccessRequestInput) (AccessRequest, error) {
	norm, err := NormalizeAccessRequest(in)
	if err != nil {
		return AccessRequest{}, err
	}
	if norm.Website != "" {
		return AccessRequest{Status: "pending", AlreadyQueued: true}, nil
	}

	var existing AccessRequest
	err = s.pool.QueryRow(ctx, `
		SELECT id, applicant_name, contact_email, contact_phone, shop_name, city, country,
		       kind, note, status, created_at, reviewed_at
		FROM access_requests
		WHERE contact_email = $1 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1
	`, norm.ContactEmail).Scan(
		&existing.ID, &existing.ApplicantName, &existing.ContactEmail, &existing.ContactPhone,
		&existing.ShopName, &existing.City, &existing.Country, &existing.Kind, &existing.Note,
		&existing.Status, &existing.CreatedAt, &existing.ReviewedAt,
	)
	if err == nil {
		existing.AlreadyQueued = true
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return AccessRequest{}, err
	}

	var row AccessRequest
	err = s.pool.QueryRow(ctx, `
		INSERT INTO access_requests (
			applicant_name, contact_email, contact_phone, shop_name, city, country, kind, note
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, applicant_name, contact_email, contact_phone, shop_name, city, country,
		          kind, note, status, created_at, reviewed_at
	`, norm.ApplicantName, norm.ContactEmail, norm.ContactPhone, norm.ShopName, norm.City, norm.Country, norm.Kind, norm.Note).Scan(
		&row.ID, &row.ApplicantName, &row.ContactEmail, &row.ContactPhone,
		&row.ShopName, &row.City, &row.Country, &row.Kind, &row.Note,
		&row.Status, &row.CreatedAt, &row.ReviewedAt,
	)
	return row, err
}

func (s *Store) ListAccessRequests(ctx context.Context) ([]AccessRequest, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, applicant_name, contact_email, contact_phone, shop_name, city, country,
		       kind, note, status, created_at, reviewed_at
		FROM access_requests
		ORDER BY CASE status WHEN 'pending' THEN 0 ELSE 1 END, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AccessRequest{}
	for rows.Next() {
		var row AccessRequest
		if err := rows.Scan(
			&row.ID, &row.ApplicantName, &row.ContactEmail, &row.ContactPhone,
			&row.ShopName, &row.City, &row.Country, &row.Kind, &row.Note,
			&row.Status, &row.CreatedAt, &row.ReviewedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) SetAccessRequestStatus(ctx context.Context, id, status string) (AccessRequest, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "provisioned" && status != "dismissed" && status != "pending" {
		return AccessRequest{}, fmt.Errorf("status must be pending, provisioned, or dismissed")
	}
	var row AccessRequest
	err := s.pool.QueryRow(ctx, `
		UPDATE access_requests
		SET status = $2, reviewed_at = CASE WHEN $2 = 'pending' THEN NULL ELSE NOW() END
		WHERE id = $1
		RETURNING id, applicant_name, contact_email, contact_phone, shop_name, city, country,
		          kind, note, status, created_at, reviewed_at
	`, id, status).Scan(
		&row.ID, &row.ApplicantName, &row.ContactEmail, &row.ContactPhone,
		&row.ShopName, &row.City, &row.Country, &row.Kind, &row.Note,
		&row.Status, &row.CreatedAt, &row.ReviewedAt,
	)
	if err != nil {
		return AccessRequest{}, fmt.Errorf("access request not found")
	}
	return row, nil
}
