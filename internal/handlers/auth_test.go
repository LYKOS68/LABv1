package handlers

import "testing"

func TestValidateRegister(t *testing.T) {
	valid := authInput{Email: "mc@example.de", Name: "MC Test", Password: "geheim123"}

	tests := []struct {
		name  string
		in    authInput
		field string
		code  string
	}{
		{"valid input", valid, "", ""},
		{"email without at", authInput{Email: "mc.example.de", Name: valid.Name, Password: valid.Password}, "email", "email_invalid"},
		{"email with display name", authInput{Email: "MC <mc@example.de>", Name: valid.Name, Password: valid.Password}, "email", "email_invalid"},
		{"name too short", authInput{Email: valid.Email, Name: "M", Password: valid.Password}, "name", "name_invalid"},
		{"name with symbol", authInput{Email: valid.Email, Name: "MC<script>", Password: valid.Password}, "name", "name_invalid"},
		{"password too short", authInput{Email: valid.Email, Name: valid.Name, Password: "kurz"}, "password", "password_short"},
		{"password over bcrypt limit", authInput{Email: valid.Email, Name: valid.Name, Password: string(make([]byte, 73))}, "password", "password_long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateRegister(tt.in)
			if tt.field == "" {
				if len(errs) != 0 {
					t.Fatalf("want no errors, got %v", errs)
				}
				return
			}
			if got := errs[tt.field]; got != tt.code {
				t.Fatalf("errs[%q] = %q, want %q (all: %v)", tt.field, got, tt.code, errs)
			}
		})
	}
}
