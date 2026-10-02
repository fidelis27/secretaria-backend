package user

import "testing"

func TestNewRejectsInvalidEmail(t *testing.T) {
	if _, err := New("user-1", "Ana", "invalid-email", false); err != ErrEmailInvalid {
		t.Fatalf("error = %v, want %v", err, ErrEmailInvalid)
	}
}
