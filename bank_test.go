package kassetpack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewBank(t *testing.T) {
	bank := NewBank()

	if bank == nil {
		t.Fatal("NewBank() returned nil")
	}

	if bank.BasePath != "" {
		t.Errorf("expected empty BasePath, got %q", bank.BasePath)
	}

	if bank.BaseName != "" {
		t.Errorf("expected empty BaseName, got %q", bank.BaseName)
	}

	if bank.XORKey != nil {
		t.Errorf("expected nil XORKey, got %v", bank.XORKey)
	}

	if bank.Index == nil {
		t.Error("expected Index to be initialized")
	}

	if len(bank.Index) != 0 {
		t.Errorf("expected empty Index, got %d entries", len(bank.Index))
	}

	if bank.Files == nil {
		t.Error("expected Files to be initialized")
	}

	if len(bank.Files) != 0 {
		t.Errorf("expected empty Files, got %d entries", len(bank.Files))
	}
}

func TestAssetBankReset(t *testing.T) {
	bank := NewBank()

	bank.BasePath = "/tmp/test"
	bank.BaseName = "game"
	bank.XORKey = []byte("test-key")
	bank.Index["sprites/player.png"] = AssetEntry{}

	bank.reset()

	if bank.BasePath != "" {
		t.Errorf("expected empty BasePath, got %q", bank.BasePath)
	}

	if bank.BaseName != "" {
		t.Errorf("expected empty BaseName, got %q", bank.BaseName)
	}

	if bank.XORKey != nil {
		t.Errorf("expected nil XORKey, got %v", bank.XORKey)
	}

	if bank.Index == nil {
		t.Fatal("expected Index to be initialized")
	}

	if len(bank.Index) != 0 {
		t.Errorf("expected empty Index, got %d entries", len(bank.Index))
	}

	if bank.Files == nil {
		t.Fatal("expected Files to be initialized")
	}

	if len(bank.Files) != 0 {
		t.Errorf("expected empty Files, got %d entries", len(bank.Files))
	}
}
func TestNewBankClose(t *testing.T) {
	bank := NewBank()

	if err := bank.Close(); err != nil {
		t.Fatalf("Close() returned an unexpected error: %v", err)
	}

	if bank.Files == nil {
		t.Fatal("expected Files to remain initialized after Close()")
	}

	if len(bank.Files) != 0 {
		t.Errorf("expected empty Files after Close(), got %d entries", len(bank.Files))
	}
}

func TestUnpackAsset(t *testing.T) {
	sourceDir := t.TempDir()
	packDir := t.TempDir()
	outputDir := t.TempDir()

	content := []byte("hello from kassetpack")

	sourcePath := filepath.Join(sourceDir, "hello.txt")
	if err := os.WriteFile(sourcePath, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	builder := NewBuilder()
	builder.AppendAsset(sourcePath, "hello.txt")

	if err := builder.Save(packDir, "test_pack"); err != nil {
		t.Fatalf("Save should not return an error: %v", err)
	}

	bank := NewBank()

	if err := bank.Load(packDir, "test_pack", nil); err != nil {
		t.Fatalf("Load should not return an error: %v", err)
	}
	defer bank.Close()

	if err := bank.UnpackAsset("hello.txt", outputDir); err != nil {
		t.Fatalf("UnpackAsset should not return an error: %v", err)
	}

	extracted, err := os.ReadFile(filepath.Join(outputDir, "hello.txt"))
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(extracted) != string(content) {
		t.Errorf("unexpected extracted content: got %q, want %q", extracted, content)
	}
}
func TestUnpackAssetWithSubdirectory(t *testing.T) {
	sourceDir := t.TempDir()
	packDir := t.TempDir()
	outputDir := t.TempDir()

	content := []byte("player asset")

	sourcePath := filepath.Join(sourceDir, "player.txt")
	if err := os.WriteFile(sourcePath, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	builder := NewBuilder()
	builder.AppendAsset(sourcePath, "sprites/player.txt")

	if err := builder.Save(packDir, "test_pack"); err != nil {
		t.Fatalf("Save should not return an error: %v", err)
	}

	bank := NewBank()

	if err := bank.Load(packDir, "test_pack", nil); err != nil {
		t.Fatalf("Load should not return an error: %v", err)
	}
	defer bank.Close()

	if err := bank.UnpackAsset("sprites/player.txt", outputDir); err != nil {
		t.Fatalf("UnpackAsset should not return an error: %v", err)
	}

	extractedPath := filepath.Join(outputDir, "sprites", "player.txt")

	extracted, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(extracted) != string(content) {
		t.Errorf("unexpected extracted content: got %q, want %q", extracted, content)
	}
}

func TestUnpackAssetOverwrite(t *testing.T) {
	sourceDir := t.TempDir()
	packDir := t.TempDir()
	outputDir := t.TempDir()

	sourcePath := filepath.Join(sourceDir, "hello.txt")
	if err := os.WriteFile(sourcePath, []byte("new content"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	builder := NewBuilder()
	builder.AppendAsset(sourcePath, "hello.txt")

	if err := builder.Save(packDir, "test_pack"); err != nil {
		t.Fatalf("Save should not return an error: %v", err)
	}

	outputPath := filepath.Join(outputDir, "hello.txt")
	if err := os.WriteFile(outputPath, []byte("old content"), 0644); err != nil {
		t.Fatalf("failed to create existing output file: %v", err)
	}

	bank := NewBank()

	if err := bank.Load(packDir, "test_pack", nil); err != nil {
		t.Fatalf("Load should not return an error: %v", err)
	}
	defer bank.Close()

	if err := bank.UnpackAsset("hello.txt", outputDir); err != nil {
		t.Fatalf("UnpackAsset should not return an error: %v", err)
	}

	extracted, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(extracted) != "new content" {
		t.Errorf("unexpected extracted content: got %q, want %q", extracted, "new content")
	}
}

func TestUnpackAssetNotFound(t *testing.T) {
	bank := NewBank()

	err := bank.UnpackAsset("missing.txt", t.TempDir())
	if err == nil {
		t.Fatal("UnpackAsset should return an error for a missing asset")
	}
}

func TestUnpackAssetWithXOR(t *testing.T) {
	sourceDir := t.TempDir()
	packDir := t.TempDir()
	outputDir := t.TempDir()

	content := []byte("hello from xor kassetpack")
	key := []byte("test-xor-key")

	sourcePath := filepath.Join(sourceDir, "hello.txt")
	if err := os.WriteFile(sourcePath, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	builder := NewBuilder()
	builder.SetXORKey(key)
	builder.AppendAsset(sourcePath, "hello.txt")

	if err := builder.Save(packDir, "test_pack"); err != nil {
		t.Fatalf("Save should not return an error: %v", err)
	}

	bank := NewBank()

	if err := bank.Load(packDir, "test_pack", key); err != nil {
		t.Fatalf("Load should not return an error: %v", err)
	}
	defer bank.Close()

	if err := bank.UnpackAsset("hello.txt", outputDir); err != nil {
		t.Fatalf("UnpackAsset should not return an error: %v", err)
	}

	extracted, err := os.ReadFile(filepath.Join(outputDir, "hello.txt"))
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	if string(extracted) != string(content) {
		t.Errorf("unexpected extracted content: got %q, want %q", extracted, content)
	}
}
