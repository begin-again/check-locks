//go:build windows
// +build windows

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// TestGetAbsoluteExclusions verifies that relative exclusions are converted correctly.
func TestGetAbsoluteExclusions(t *testing.T) {
	root := `C:\root`
	excludes := []string{"sub1", "sub2"}
	expected := []string{filepath.Join(root, "sub1"), filepath.Join(root, "sub2")}
	result := getAbsoluteExclusions(root, excludes)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// TestIsFileLocked creates a temporary file, checks it is not locked,
// then opens it exclusively using the Windows API so that isFileLocked returns true.
func TestIsFileLocked(t *testing.T) {
	// Create temporary file.
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	tmpFileName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFileName)

	// Initially, file should not be locked.
	if isFileLocked(tmpFileName) {
		t.Errorf("Expected file not locked")
	}

	// Lock file: open it exclusively (no sharing allowed).
	ptr, err := windows.UTF16PtrFromString(tmpFileName)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(
		ptr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, // No sharing.
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	// Ensure the handle is closed after testing.
	defer windows.CloseHandle(handle)

	// Now, the file should be considered locked.
	if !isFileLocked(tmpFileName) {
		t.Errorf("Expected file to be locked")
	}
}

// TestIsFolderLocked creates a temporary folder with a subfolder.
// First, it verifies that an unlocked folder is reported as not locked.
// Then it simulates a lock by opening an exclusive handle on the subfolder.
func TestIsFolderLocked(t *testing.T) {
	// Create a temporary folder.
	tmpDir, err := os.MkdirTemp("", "testfolder")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subfolder inside.
	subFolder := filepath.Join(tmpDir, "sub")
	err = os.Mkdir(subFolder, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// With an unlocked subfolder, the folder should not be reported as locked.
	if isFolderLocked(tmpDir) {
		t.Errorf("Expected folder not locked")
	}

	// Now, simulate a locked folder by opening an exclusive handle on the subfolder.
	ptr, err := windows.UTF16PtrFromString(subFolder)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(
		ptr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, // No sharing.
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS, // Required for directories.
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)

	// With the subfolder exclusively opened, the folder should now be reported as locked.
	if !isFolderLocked(tmpDir) {
		t.Errorf("Expected folder to be locked")
	}
}

func TestCheckLocks_NoLocks(t *testing.T) {
	// Create a temporary directory.
	tmpDir, err := os.MkdirTemp("", "test_no_locks")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file that is not locked.
	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Call checkLocks directly.
	output, exitCode := checkLocks(tmpDir, []string{})

	// Expect output indicating no locks.
	expected := `{"locked_files": [], "locked_folders": []}`
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain %s, got: %s", expected, output)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got: %d", exitCode)
	}
}

func TestCheckLocks_FileLocked(t *testing.T) {
	// Create a temporary directory.
	tmpDir, err := os.MkdirTemp("", "test_file_locked")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file.
	filePath := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Lock the file using the Windows API.
	ptr, err := windows.UTF16PtrFromString(filePath)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(
		ptr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, // no sharing
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	// Keep the handle open to maintain the lock.

	output, exitCode := checkLocks(tmpDir, []string{})
	// Release the lock.
	windows.CloseHandle(handle)

	if !strings.Contains(output, `"locked_files": [`) {
		t.Errorf("Expected output to contain locked_files, got: %s", output)
	}
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got: %d", exitCode)
	}
}

func TestCheckLocks_FolderLocked(t *testing.T) {
	// Create a temporary directory.
	tmpDir, err := os.MkdirTemp("", "test_folder_locked")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subfolder inside.
	subFolder := filepath.Join(tmpDir, "subfolder")
	if err := os.Mkdir(subFolder, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file in the subfolder.
	filePath := filepath.Join(subFolder, "file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Lock the subfolder by obtaining an exclusive handle.
	ptr, err := windows.UTF16PtrFromString(subFolder)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(
		ptr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, // no sharing
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS, // necessary for directories
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	// Keep the handle open to maintain the lock.

	output, exitCode := checkLocks(tmpDir, []string{})
	// Release the lock.
	windows.CloseHandle(handle)

	if !strings.Contains(output, `"locked_folders": [`) {
		t.Errorf("Expected output to contain locked_folders, got: %s", output)
	}
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got: %d", exitCode)
	}
}

func TestRun_NoLocks(t *testing.T) {
	// Create a temporary directory.
	tmpDir, err := os.MkdirTemp("", "run_no_locks")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file that is not locked.
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Build a config with no exclusions.
	cfg := Config{
		Root:    tmpDir,
		Exclude: []string{},
	}

	// Call run.
	output, exitCode := run(cfg)

	// Expect no locks.
	expected := `{"locked_files": [], "locked_folders": []}`
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain %s, got: %s", expected, output)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got: %d", exitCode)
	}
}

func TestRun_FileLocked(t *testing.T) {
	// Create a temporary directory.
	tmpDir, err := os.MkdirTemp("", "run_file_locked")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file.
	filePath := filepath.Join(tmpDir, "locked.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Lock the file using the Windows API.
	ptr, err := windows.UTF16PtrFromString(filePath)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(
		ptr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, // no sharing
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	// Ensure the handle remains open to maintain the lock.

	cfg := Config{
		Root:    tmpDir,
		Exclude: []string{},
	}
	output, exitCode := run(cfg)

	// Release the lock.
	windows.CloseHandle(handle)

	if !strings.Contains(output, `"locked_files": [`) {
		t.Errorf("Expected output to contain locked_files, got: %s", output)
	}
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got: %d", exitCode)
	}
}
