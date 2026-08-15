package ledger

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"mechazone/cloud-backend/internal/auth"
)

type Shop struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Country   string    `json:"location_country"`
	City      string    `json:"location_city"`
	CreatedAt time.Time `json:"created_at"`
}

type Technician struct {
	ID         string    `json:"id"`
	ShopID     string    `json:"shop_id,omitempty"`
	ShopName   string    `json:"shop_name,omitempty"`
	FullName   string    `json:"full_name"`
	Email      string    `json:"email"`
	Reputation int       `json:"reputation_score"`
	Freelancer bool      `json:"freelancer"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateShopInput struct {
	Name    string `json:"name"`
	Country string `json:"location_country"`
	City    string `json:"location_city"`
}

type CreateTechnicianInput struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	ShopID   string `json:"shop_id"`
}

func (s *Store) EnsureSuperAdmin(ctx context.Context, email, password string) error {
	email = auth.NormalizeEmail(email)
	if email == "" || password == "" {
		return fmt.Errorf("SUPERADMIN_EMAIL and SUPERADMIN_PASSWORD are required")
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role = 'super_admin')`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, 'super_admin')
	`, email, hash)
	return err
}

func (s *Store) Authenticate(ctx context.Context, email, password string) (auth.Principal, error) {
	email = auth.NormalizeEmail(email)
	var (
		userID, hash, role string
		techID, techName   *string
		shopID, shopName   *string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.password_hash, u.role,
		       t.id, t.full_name, t.shop_id, sh.name
		FROM users u
		LEFT JOIN technicians t ON t.id = u.technician_id
		LEFT JOIN shops sh ON sh.id = t.shop_id
		WHERE u.email = $1
	`, email).Scan(&userID, &hash, &role, &techID, &techName, &shopID, &shopName)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Principal{}, fmt.Errorf("invalid email or password")
	}
	if err != nil {
		return auth.Principal{}, err
	}
	if !auth.CheckPassword(hash, password) {
		return auth.Principal{}, fmt.Errorf("invalid email or password")
	}
	return principalFromRow(userID, email, role, techID, techName, shopID, shopName), nil
}

func (s *Store) CreateSession(ctx context.Context, userID string) (raw string, err error) {
	raw, hash, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO auth_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, hash, time.Now().Add(auth.SessionTTL))
	return raw, err
}

func (s *Store) PrincipalByToken(ctx context.Context, raw string) (auth.Principal, error) {
	if raw == "" {
		return auth.Principal{}, fmt.Errorf("not authenticated")
	}
	var (
		userID, email, role string
		techID, techName    *string
		shopID, shopName    *string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.role, t.id, t.full_name, t.shop_id, sh.name
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		LEFT JOIN technicians t ON t.id = u.technician_id
		LEFT JOIN shops sh ON sh.id = t.shop_id
		WHERE s.token_hash = $1 AND s.expires_at > NOW()
	`, auth.HashToken(raw)).Scan(&userID, &email, &role, &techID, &techName, &shopID, &shopName)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Principal{}, fmt.Errorf("not authenticated")
	}
	if err != nil {
		return auth.Principal{}, err
	}
	return principalFromRow(userID, email, role, techID, techName, shopID, shopName), nil
}

func (s *Store) RevokeToken(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM auth_sessions WHERE token_hash = $1`, auth.HashToken(raw))
	return err
}

func (s *Store) CreateShop(ctx context.Context, in CreateShopInput) (Shop, error) {
	name := strings.TrimSpace(in.Name)
	city := strings.TrimSpace(in.City)
	country := strings.TrimSpace(in.Country)
	if name == "" || city == "" {
		return Shop{}, fmt.Errorf("shop name and city are required")
	}
	if country == "" {
		country = "Nigeria"
	}
	var shop Shop
	err := s.pool.QueryRow(ctx, `
		INSERT INTO shops (name, location_country, location_city)
		VALUES ($1, $2, $3)
		RETURNING id, name, location_country, location_city, created_at
	`, name, country, city).Scan(&shop.ID, &shop.Name, &shop.Country, &shop.City, &shop.CreatedAt)
	return shop, err
}

func (s *Store) ListShops(ctx context.Context) ([]Shop, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, location_country, location_city, created_at
		FROM shops ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Shop{}
	for rows.Next() {
		var shop Shop
		if err := rows.Scan(&shop.ID, &shop.Name, &shop.Country, &shop.City, &shop.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, shop)
	}
	return out, rows.Err()
}

func (s *Store) CreateTechnician(ctx context.Context, in CreateTechnicianInput) (Technician, error) {
	name := strings.TrimSpace(in.FullName)
	email := auth.NormalizeEmail(in.Email)
	if name == "" || email == "" {
		return Technician{}, fmt.Errorf("full_name and email are required")
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return Technician{}, err
	}
	var shop any
	if strings.TrimSpace(in.ShopID) != "" {
		shop = in.ShopID
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Technician{}, err
	}
	defer tx.Rollback(ctx)

	var tech Technician
	err = tx.QueryRow(ctx, `
		INSERT INTO technicians (shop_id, full_name)
		VALUES ($1, $2)
		RETURNING id, COALESCE(shop_id::text, ''), full_name, reputation_score, created_at
	`, shop, name).Scan(&tech.ID, &tech.ShopID, &tech.FullName, &tech.Reputation, &tech.CreatedAt)
	if err != nil {
		return Technician{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (email, password_hash, role, technician_id)
		VALUES ($1, $2, 'technician', $3)
	`, email, hash, tech.ID); err != nil {
		if strings.Contains(err.Error(), "users_email") || strings.Contains(err.Error(), "duplicate") {
			return Technician{}, fmt.Errorf("email already in use")
		}
		return Technician{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Technician{}, err
	}
	tech.Email = email
	tech.Freelancer = tech.ShopID == ""
	if tech.ShopID != "" {
		_ = s.pool.QueryRow(ctx, `SELECT name FROM shops WHERE id = $1`, tech.ShopID).Scan(&tech.ShopName)
	}
	return tech, nil
}

func (s *Store) ListTechnicians(ctx context.Context) ([]Technician, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, COALESCE(t.shop_id::text, ''), COALESCE(sh.name, ''), t.full_name,
		       u.email, t.reputation_score, t.created_at
		FROM technicians t
		JOIN users u ON u.technician_id = t.id
		LEFT JOIN shops sh ON sh.id = t.shop_id
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Technician{}
	for rows.Next() {
		var tech Technician
		if err := rows.Scan(&tech.ID, &tech.ShopID, &tech.ShopName, &tech.FullName, &tech.Email, &tech.Reputation, &tech.CreatedAt); err != nil {
			return nil, err
		}
		tech.Freelancer = tech.ShopID == ""
		out = append(out, tech)
	}
	return out, rows.Err()
}

func principalFromRow(userID, email, role string, techID, techName, shopID, shopName *string) auth.Principal {
	p := auth.Principal{UserID: userID, Email: email, Role: role}
	if techID != nil {
		p.TechnicianID = *techID
	}
	if techName != nil {
		p.TechnicianName = *techName
	}
	if shopID != nil {
		p.ShopID = *shopID
	}
	if shopName != nil {
		p.ShopName = *shopName
	}
	p.Freelancer = p.Role == "technician" && p.ShopID == ""
	return p
}
