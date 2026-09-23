package kassetpack

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAddFolderFilters verifies that extension filtering works correctly.
func TestAddFolderFilters(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "kassetpack_filter_test")
	defer os.RemoveAll(tempDir)

	// Create a PNG, a JPG, and a TXT file.
	os.WriteFile(filepath.Join(tempDir, "foo.png"), []byte("png"), 0644)
	os.WriteFile(filepath.Join(tempDir, "bar.jpg"), []byte("jpg"), 0644)
	os.WriteFile(filepath.Join(tempDir, "ignore.txt"), []byte("txt"), 0644)

	// Only include PNG and JPG files.
	builder := NewAssetBuilder()

	err := builder.AddFolder(
		tempDir,
		[]string{"png", "jpg"},
		"images",
	)
	if err != nil {
		t.Fatalf("AddFolder should not return an error: %v", err)
	}

	outDir, _ := os.MkdirTemp("", "kassetpack_filter_out")
	defer os.RemoveAll(outDir)

	builder.Save(outDir, "test_pack")

	bank := &AssetBank{}
	bank.Load(outDir, "test_pack", nil)
	defer bank.Close()

	// Verify that the expected files are present.
	if _, ok := bank.Index["images/foo.png"]; !ok {
		t.Errorf("foo.png should be present in the index")
	}

	if _, ok := bank.Index["images/bar.jpg"]; !ok {
		t.Errorf("bar.jpg should be present in the index")
	}

	// The TXT file MUST NOT be included.
	if _, ok := bank.Index["images/ignore.txt"]; ok {
		t.Errorf("ignore.txt should NOT be present in the index")
	}
}

// TestAddFolderNoFilter verifies that ALL files are included
// when no extension filter is provided.
func TestAddFolderNoFilter(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "kassetpack_nofilter_test")
	defer os.RemoveAll(tempDir)

	// Create a PNG, a JPG, and a TXT file.
	os.WriteFile(filepath.Join(tempDir, "foo.png"), []byte("png"), 0644)
	os.WriteFile(filepath.Join(tempDir, "bar.jpg"), []byte("jpg"), 0644)
	os.WriteFile(filepath.Join(tempDir, "ignore.txt"), []byte("txt"), 0644)

	// Pass an EMPTY extension list to include all files.
	builder := NewAssetBuilder()

	err := builder.AddFolder(
		tempDir,
		[]string{},
		"all_assets",
	)
	if err != nil {
		t.Fatalf("AddFolder should not return an error: %v", err)
	}

	outDir, _ := os.MkdirTemp("", "kassetpack_nofilter_out")
	defer os.RemoveAll(outDir)

	builder.Save(outDir, "test_pack_all")

	bank := &AssetBank{}
	bank.Load(outDir, "test_pack_all", nil)
	defer bank.Close()

	// All files MUST be present, including the TXT file.
	if _, ok := bank.Index["all_assets/foo.png"]; !ok {
		t.Errorf("foo.png should be present in the index")
	}

	if _, ok := bank.Index["all_assets/bar.jpg"]; !ok {
		t.Errorf("bar.jpg should be present in the index")
	}

	if _, ok := bank.Index["all_assets/ignore.txt"]; !ok {
		t.Errorf("ignore.txt SHOULD be present in the index because no filter was provided")
	}
}
