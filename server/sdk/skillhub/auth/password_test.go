package auth

import (
	"testing"
)

func TestHashPasswordAndCheckPassword(t *testing.T) {
	password := "MyP@ssw0rd123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if err := CheckPassword(password, hash); err != nil {
		t.Fatalf("CheckPassword with correct password failed: %v", err)
	}
}

func TestCheckPasswordWrongPassword(t *testing.T) {
	hash, err := HashPassword("MyP@ssw0rd123")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	err = CheckPassword("WrongPassword999", hash)
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestDefaultPasswordPolicyTooShort(t *testing.T) {
	policy := DefaultPasswordPolicy()
	err := policy.Validate("abc1234")
	if err == nil {
		t.Fatal("expected error for 7-char password (less than min 8)")
	}
}

func TestDefaultPasswordPolicyTooWeak(t *testing.T) {
	policy := DefaultPasswordPolicy()
	err := policy.Validate("abcdefgh")
	if err == nil {
		t.Fatal("expected error for all-lowercase password (only 1 char type, need 3)")
	}
}

func TestDefaultPasswordPolicyMaxLength(t *testing.T) {
	policy := DefaultPasswordPolicy()
	long := make([]byte, 129)
	for i := range long {
		long[i] = 'a'
	}
	// Add other char types so it passes the char-type check but fails length.
	long[0] = 'A'
	long[1] = '1'
	err := policy.Validate(string(long))
	if err == nil {
		t.Fatal("expected error for password exceeding max length 128")
	}
}

func TestDefaultPasswordPolicyValid(t *testing.T) {
	policy := DefaultPasswordPolicy()
	err := policy.Validate("MyP@ssw0rd")
	if err != nil {
		t.Fatalf("expected valid password (10 chars, 4 types), got error: %v", err)
	}
}

func TestNormalizeUsernameValid(t *testing.T) {
	username, err := NormalizeUsername("TestUser")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if username != "testuser" {
		t.Fatalf("expected 'testuser', got '%s'", username)
	}

	username, err = NormalizeUsername("  My_Name123  ")
	if err != nil {
		t.Fatalf("expected no error for whitespace-padded username, got: %v", err)
	}
	if username != "my_name123" {
		t.Fatalf("expected 'my_name123', got '%s'", username)
	}
}

func TestNormalizeUsernameInvalid(t *testing.T) {
	_, err := NormalizeUsername("ab")
	if err == nil {
		t.Fatal("expected error for username shorter than 3 chars")
	}

	_, err = NormalizeUsername("")
	if err == nil {
		t.Fatal("expected error for empty username")
	}

	_, err = NormalizeUsername("invalid-username")
	if err == nil {
		t.Fatal("expected error for username with dashes")
	}
}

func TestNormalizeEmailValid(t *testing.T) {
	result, err := NormalizeEmail("Test@Example.com")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != "test@example.com" {
		t.Fatalf("expected 'test@example.com', got '%s'", result)
	}
}

func TestNormalizeEmailBlank(t *testing.T) {
	result, err := NormalizeEmail("")
	if err != nil {
		t.Fatalf("expected no error for blank email, got: %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty string for blank email, got '%s'", result)
	}

	result, err = NormalizeEmail("   ")
	if err != nil {
		t.Fatalf("expected no error for whitespace-only email, got: %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty string for whitespace-only email, got '%s'", result)
	}
}

func TestNormalizeEmailInvalid(t *testing.T) {
	// NormalizeEmail only trims and lowercases; it does not validate format.
	result, err := NormalizeEmail("NotAnEmail")
	if err != nil {
		t.Fatalf("expected no error for non-email string, got: %v", err)
	}
	if result != "notanemail" {
		t.Fatalf("expected 'notanemail', got '%s'", result)
	}
}

func TestGenerateSecureCode(t *testing.T) {
	code, err := GenerateSecureCode()
	if err != nil {
		t.Fatalf("GenerateSecureCode failed: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6-character code, got %d characters: '%s'", len(code), code)
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			t.Fatalf("expected all digits, got non-digit character: %c", r)
		}
	}
}
