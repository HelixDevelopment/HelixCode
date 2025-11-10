package security

import (
	"strings"
	"testing"
)

// Simple security tests
func TestInputValidation(t *testing.T) {
	// Test input validation
	maliciousInputs := []string{
		"../../../etc/passwd",
		"<script>alert('xss')</script>",
		"' OR '1'='1",
	}

	for _, input := range maliciousInputs {
		if isSafeInput(input) {
			t.Errorf("Input should be flagged as unsafe: %s", input)
		}
	}
}

func TestPasswordValidation(t *testing.T) {
	// Test password validation
	weakPasswords := []string{
		"123456",
		"password",
		"qwerty",
	}

	for _, password := range weakPasswords {
		if isStrongPassword(password) {
			t.Errorf("Password should be flagged as weak: %s", password)
		}
	}

	strongPassword := "MyStr0ng!P@ssw0rd"
	if !isStrongPassword(strongPassword) {
		t.Errorf("Password should be considered strong: %s", strongPassword)
	}
}

func TestURLValidation(t *testing.T) {
	// Test URL validation
	unsafeURLs := []string{
		"http://127.0.0.1:22",
		"file:///etc/passwd",
		"ftp://malicious.com",
	}

	for _, url := range unsafeURLs {
		if isSecureURL(url) {
			t.Errorf("URL should be flagged as insecure: %s", url)
		}
	}

	safeURL := "https://api.example.com"
	if !isSecureURL(safeURL) {
		t.Errorf("URL should be considered safe: %s", safeURL)
	}
}

// Helper functions
func isSafeInput(input string) bool {
	// Basic input validation
	dangerousPatterns := []string{"../", "<script", "'", "OR"}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(strings.ToLower(input), strings.ToLower(pattern)) {
			return false
		}
	}
	return true
}

func isStrongPassword(password string) bool {
	// Basic password validation
	if len(password) < 8 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}

func isSecureURL(url string) bool {
	// Basic URL validation
	if strings.HasPrefix(url, "http://127.0.0.1") {
		return false
	}
	if strings.HasPrefix(url, "file://") {
		return false
	}
	if strings.HasPrefix(url, "ftp://") {
		return false
	}
	if !strings.HasPrefix(url, "https://") {
		return false
	}
	return true
}
