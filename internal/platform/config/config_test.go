package config

import "testing"

func TestLoadAuthRequireVerifiedEmailDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_REQUIRE_VERIFIED_EMAIL", "")

	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !config.AuthRequireVerifiedEmail {
		t.Fatal("email verification must be required by default in production")
	}

	t.Setenv("APP_ENV", "development")
	config, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.AuthRequireVerifiedEmail {
		t.Fatal("email verification should default to optional outside production")
	}
}

func TestLoadAuthRequireVerifiedEmailCanBeConfigured(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_REQUIRE_VERIFIED_EMAIL", "false")

	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.AuthRequireVerifiedEmail {
		t.Fatal("expected AUTH_REQUIRE_VERIFIED_EMAIL=false to disable the missing-claim requirement")
	}

	t.Setenv("AUTH_REQUIRE_VERIFIED_EMAIL", "not-a-bool")
	if _, err := Load(); err == nil {
		t.Fatal("invalid AUTH_REQUIRE_VERIFIED_EMAIL value should be rejected")
	}
}

func TestLoadRejectsInvalidRateLimit(t *testing.T) {
	t.Setenv("RATE_LIMIT_PER_MINUTE", "0")
	if _, err := Load(); err == nil {
		t.Fatal("zero RATE_LIMIT_PER_MINUTE should be rejected")
	}
}
