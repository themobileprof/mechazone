package ledger

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// ShopCustomer is this shop's (or freelancer's) name/phone/plate on a VIN.
// It follows the login, not the laptop. Other shops cannot read it.
type ShopCustomer struct {
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone"`
	Plate       string `json:"plate"`
}

func (s *Store) ShopCustomer(ctx context.Context, vin, shopID, technicianID string) (ShopCustomer, error) {
	var c ShopCustomer
	err := s.pool.QueryRow(ctx, `
		SELECT display_name, phone, plate
		FROM shop_customers
		WHERE vin = $1
		  AND (
		        ($2 <> '' AND shop_id::text = $2)
		        OR ($2 = '' AND $3 <> '' AND shop_id IS NULL AND technician_id::text = $3)
		      )
	`, vin, strings.TrimSpace(shopID), strings.TrimSpace(technicianID)).Scan(&c.DisplayName, &c.Phone, &c.Plate)
	if errors.Is(err, pgx.ErrNoRows) {
		return ShopCustomer{}, nil
	}
	if err != nil {
		return ShopCustomer{}, err
	}
	return c, nil
}

func (s *Store) UpsertShopCustomer(ctx context.Context, vin, shopID, technicianID string, in ShopCustomer) (ShopCustomer, error) {
	in.DisplayName = clipRunes(strings.TrimSpace(in.DisplayName), 200)
	in.Phone = clipRunes(strings.TrimSpace(in.Phone), 40)
	in.Plate = strings.ToUpper(clipRunes(strings.TrimSpace(in.Plate), 16))
	var shop any
	if strings.TrimSpace(shopID) != "" {
		shop = shopID
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE shop_customers
		SET display_name = $4, phone = $5, plate = $6, technician_id = $3::uuid, updated_at = NOW()
		WHERE vin = $1
		  AND (
		        ($2 <> '' AND shop_id::text = $2)
		        OR ($2 = '' AND shop_id IS NULL AND technician_id::text = $3)
		      )
	`, vin, strings.TrimSpace(shopID), technicianID, in.DisplayName, in.Phone, in.Plate)
	if err != nil {
		return ShopCustomer{}, err
	}
	if tag.RowsAffected() == 0 {
		_, err = s.pool.Exec(ctx, `
			INSERT INTO shop_customers (shop_id, technician_id, vin, display_name, phone, plate)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
		`, shop, technicianID, vin, in.DisplayName, in.Phone, in.Plate)
		if err != nil {
			return ShopCustomer{}, err
		}
	}
	return in, nil
}

func clipRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n])
}
