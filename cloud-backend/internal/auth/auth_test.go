package auth

import "testing"

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("bay-secret-1")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "bay-secret-1") {
		t.Fatal("expected match")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Fatal("expected reject")
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Admin@Mechazone.Local "); got != "admin@mechazone.local" {
		t.Fatalf("got %s", got)
	}
}
