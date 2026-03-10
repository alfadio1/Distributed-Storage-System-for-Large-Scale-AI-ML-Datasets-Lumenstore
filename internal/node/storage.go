package node

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
)

// Storage handles chunk persistence on disk.
type Storage struct {
	baseDir string
}

// NewStorage initializes storage under a base directory.
func NewStorage(baseDir string) *Storage {
	return &Storage{baseDir: baseDir}
}

// safeChunkFileName converts a chunkID into a filesystem-safe deterministic filename.
func safeChunkFileName(chunkID string) string {
	sum := sha256.Sum256([]byte(chunkID))
	return hex.EncodeToString(sum[:])
}

// chunkPath returns the on-disk path for a chunk.
func (s *Storage) chunkPath(chunkID string) string {
	return filepath.Join(s.baseDir, safeChunkFileName(chunkID))
}

// PutChunk writes a chunk to disk using an atomic write.
func (s *Storage) PutChunk(chunkID string, data []byte, expectedCRC uint32) (uint64, error) {
	actualCRC := crc32.ChecksumIEEE(data)
	if actualCRC != expectedCRC {
		return 0, errors.New("crc mismatch")
	}

	finalPath := s.chunkPath(chunkID)
	tmpPath := finalPath + ".tmp"

	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return 0, err
	}

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return 0, err
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return 0, err
	}

	return uint64(len(data)), nil
}

// GetChunk reads a chunk from disk.
func (s *Storage) GetChunk(chunkID string) ([]byte, uint32, uint64, error) {
	path := s.chunkPath(chunkID)

	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, 0, 0, err
	}

	crc := crc32.ChecksumIEEE(data)

	return data, crc, uint64(len(data)), nil
}
