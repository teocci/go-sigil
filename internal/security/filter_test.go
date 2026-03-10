package security

import (
	"os"
	"path/filepath"
	"testing"
)

// ---- SecurityFilter ---------------------------------------------------------

func TestSecurityFilter_ClassifyFile_Normal(t *testing.T) {
	f, err := NewFilter(nil, nil, nil)
	if err != nil {
		t.Fatalf("NewFilter: %v", err)
	}
	for _, path := range []string{
		"main.go",
		"src/util.go",
		"internal/config/config.go",
		"README.md",
	} {
		tier := f.ClassifyFile(path)
		if tier != TierNormal {
			t.Errorf("ClassifyFile(%q) = %v, want TierNormal", path, tier)
		}
	}
}

func TestSecurityFilter_ClassifyFile_Redacted(t *testing.T) {
	f, err := NewFilter(nil, nil, nil)
	if err != nil {
		t.Fatalf("NewFilter: %v", err)
	}
	redactedPaths := []string{
		".env",
		"config.env.local",
		"server.pem",
		"id_rsa",
		"id_rsa.pub",
		"credentials.json",
		"server.key",
		"cert.p12",
		".netrc",
	}
	for _, path := range redactedPaths {
		tier := f.ClassifyFile(path)
		if tier != TierRedacted {
			t.Errorf("ClassifyFile(%q) = %v, want TierRedacted", path, tier)
		}
	}
}

func TestSecurityFilter_ClassifyFile_Ignored(t *testing.T) {
	f, err := NewFilter([]string{"secrets.yaml", "*.private"}, nil, nil)
	if err != nil {
		t.Fatalf("NewFilter: %v", err)
	}

	if f.ClassifyFile("secrets.yaml") != TierIgnored {
		t.Error("secrets.yaml should be TierIgnored")
	}
	if f.ClassifyFile("data.private") != TierIgnored {
		t.Error("data.private should be TierIgnored")
	}
	if f.ClassifyFile("main.go") != TierNormal {
		t.Error("main.go should be TierNormal")
	}
}

func TestSecurityFilter_ClassifyFile_IgnoredBeforeRedacted(t *testing.T) {
	// .env is normally redacted; if also in extraIgnore it should be ignored.
	f, err := NewFilter([]string{".env"}, nil, nil)
	if err != nil {
		t.Fatalf("NewFilter: %v", err)
	}
	if f.ClassifyFile(".env") != TierIgnored {
		t.Error(".env in extraIgnore should be TierIgnored, not TierRedacted")
	}
}

func TestSecurityFilter_IsSecretValue(t *testing.T) {
	f, err := NewFilter(nil, nil, nil)
	if err != nil {
		t.Fatalf("NewFilter: %v", err)
	}

	secrets := []string{
		"api_key = 'AKIAIOSFODNN7EXAMPLE123'",
		"password = 'mysupersecretpassword'",
		"-----BEGIN RSA PRIVATE KEY-----",
		"postgres://user:pass@localhost/db",
	}
	for _, s := range secrets {
		if !f.IsSecretValue(s) {
			t.Errorf("IsSecretValue(%q) = false, want true", s)
		}
	}

	notSecrets := []string{
		"hello world",
		"const Timeout = 30",
		"type User struct {}",
	}
	for _, s := range notSecrets {
		if f.IsSecretValue(s) {
			t.Errorf("IsSecretValue(%q) = true, want false", s)
		}
	}
}

func TestSecurityFilter_ExtraSecretFilenames(t *testing.T) {
	f, err := NewFilter(nil, []string{"*.token"}, nil)
	if err != nil {
		t.Fatalf("NewFilter: %v", err)
	}
	if f.ClassifyFile("auth.token") != TierRedacted {
		t.Error("auth.token should be TierRedacted via extraSecret")
	}
}

// ---- IsBinary ---------------------------------------------------------------

func TestIsBinary_TextFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "code.go")
	if err := os.WriteFile(path, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin, err := IsBinary(path)
	if err != nil {
		t.Fatalf("IsBinary: %v", err)
	}
	if bin {
		t.Error("Go source file should not be binary")
	}
}

func TestIsBinary_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prog.exe")
	// Create a file with many null bytes (simulating binary).
	data := make([]byte, 1024)
	for i := range data {
		if i%2 == 0 {
			data[i] = 0 // null byte — 50% → well above threshold
		} else {
			data[i] = 0x41 // 'A'
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	bin, err := IsBinary(path)
	if err != nil {
		t.Fatalf("IsBinary: %v", err)
	}
	if !bin {
		t.Error("file with 50% null bytes should be detected as binary")
	}
}

func TestIsBinary_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	bin, err := IsBinary(path)
	if err != nil {
		t.Fatalf("IsBinary: %v", err)
	}
	if bin {
		t.Error("empty file should not be binary")
	}
}

// ---- IsPathSafe -------------------------------------------------------------

func TestIsPathSafe_NormalPath(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "src", "main.go")
	if err := os.MkdirAll(filepath.Dir(child), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	safe, err := IsPathSafe(root, child)
	if err != nil {
		t.Fatalf("IsPathSafe: %v", err)
	}
	if !safe {
		t.Error("child path should be safe")
	}
}

func TestIsPathSafe_TraversalPath(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "..", "outside.txt")

	// Create the outside file so EvalSymlinks can resolve it.
	outsideAbs := filepath.Join(filepath.Dir(root), "outside.txt")
	if err := os.WriteFile(outsideAbs, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(outsideAbs) })

	safe, err := IsPathSafe(root, outside)
	if err != nil {
		t.Fatalf("IsPathSafe: %v", err)
	}
	if safe {
		t.Error("path escaping root via .. should not be safe")
	}
}

func TestIsPathSafe_NonExistentFile(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "does_not_exist.go")

	safe, err := IsPathSafe(root, missing)
	if err != nil {
		t.Fatalf("IsPathSafe for non-existent file: %v", err)
	}
	if safe {
		t.Error("non-existent file should return safe=false")
	}
}

// ---- IsPlaceholder ----------------------------------------------------------

func TestIsPlaceholder(t *testing.T) {
	placeholders := []string{
		"YOUR_API_KEY",
		"<your-token>",
		"changeme",
		"xxxxxxxx",
		"REPLACE_THIS",
		"dummy",
		"fake_value",
		"example_key",
	}
	for _, v := range placeholders {
		if !IsPlaceholder(v) {
			t.Errorf("IsPlaceholder(%q) = false, want true", v)
		}
	}

	notPlaceholders := []string{
		"",
		"AKIAIOSFODNN7EXAMPLE123",
		"my-real-database-password",
		"sk-abc123realkey",
	}
	for _, v := range notPlaceholders {
		if IsPlaceholder(v) {
			t.Errorf("IsPlaceholder(%q) = true, want false", v)
		}
	}
}

// ---- ContainsSecretHint -----------------------------------------------------

func TestContainsSecretHint(t *testing.T) {
	hints := []string{"API_KEY", "SECRET", "PASSWORD", "DB_PASSWD", "TOKEN", "PRIVATE_KEY"}
	for _, name := range hints {
		if !ContainsSecretHint(name) {
			t.Errorf("ContainsSecretHint(%q) = false, want true", name)
		}
	}
	noHints := []string{"DATABASE_HOST", "SERVER_PORT", "APP_NAME"}
	for _, name := range noHints {
		if ContainsSecretHint(name) {
			t.Errorf("ContainsSecretHint(%q) = true, want false", name)
		}
	}
}
