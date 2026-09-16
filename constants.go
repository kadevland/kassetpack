package kassetpack

// File extensions for the KAssetPack format
const (
	IndexFileExt = ".kdx" // File index
	DataFileExt  = ".kdt" // File data
)

// DefaultMaxDataSize is the default maximum size for a .kdt file before it splits.
// Expressed in Megabytes (MB).
const DefaultMaxDataSize int64 = 600
