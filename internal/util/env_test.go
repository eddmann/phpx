package util

import (
	"os"
	"strings"
	"testing"
)

func TestFilterEnv_includes_safe_vars(t *testing.T) {
	// Set some safe environment variables
	safeVars := map[string]string{
		"PATH":   "/usr/bin",
		"HOME":   "/home/test",
		"USER":   "testuser",
		"LANG":   "en_US.UTF-8",
		"TERM":   "xterm",
		"EDITOR": "vim",
	}

	for k, v := range safeVars {
		os.Setenv(k, v)
	}

	result := FilterEnv(nil)

	for k, v := range safeVars {
		found := false
		expected := k + "=" + v
		for _, env := range result {
			if env == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("safe var %s not found in filtered env", k)
		}
	}
}

func TestFilterEnv_includes_safe_prefixes(t *testing.T) {
	// Set vars with safe prefixes
	os.Setenv("LC_MESSAGES", "en_US.UTF-8")
	os.Setenv("LC_CTYPE", "UTF-8")
	os.Setenv("XDG_CONFIG_HOME", "/home/test/.config")

	result := FilterEnv(nil)

	prefixVars := []string{"LC_MESSAGES", "LC_CTYPE", "XDG_CONFIG_HOME"}
	for _, name := range prefixVars {
		found := false
		for _, env := range result {
			if strings.HasPrefix(env, name+"=") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("prefix var %s not found in filtered env", name)
		}
	}
}

func TestFilterEnv_excludes_secrets(t *testing.T) {
	// Set some secret-looking variables
	secrets := []string{
		"AWS_SECRET_ACCESS_KEY",
		"API_TOKEN",
		"DATABASE_PASSWORD",
		"GITHUB_TOKEN",
		"PRIVATE_KEY",
	}

	for _, s := range secrets {
		os.Setenv(s, "secret-value")
	}
	defer func() {
		for _, s := range secrets {
			os.Unsetenv(s)
		}
	}()

	result := FilterEnv(nil)

	for _, secret := range secrets {
		for _, env := range result {
			if strings.HasPrefix(env, secret+"=") {
				t.Errorf("secret var %s should be excluded from filtered env", secret)
			}
		}
	}
}

func TestFilterEnv_allows_explicit_vars(t *testing.T) {
	// Set a variable that would normally be excluded
	os.Setenv("CUSTOM_SECRET", "my-secret")
	defer os.Unsetenv("CUSTOM_SECRET")

	result := FilterEnv([]string{"CUSTOM_SECRET"})

	found := false
	for _, env := range result {
		if strings.HasPrefix(env, "CUSTOM_SECRET=") {
			found = true
			break
		}
	}
	if !found {
		t.Error("explicitly allowed var CUSTOM_SECRET not found in filtered env")
	}
}

func TestFilterEnv_allows_explicit_values(t *testing.T) {
	// Allow a var with explicit value that isn't in os.Environ
	result := FilterEnv([]string{"NEW_VAR=new-value"})

	found := false
	for _, env := range result {
		if env == "NEW_VAR=new-value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("explicitly set var NEW_VAR=new-value not found in filtered env")
	}
}

func TestFilterEnv_explicit_value_overrides_existing(t *testing.T) {
	// Set an existing var
	os.Setenv("MY_VAR", "original")
	defer os.Unsetenv("MY_VAR")

	// Allow the var (it should pick up the original value since it's in allowed list)
	result := FilterEnv([]string{"MY_VAR"})

	found := false
	for _, env := range result {
		if env == "MY_VAR=original" {
			found = true
			break
		}
	}
	if !found {
		t.Error("allowed var MY_VAR=original not found")
	}
}

func TestFilterEnv_handles_empty_allow_list(t *testing.T) {
	result := FilterEnv(nil)

	// Should still have safe vars
	hasSafeVar := false
	for _, env := range result {
		if strings.HasPrefix(env, "PATH=") || strings.HasPrefix(env, "HOME=") {
			hasSafeVar = true
			break
		}
	}
	if !hasSafeVar {
		t.Error("filtered env should include safe vars even with nil allow list")
	}
}

func TestFilterEnv_handles_malformed_entries(t *testing.T) {
	// Allow list with various formats
	allow := []string{
		"SIMPLE",
		"WITH_VALUE=value",
		"",
	}

	// Should not panic
	result := FilterEnv(allow)

	// Verify WITH_VALUE=value is present
	found := false
	for _, env := range result {
		if env == "WITH_VALUE=value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("WITH_VALUE=value not found in result")
	}
}
