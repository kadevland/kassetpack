package kassetpack

import "testing"

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
	bank.Files[0] = nil

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
