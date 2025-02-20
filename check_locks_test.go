package main

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

// Lock folder using Windows API (using modern `windows` package)
func lockFolder(folderPath string) (windows.Handle, error) {
	ptr, err := windows.UTF16PtrFromString(folderPath)
	if err != nil {
		return 0, err
	}

	handle, err := windows.CreateFile(
		ptr,
		windows.GENERIC_READ,
		0, // No sharing, exclusive lock
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	return handle, err
}

func lockFile(filePath string) (windows.Handle, error) {
	ptr, err := windows.UTF16PtrFromString(filePath)
	if err != nil {
		return 0, err
	}

	handle, err := windows.CreateFile(
		ptr,
		windows.GENERIC_READ|windows.GENERIC_WRITE, // Open for read/write
		0, // No sharing allowed (exclusive lock)
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	return handle, err
}

// TestIsFolderLocked verifies that folder lock detection works.
func TestIsFolderLocked(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "testfolder-*")
	if err != nil {
		t.Fatalf("Failed to create temp folder: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Initially, the folder should NOT be locked
	if isFolderLocked(tempDir) {
		t.Errorf("Expected folder to be unlocked, but it was detected as locked.")
	}

	// Lock the folder using the Windows API
	handle, err := lockFolder(tempDir)
	if err != nil {
		t.Fatalf("Failed to lock folder: %v", err)
	}
	defer windows.CloseHandle(handle) // ✅ Corrected: Uses `windows.CloseHandle`

	// Now, the folder should be detected as locked
	if !isFolderLocked(tempDir) {
		t.Errorf("Expected folder to be locked, but it was detected as unlocked.")
	}
}

// TestIsFileLocked verifies that file lock detection works.
func TestIsFileLocked(t *testing.T) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "testfile-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close() // ✅ Close the temp file before testing

	defer os.Remove(tempFile.Name()) // Clean up after test

	// Ensure the file is initially unlocked
	if isFileLocked(tempFile.Name()) {
		t.Errorf("Expected file to be unlocked, but it was detected as locked.")
	}

	// Lock the file using Windows API
	handle, err := lockFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to lock file: %v", err)
	}
	defer windows.CloseHandle(handle) // Keep it open to simulate a lock

	// Now, the file should be detected as locked
	if !isFileLocked(tempFile.Name()) {
		t.Errorf("Expected file to be locked, but it was detected as unlocked.")
	}
}
