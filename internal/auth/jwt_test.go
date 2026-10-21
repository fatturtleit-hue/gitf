package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("my-secret-password")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == "my-secret-password" {
		t.Fatal("hash should not equal original password")
	}
}

func TestCheckPassword(t *testing.T) {
	hash, _ := HashPassword("correct")
	if !CheckPassword("correct", hash) {
		t.Fatal("CheckPassword should return true for correct password")
	}
	if CheckPassword("wrong", hash) {
		t.Fatal("CheckPassword should return false for wrong password")
	}
}

func TestGenerateTokenAndValidate(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	token, err := GenerateToken(1, "alice", "alice@example.com", secret)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.UserID != 1 {
		t.Errorf("expected UserID=1, got %d", claims.UserID)
	}
	if claims.Username != "alice" {
		t.Errorf("expected Username=alice, got %s", claims.Username)
	}
	if claims.Email != "alice@example.com" {
		t.Errorf("expected Email=alice@example.com, got %s", claims.Email)
	}
}

func TestValidateTokenWithWrongSecret(t *testing.T) {
	secret := "correct-secret-key-for-testing!!"
	token, _ := GenerateToken(1, "alice", "alice@example.com", secret)
	_, err := ValidateToken(token, "wrong-secret-key-for-testing!")
	if err == nil {
		t.Fatal("expected error when validating with wrong secret")
	}
}

func TestValidateExpiredToken(t *testing.T) {
	secret := "test-secret-for-expiry-testing!"
	claims := Claims{
		UserID:   1,
		Username: "alice",
		Email:    "alice@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to create expired token: %v", err)
	}

	_, err = ValidateToken(token, secret)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateTokenInvalidFormat(t *testing.T) {
	_, err := ValidateToken("not-a-valid-jwt", "secret-key")
	if err == nil {
		t.Fatal("expected error for invalid token format")
	}
}

func TestConstantTimeDelay(t *testing.T) {
	start := time.Now()
	ConstantTimeDelay()
	elapsed := time.Since(start)
	if elapsed < 50*time.Millisecond {
		t.Errorf("delay too short: %v (expected ≥ 50ms)", elapsed)
	}
	if elapsed > 250*time.Millisecond {
		t.Errorf("delay too long: %v (expected ≤ 250ms)", elapsed)
	}
}
