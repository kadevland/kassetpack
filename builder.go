package kassetpack

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// AssetBuilderInterface defines how the builder is used
type AssetBuilderInterface interface {
	// SetXORKey sets the obfuscation key
	SetXORKey(key []byte)

	// SetMaxDataSize sets the maximum size of data files (in megabytes)
	SetMaxDataSize(sizeInMB int64)

	// AppendFile adds a file to the list of files to package
	AppendAsset(filePath string)

	// Save processes the added files and generates the index and data files
	Save(destPath string, baseName string) error

	// ClearFiles resets ONLY the list of files to package
	// to reuse the same instance with the same configuration (XOR, MaxSize).
	ClearFiles()
}

// AssetBuilder is used to construct the data and index files.
// It handles asset deduplication, XOR obfuscation, and file splitting.
type assetFile struct {
	physicalPath string
	logicalKey   string
}

// AssetBuilder is used to construct the data and index files.
// It handles asset deduplication, XOR encryption, and file splitting.
type AssetBuilder struct {
	XORKey      []byte
	MaxDataSize int64 // En Mégaoctets

	// État de configuration
	assetToPack []assetFile
}

// NewAssetBuilder creates and returns a new AssetBuilder instance.
func NewAssetBuilder() *AssetBuilder {
	return &AssetBuilder{
		MaxDataSize: DefaultMaxDataSize,
	}
}

// SetXORKey defines the binary key used to obfuscate the assets.
func (b *AssetBuilder) SetXORKey(key []byte) {
	b.XORKey = key
}

// SetMaxDataSize defines the maximum size of a .kdt file before it splits (in Megabytes).
func (b *AssetBuilder) SetMaxDataSize(sizeInMB int64) {
	b.MaxDataSize = sizeInMB
}

// AppendAsset adds a physical file to the list of assets to be packaged.
// It accepts an optional alias: if provided, the asset will be stored in the index
// under this alias instead of its physical file path.
func (b *AssetBuilder) AppendAsset(filePath string, alias ...string) {
	key := filePath
	if len(alias) > 0 {
		key = alias[0]
	}
	b.assetToPack = append(b.assetToPack, assetFile{
		physicalPath: filePath,
		logicalKey:   key,
	})
}

// ClearFiles resets the list of assets to be packaged, keeping the configuration.
func (b *AssetBuilder) ClearFiles() {
	b.assetToPack = nil
}

// Save processes the added assets and generates the data and index files.
func (b *AssetBuilder) Save(destPath string, baseName string) error {

	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	maxBytes := b.MaxDataSize * 1024 * 1024

	index := make(map[string]AssetEntry)
	checksums := make(map[string]AssetEntry) // For deduplication

	var currentFile *os.File
	var currentSize int64 = 0
	fileID := 0

	defer func() {
		if currentFile != nil {
			currentFile.Close()
		}
	}()

	// Open the first data file
	var err error
	currentFile, err = b.openNewDataFile(destPath, baseName, fileID)
	if err != nil {
		return err
	}

	// Process each requested asset
	for _, asset := range b.assetToPack {
		// Calculate the checksum from the physical asset
		checksum, err := calculateChecksum(asset.physicalPath)
		if err != nil {
			return fmt.Errorf("failed to checksum %s: %w", asset.physicalPath, err)
		}

		// Check for deduplication
		if existing, exists := checksums[checksum]; exists {
			// The asset already exists physically in the data file!
			// Create a new entry pointing to the same physical data,
			// but with the current logical key (alias).
			entry := AssetEntry{
				Path:     asset.logicalKey,
				FileID:   existing.FileID,
				Offset:   existing.Offset,
				Size:     existing.Size,
				Checksum: existing.Checksum,
			}

			index[asset.logicalKey] = entry
			continue
		}

		// Get the physical file size
		info, err := os.Stat(asset.physicalPath)
		if err != nil {
			return fmt.Errorf("file not found %s: %w", asset.physicalPath, err)
		}
		fileSize := info.Size()

		// Check whether the current data file would exceed the limit
		if currentSize+fileSize > maxBytes {
			currentFile.Close()
			fileID++
			currentSize = 0

			currentFile, err = b.openNewDataFile(destPath, baseName, fileID)
			if err != nil {
				return err
			}
		}

		// Write the asset to the data file with XOR
		offset := currentSize
		// The copyWithXOR function already contains the XOR logic for the data!
		if err := b.copyWithXOR(asset.physicalPath, currentFile); err != nil {
			return fmt.Errorf("failed to write %s: %w", asset.physicalPath, err)
		}
		currentSize += fileSize

		// Create the index entry with the logical key
		entry := AssetEntry{
			Path:     asset.logicalKey,
			FileID:   fileID,
			Offset:   offset,
			Size:     fileSize,
			Checksum: checksum,
		}

		index[asset.logicalKey] = entry
		checksums[checksum] = entry
	}

	// Write the index using XOR obfuscation
	idxPath := filepath.Join(destPath, baseName+IndexFileExt)

	// Encode the index in memory (buffer)
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(index); err != nil {
		return fmt.Errorf("failed to encode index: %w", err)
	}

	// Retrieve the bytes
	idxData := buffer.Bytes()

	// Apply XOR to the entire index!
	if len(b.XORKey) > 0 {
		for i := 0; i < len(idxData); i++ {
			idxData[i] ^= b.XORKey[i%len(b.XORKey)]
		}
	}

	// Write the obfuscated index to disk
	if err := os.WriteFile(idxPath, idxData, 0644); err != nil {
		return fmt.Errorf("failed to write index file %s: %w", idxPath, err)
	}

	return nil
}

// openNewDataFile creates and opens a new data file.
func (b *AssetBuilder) openNewDataFile(destPath, baseName string, fileID int) (*os.File, error) {
	var datPath string
	if fileID == 0 {
		datPath = filepath.Join(destPath, baseName+DataFileExt)
	} else {
		datPath = filepath.Join(destPath, fmt.Sprintf("%s_%02d%s", baseName, fileID, DataFileExt))
	}

	datFile, err := os.Create(datPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create data file %s: %w", datPath, err)
	}

	return datFile, nil
}

// copyWithXOR reads a source file, applies XOR, and writes it to the destination.
func (b *AssetBuilder) copyWithXOR(srcPath string, dest *os.File) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	buffer := make([]byte, 32*1024)
	var pos int64 = 0

	for {
		n, readErr := src.Read(buffer)
		if n > 0 {
			if len(b.XORKey) > 0 {
				for i := 0; i < n; i++ {
					buffer[i] ^= b.XORKey[(pos+int64(i))%int64(len(b.XORKey))]
				}
			}

			if _, err := dest.Write(buffer[:n]); err != nil {
				return err
			}
			pos += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

// AddFolder recursively scans a directory and adds files matching
// the allowed extensions.
//
// It replaces the base of the physical path with the logical rebasePath.
//
// Example:
//
// folderPath:  "./assets/images"
// extensions:  []string{"jpg", "png"}
// rebasePath:  "images"
//
// A file located at "./assets/images/bg.png" will be stored in the index
// with the key "images/bg.png".
func (b *AssetBuilder) AddFolder(folderPath string, extensions []string, rebasePath string) error {
	// If the extension list is empty, allow all files.
	allowAll := len(extensions) == 0

	// Prepare a map for fast extension lookup (lowercase).
	allowedExt := make(map[string]bool)

	for _, ext := range extensions {
		// Clean the extension by removing dots or asterisks
		// in case the user included them.
		cleanExt := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(ext, "*"), "."))
		allowedExt[cleanExt] = true
	}

	// Walk through the physical directory.
	err := filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Ignore directories.
		if d.IsDir() {
			return nil
		}

		if !allowAll {
			// Check the file extension.
			fileExt := strings.ToLower(
				strings.TrimPrefix(filepath.Ext(path), "."),
			)

			if !allowedExt[fileExt] {
				return nil // Ignore this file.
			}
		}

		// Compute the logical path (alias).
		//
		// Example:
		// path       = "./assets/images/bg.png"
		// folderPath = "./assets/images"
		// relPath    = "bg.png"
		relPath, err := filepath.Rel(folderPath, path)
		if err != nil {
			return err
		}

		// Replace Windows path separators (\) with forward slashes (/).
		relPath = filepath.ToSlash(relPath)

		// Build the final key: "images/bg.png".
		var logicalKey string
		if rebasePath != "" {
			logicalKey = rebasePath + "/" + relPath
		} else {
			logicalKey = relPath
		}

		// Add the asset using its logical alias.
		b.AppendAsset(path, logicalKey)

		return nil
	})

	return err
}

// calculateChecksum reads a file and returns its SHA256 hash.
func calculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
