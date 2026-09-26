package kassetpack

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// AssetBankInterface defines the contract for reading an asset bank.
// It abstracts the file system access, allowing implementations for
// packed files (data and index) or direct OS file system reading (Mockup/Dev mode).
type AssetBankInterface interface {
	// Load initializes the bank by reading the specified index file (.kdx).
	// It opens the data files lazily as they are requested.
	Load(basePath string, baseName string, xorKey []byte) error

	// Read loads a full asset from the data file and returns its raw, decrypted bytes.
	Read(path string) ([]byte, error)

	// GetFile retrieves the underlying OS file handle for a specific FileID.
	// It implements Lazy Loading: if the file isn't open, it opens it and caches the pointer.
	GetFile(fileID int) (*os.File, error)

	// Open returns an io.ReadCloser for streaming large assets (like audio or video)
	// without loading the entire file into RAM.
	Open(path string) (io.ReadCloser, error)

	// Close safely closes all opened data files
	// It should be called when the game shuts down.
	Close() error
}

// AssetBank represents the runtime storage for packed assets.
// It reads encrypted data files using a pre-loaded index
type AssetBank struct {
	BasePath string
	BaseName string
	XORKey   []byte
	Index    map[string]AssetEntry

	// Lazy Loading: A map to store opened files on the fly
	Files  map[int]*os.File
	locker sync.Mutex // Readability over conventions!
}

// NewBank creates and initializes a new AssetBank.
func NewBank() *AssetBank {
	bank := &AssetBank{}
	bank.reset()

	return bank
}

// reset restores the bank to a clean, initialized state.
func (bank *AssetBank) reset() {
	bank.BasePath = ""
	bank.BaseName = ""
	bank.XORKey = nil
	bank.Index = make(map[string]AssetEntry)
	bank.Files = make(map[int]*os.File)
}

// Load initializes the bank by reading the index file.
// It maps the logical asset paths to their physical location in the data files.
func (bank *AssetBank) Load(basePath string, baseName string, xorKey []byte) error {

	if err := bank.Close(); err != nil {
		return err
	}

	bank.reset()

	bank.BasePath = basePath
	bank.BaseName = baseName
	bank.XORKey = xorKey

	// Read the encrypted index file
	idxPath := filepath.Join(basePath, baseName+IndexFileExt)
	encryptedData, err := os.ReadFile(idxPath)
	if err != nil {
		return fmt.Errorf("failed to open index file %s: %w", idxPath, err)
	}

	// Decrypt the XOR data in memory
	if len(bank.XORKey) > 0 {
		for i := 0; i < len(encryptedData); i++ {
			encryptedData[i] ^= bank.XORKey[i%len(bank.XORKey)]
		}
	}

	// Decode the Gob from the decrypted bytes
	decoder := gob.NewDecoder(bytes.NewReader(encryptedData))
	if err := decoder.Decode(&bank.Index); err != nil {
		return fmt.Errorf("failed to decode index data: %w", err)
	}

	return nil
}

// GetFile retrieves the underlying OS file handle for a specific FileID.
// It implements Lazy Loading: if the file isn't open, it opens it and caches the pointer.
func (bank *AssetBank) GetFile(fileID int) (*os.File, error) {
	// Validate FileID
	if fileID < 0 {
		return nil, fmt.Errorf("invalid FileID: %d", fileID)
	}

	bank.locker.Lock()
	defer bank.locker.Unlock()

	// Double-check if the file is already open
	if file, exists := bank.Files[fileID]; exists {
		return file, nil
	}

	// Construct path and open
	var datPath string
	if fileID == 0 {
		// No _number
		datPath = filepath.Join(bank.BasePath, bank.BaseName+DataFileExt)
	} else {
		// With _number (ex: core_01)
		datPath = filepath.Join(bank.BasePath, fmt.Sprintf("%s_%02d%s", bank.BaseName, fileID, DataFileExt))
	}

	datFile, err := os.Open(datPath)
	if err != nil {
		return nil, fmt.Errorf("invalid FileID or missing file: %w", err)
	}

	// Store it for next time
	bank.Files[fileID] = datFile
	return datFile, nil
}

// Read loads a full asset from the data file, decrypts it on the fly, and returns the raw bytes.
func (bank *AssetBank) Read(path string) ([]byte, error) {
	entry, exists := bank.Index[path]
	if !exists {
		return nil, fmt.Errorf("file not found in index: %s", path)
	}

	// Lazy Load: Get the file (opens it if not already open)
	file, err := bank.GetFile(entry.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data file for %s: %w", path, err)
	}

	// Thread-safe section reading
	section := io.NewSectionReader(file, entry.Offset, entry.Size)

	// Decrypt on the fly
	var reader io.Reader = section
	if len(bank.XORKey) > 0 {
		reader = NewXORReader(section, bank.XORKey)
	}

	// Read all bytes
	buffer, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read/decrypt %s: %w", path, err)
	}

	return buffer, nil
}

// Open returns an io.ReadCloser for streaming large assets (like audio or video)
// without loading the entire file into RAM.
func (bank *AssetBank) Open(path string) (io.ReadCloser, error) {
	entry, exists := bank.Index[path]
	if !exists {
		return nil, fmt.Errorf("file not found in index: %s", path)
	}

	file, err := bank.GetFile(entry.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data file for %s: %w", path, err)
	}

	section := io.NewSectionReader(file, entry.Offset, entry.Size)

	var reader io.Reader = section
	if len(bank.XORKey) > 0 {
		reader = NewXORReader(section, bank.XORKey)
	}

	// Wrap in NopCloser so it implements io.ReadCloser
	return io.NopCloser(reader), nil
}

// Close safely closes all opened data files.
func (bank *AssetBank) Close() error {
	bank.locker.Lock()
	defer bank.locker.Unlock()

	var lastErr error
	for _, file := range bank.Files {
		if err := file.Close(); err != nil {
			lastErr = err
		}
	}

	// Clear the map
	bank.Files = make(map[int]*os.File)

	return lastErr
}
