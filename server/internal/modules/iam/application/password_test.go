package application

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Fatal("expected valid password")
	}
	if VerifyPassword(hash, "incorrect horse battery staple") {
		t.Fatal("expected invalid password")
	}
}

func TestPasswordHashRejectsWeakInput(t *testing.T) {
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("expected short password rejection")
	}
	if VerifyPassword("not-an-argon-hash", "password") {
		t.Fatal("malformed hash must not verify")
	}
}
