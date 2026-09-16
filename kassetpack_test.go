package kassetpack

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// generateRandomBytes generates a byte slice of the requested size.
func generateRandomBytes(size int) []byte {
	b := make([]byte, size)
	rand.Read(b) // Fill the buffer with random data
	return b
}

// TestNoAlias verifies that when no alias is provided,
// the physical path automatically becomes the logical key in the index.
func TestNoAlias(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kassetpack_noalias") // Create a temporary directory for the test
	if err != nil {
		t.Fatalf("Unable to create the temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test

	// Generate random asset data and a random key
	assetData := generateRandomBytes(128)
	secretKey := generateRandomBytes(16)

	// Create the physical file
	physicalPath := filepath.Join(tempDir, "background.txt")
	os.WriteFile(physicalPath, assetData, 0644)

	// Use the Builder WITHOUT providing an alias (1 argument only)
	builder := NewAssetBuilder()
	builder.SetXORKey(secretKey)
	builder.AppendAsset(physicalPath) // <--- No alias provided

	if err := builder.Save(tempDir, "noalias_pack"); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Load the pack
	bank := &AssetBank{}
	if err := bank.Load(tempDir, "noalias_pack", secretKey); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer bank.Close()

	// VERIFICATION 1: The index key must exactly match the physical path
	entry, exists := bank.Index[physicalPath]
	if !exists {
		t.Fatalf("The file should be indexed under its physical path: %s", physicalPath)
	}

	// VERIFICATION 2: The asset must be readable using the same path
	readData, err := bank.Read(physicalPath)
	if err != nil {
		t.Fatalf("Error reading with the default path: %v", err)
	}

	if !bytes.Equal(readData, assetData) {
		t.Errorf("Data read without an alias does not match.")
	}

	// Small log to confirm that it works
	t.Logf("File successfully read without an alias using the key: %s", entry.Path)
}

// TestBuildAndRead is an end-to-end test.
func TestBuildAndRead(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "kassetpack_test") // Create a temporary directory for the test
	if err != nil {
		t.Fatalf("Unable to create the temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test

	// Create fake files with random data
	assetOneData := generateRandomBytes(256)
	assetTwoData := generateRandomBytes(1024)

	os.WriteFile(filepath.Join(tempDir, "asset1.txt"), assetOneData, 0644)
	os.WriteFile(filepath.Join(tempDir, "asset2.txt"), assetTwoData, 0644)

	secretKey := generateRandomBytes(32)
	// Use the Builder to create the pack
	builder := NewAssetBuilder()
	builder.SetXORKey(secretKey)
	builder.SetMaxDataSize(1) // Set a 1 MB limit for testing

	// Add assetOne normally
	builder.AppendAsset(filepath.Join(tempDir, "asset1.txt"), "logical/asset1.txt")
	// Add assetTwo with a completely different alias
	builder.AppendAsset(filepath.Join(tempDir, "asset2.txt"), "logical/asset2_renamed.txt")

	// Generate the .kdx and .kdt files
	if err := builder.Save(tempDir, "core"); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Load the pack
	bank := &AssetBank{}
	if err := bank.Load(tempDir, "core", secretKey); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer bank.Close()

	// Assertions

	// AssetOne: The read content must exactly match the original file
	readAssetOne, err := bank.Read("logical/asset1.txt")
	if err != nil {
		t.Fatalf("Error reading assetOne: %v", err)
	}
	if !bytes.Equal(readAssetOne, assetOneData) {
		t.Errorf("assetOne data does not match.")
	}

	// AssetTwo: The asset must be readable using its logical alias
	readAssetTwo, err := bank.Read("logical/asset2_renamed.txt")
	if err != nil {
		t.Fatalf("Error reading assetTwo (alias): %v", err)
	}
	if !bytes.Equal(readAssetTwo, assetTwoData) {
		t.Errorf("assetTwo data does not match.")
	}

	// Test missing file
	_, err = bank.Read("logical/inexistant.txt")
	if err == nil {
		t.Errorf("An error should have been returned for a missing file")
	}
}

// TestDeduplication verifies that two different aliases pointing to the same physical file
// are only written once to the data file.
func TestDeduplication(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kassetpack_dedup") // Create a temporary directory for the test
	if err != nil {
		t.Fatalf("Unable to create the temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test

	// Create a file with random data
	sharedData := generateRandomBytes(512)
	os.WriteFile(filepath.Join(tempDir, "shared.txt"), sharedData, 0644)

	secretKey := generateRandomBytes(32)
	builder := NewAssetBuilder()
	builder.SetXORKey(secretKey)

	// Add the SAME physical file, but with two different logical aliases
	builder.AppendAsset(filepath.Join(tempDir, "shared.txt"), "alias_1/data.txt")
	builder.AppendAsset(filepath.Join(tempDir, "shared.txt"), "alias_2/data.txt")

	builder.Save(tempDir, "dedup_pack")

	// Load the pack
	bank := &AssetBank{}
	bank.Load(tempDir, "dedup_pack", secretKey)
	defer bank.Close()

	// Read both entries from the index
	entry1 := bank.Index["alias_1/data.txt"]
	entry2 := bank.Index["alias_2/data.txt"]

	// ASSERTION: Both entries must point to the SAME Offset and the SAME Size
	if entry1.Offset != entry2.Offset || entry1.Size != entry2.Size {
		t.Errorf("Deduplication failed: offsets or sizes differ.")
	}

	// Both must return the correct data
	data1, _ := bank.Read("alias_1/data.txt")
	data2, _ := bank.Read("alias_2/data.txt")

	if !bytes.Equal(data1, sharedData) || !bytes.Equal(data2, sharedData) {
		t.Errorf("Deduplicated data is incorrect.")
	}
}

// TestSplitting verifies that data files are physically split
// when they exceed MaxDataSize, and that reading still works.
func TestSplitting(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kassetpack_split") // Create a temporary directory for the test
	if err != nil {
		t.Fatalf("Unable to create the temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test

	// Generate two 700 KB assets.
	// With MaxDataSize=1 (1 MB), the second asset exceeds the limit
	// and forces the creation of a new data file.
	assetOneData := generateRandomBytes(700 * 1024)
	assetTwoData := generateRandomBytes(700 * 1024)
	secretKey := generateRandomBytes(32)

	os.WriteFile(filepath.Join(tempDir, "asset1.txt"), assetOneData, 0644)
	os.WriteFile(filepath.Join(tempDir, "asset2.txt"), assetTwoData, 0644)

	// Force the limit to 1 MB (1 * 1024 * 1024 bytes)
	builder := NewAssetBuilder()
	builder.SetXORKey(secretKey)
	builder.SetMaxDataSize(1)

	builder.AppendAsset(filepath.Join(tempDir, "asset1.txt"), "data/asset1.txt")
	builder.AppendAsset(filepath.Join(tempDir, "asset2.txt"), "data/asset2.txt")

	if err := builder.Save(tempDir, "core"); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify that the first data file exists physically (core.kdt)
	datPath1 := filepath.Join(tempDir, "core.kdt")
	info1, err := os.Stat(datPath1)
	if err != nil {
		t.Fatalf("The core.kdt file was not created: %v", err)
	}
	t.Logf("Size of core.kdt (FileID 0): %d bytes", info1.Size())

	// Verify that the SECOND data file was created physically (core_01.kdt)
	datPath2 := filepath.Join(tempDir, "core_01.kdt")
	info2, err := os.Stat(datPath2)
	if err != nil {
		t.Fatalf("The core_01.kdt file was not created (splitting failed): %v", err)
	}
	t.Logf("Size of core_01.kdt (FileID 1): %d bytes", info2.Size())

	// Load the pack
	bank := &AssetBank{}
	if err := bank.Load(tempDir, "core", secretKey); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer bank.Close()

	// Verify that the index points to the correct FileIDs
	entry1 := bank.Index["data/asset1.txt"]
	if entry1.FileID != 0 {
		t.Errorf("asset1 FileID should be 0 (in core.kdt), got: %d", entry1.FileID)
	}

	entry2 := bank.Index["data/asset2.txt"]
	if entry2.FileID != 1 {
		t.Errorf("asset2 FileID should be 1 (in core_01.kdt), got: %d", entry2.FileID)
	}

	// Verify that asset1 can be read from core.kdt
	readData1, err := bank.Read("data/asset1.txt")
	if err != nil {
		t.Fatalf("Error reading asset1 from core.kdt: %v", err)
	}
	if !bytes.Equal(readData1, assetOneData) {
		t.Errorf("asset1 data is incorrect after splitting.")
	}
	t.Logf("Asset1 successfully read from core.kdt")

	// THE CRUCIAL TEST: Retrieve asset2 from core_01.kdt
	readData2, err := bank.Read("data/asset2.txt")
	if err != nil {
		t.Fatalf("Error reading asset2 from core_01.kdt: %v", err)
	}
	if !bytes.Equal(readData2, assetTwoData) {
		t.Errorf("asset2 data is incorrect after splitting.")
	}
	t.Logf("Asset2 successfully read from core_01.kdt (FileID 1)")
}

// TestMassiveDeduplication verifies that adding the same physical file 20 times
// generates 20 index entries, but writes it only ONCE physically to the data file.

func TestMassiveDeduplication(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kassetpack_massdup") // Create a temporary directory for the test
	if err != nil {
		t.Fatalf("Unable to create the temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after the test

	// Create a 100 KB file
	originalData := generateRandomBytes(100 * 1024)
	secretKey := generateRandomBytes(32)

	physicalPath := filepath.Join(tempDir, "original.txt")
	os.WriteFile(physicalPath, originalData, 0644)

	// Get the original file size for comparison
	fileInfo, _ := os.Stat(physicalPath)
	expectedSize := fileInfo.Size()

	builder := NewAssetBuilder()
	builder.SetXORKey(secretKey)

	// Add the SAME file 20 times with different aliases
	for i := 1; i <= 20; i++ {
		alias := fmt.Sprintf("duplicates/file_%d.txt", i)
		builder.AppendAsset(physicalPath, alias)
	}
	if err := builder.Save(tempDir, "mass_dup"); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify that we have exactly 20 entries in the index
	bank := &AssetBank{}
	if err := bank.Load(tempDir, "mass_dup", secretKey); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	defer bank.Close()

	if len(bank.Index) != 20 {
		t.Fatalf("The index should contain 20 entries, but contains %d", len(bank.Index))
	}

	// Verify the physical size of the data file on disk
	datPath := filepath.Join(tempDir, "mass_dup.kdt")
	datInfo, err := os.Stat(datPath)
	if err != nil {
		t.Fatalf("The .kdt file was not created: %v", err)
	}

	t.Logf("Original file size: %d bytes", expectedSize)
	t.Logf("Final .kdt file size: %d bytes", datInfo.Size())

	// CRUCIAL TEST: The data file must be exactly the size of ONE file (not 20)
	if datInfo.Size() != expectedSize {
		t.Errorf("The .kdt file should be %d bytes (1x file), but is %d bytes (duplicates were written physically!)", expectedSize, datInfo.Size())
	}

	// Verify that EACH of the 20 index keys exists and returns the correct data
	for i := 1; i <= 20; i++ {
		alias := fmt.Sprintf("duplicates/file_%d.txt", i)

		// Verify that the key exists
		entry, exists := bank.Index[alias]
		if !exists {
			t.Errorf("Entry %s should exist in the index", alias)
			continue // Skip if the key does not exist
		}

		// Verify that all entries point to the same Offset (proof of deduplication)
		if entry.Offset != 0 {
			t.Errorf("Entry %s points to offset %d instead of 0 (logical deduplication failed)", alias, entry.Offset)
		}

		// Read and verify the data
		readData, err := bank.Read(alias)
		if err != nil {
			t.Errorf("Error reading alias %s: %v", alias, err)
			continue
		}
		if !bytes.Equal(readData, originalData) {
			t.Errorf("Data for alias %s is incorrect.", alias)
		}
	}
}
