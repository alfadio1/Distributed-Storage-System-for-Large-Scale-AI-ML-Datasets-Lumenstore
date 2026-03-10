package node

import (
	"hash/crc32"
	"testing"
)

func TestStoragePutAndGetChunk(t *testing.T) {
	baseDir := t.TempDir()
	storage := NewStorage(baseDir)

	chunkID := "my/object:path v1"
	data := []byte("hello-test")
	expectedCRC := crc32.ChecksumIEEE(data)

	bytesWritten, err := storage.PutChunk(chunkID, data, expectedCRC)
	if err != nil {
		t.Fatalf("PutChunk returned error: %v", err)
	}

	if bytesWritten != uint64(len(data)) {
		t.Fatalf("bytesWritten = %d, want %d", bytesWritten, len(data))
	}

	gotData, gotCRC, bytesRead, err := storage.GetChunk(chunkID)
	if err != nil {
		t.Fatalf("GetChunk returned error: %v", err)
	}

	if bytesRead != uint64(len(data)) {
		t.Fatalf("bytesRead = %d, want %d", bytesRead, len(data))
	}

	if string(gotData) != string(data) {
		t.Fatalf("data mismatch: got %q, want %q", string(gotData), string(data))
	}

	if gotCRC != expectedCRC {
		t.Fatalf("crc mismatch: got %d, want %d", gotCRC, expectedCRC)
	}
}
