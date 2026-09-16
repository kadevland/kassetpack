package kassetpack

// AssetEntry represents the metadata of a file stored in a data file.
// It contains the physical location (FileID, Offset, Size) and the integrity hash.
type AssetEntry struct {
	Path     string // The original full path (e.g. "assets/background.png")
	FileID   int    // ID of the data file containing the asset (e.g. 1 for core_01 data file)
	Offset   int64  // Starting byte offset
	Size     int64  // Number of bytes to read
	Checksum string // SHA256 hash used for deduplication during the build
}
