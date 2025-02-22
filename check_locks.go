package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// LockInfo struct to store locked files/folders
type LockInfo struct {
	LockedFiles   []string `json:"locked_files"`
	LockedFolders []string `json:"locked_folders"`
}

type Config struct {
	Root    string
	Exclude []string
}

var exitFunc = os.Exit
var version = "dev"

// Check if a file is locked by trying to open it in exclusive mode
func isFileLocked(filePath string) bool {
	ptr, err := windows.UTF16PtrFromString(filePath)
	if err != nil {
		return true // Assume locked if we can't convert string
	}

	// Try to open the file with no sharing (exclusive lock)
	handle, err := windows.CreateFile(
		ptr,
		windows.GENERIC_READ|windows.GENERIC_WRITE, // Open for read/write
		0, // No sharing allowed (exclusive lock)
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)

	// If opening fails, the file is locked
	if err != nil {
		return true
	}

	// If we got a valid handle, close it and return false (not locked)
	windows.CloseHandle(handle)
	return false
}

// Check if a folder is locked by attempting to rename a subfolder inside it
func isFolderLocked(folderPath string) bool {
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return true // If we can't read the folder, assume it's locked
	}

	// Find the first subfolder inside the target folder
	for _, file := range files {
		if file.IsDir() {
			subFolder := filepath.Join(folderPath, file.Name())
			testPath := subFolder + "_locktest"

			// Try renaming the subfolder
			err := os.Rename(subFolder, testPath)
			if err != nil {
				return true // Subfolder rename failed → folder is locked
			}

			// Rename it back
			err = os.Rename(testPath, subFolder)
			if err != nil {
				fmt.Println("Warning: Unable to restore subfolder name:", err)
			}
			return false // Folder is not locked
		}
	}

	// If no subfolders exist, fall back to a temporary file check
	tempFile := filepath.Join(folderPath, "locktest.tmp")
	file, err := os.Create(tempFile)
	if err != nil {
		return true // Cannot create a file → folder is locked
	}
	file.Close()
	os.Remove(tempFile)

	return false // Folder is not locked
}

// Convert relative exclude paths to absolute paths
func getAbsoluteExclusions(rootPath string, excluded []string) []string {
	var absExcluded []string
	for _, relPath := range excluded {
		absPath := filepath.Join(rootPath, relPath)
		absExcluded = append(absExcluded, absPath)
	}
	return absExcluded
}

// Print usage instructions
func printHelp() {
	fmt.Println(`
        Usage: check_locks.exe -root <folder> [-exclude <relative_folder1,relative_folder2>]

        Options:
        -root      Specify the root folder to scan for locks.
        -exclude   Comma-separated list of subfolders to exclude (relative to root).
        -help      Display this help message.

        Examples:
        check_locks.exe -root "D:\projects\infoscanjs"
        check_locks.exe -root "D:\projects\infoscanjs" -exclude "logs,temp"
        `)
	exitFunc(0)
}

// checkLocks walks the directory tree starting at rootPath, skipping excluded paths.
// Returns a JSON string and an exit code (1 if a lock is found, 0 otherwise).
func checkLocks(rootPath string, excludedPaths []string) (string, int) {
	var output string
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip excluded paths
		for _, exclude := range excludedPaths {
			if strings.HasPrefix(path, exclude) {
				return nil
			}
		}

		// Check if the folder itself is locked
		if info.IsDir() && isFolderLocked(path) {
			result := LockInfo{LockedFiles: []string{}, LockedFolders: []string{path}}
			jsonOutput, _ := json.MarshalIndent(result, "", "  ")
			output = string(jsonOutput)
			// Return an error to break out of Walk.
			return fmt.Errorf("lock found")
		}

		// Check if the file is locked
		if !info.IsDir() && isFileLocked(path) {
			result := LockInfo{LockedFiles: []string{path}, LockedFolders: []string{}}
			jsonOutput, _ := json.MarshalIndent(result, "", "  ")
			output = string(jsonOutput)
			return fmt.Errorf("lock found")
		}

		return nil
	})

	if err != nil {
		return output, 1
	}

	// If no locks were found, return an empty result
	return `{"locked_files": [], "locked_folders": []}`, 0
}

func run(cfg Config) (string, int) {
	// Convert relative exclude paths to absolute paths
	excludedPaths := getAbsoluteExclusions(cfg.Root, cfg.Exclude)

	return checkLocks(cfg.Root, excludedPaths)
}

func main() {
	// Define command-line arguments
	rootFolder := flag.String("root", "", "Root folder to scan")
	excludeList := flag.String("exclude", "", "Comma-separated list of relative folders to exclude")
	helpFlag := flag.Bool("help", false, "Display help information")
	versionFlag := flag.Bool("version", false, "Print the version of the program")

	// Parse command-line arguments
	flag.Parse()

	// Show help if -help is used
	if *helpFlag {
		printHelp()
	}

	if *versionFlag {
		fmt.Println("check_locks version:", version)
		return
	}

	// Validate required arguments
	if *rootFolder == "" {
		fmt.Println(`Error: You must specify a root folder using -root`)
		printHelp()
		exitFunc(1)
	}

	// Convert relative exclude paths to absolute paths
	var excludedPaths []string
	if *excludeList != "" {
		relativeExcludes := strings.Split(*excludeList, ",")
		for i, path := range relativeExcludes {
			relativeExcludes[i] = strings.TrimSpace(path)
		}
		excludedPaths = getAbsoluteExclusions(*rootFolder, relativeExcludes)
	}

	// Create a config instance and run the main logic.
	cfg := Config{
		Root:    *rootFolder,
		Exclude: excludedPaths,
	}
	output, exitCode := run(cfg)
	fmt.Println(output)
	exitFunc(exitCode)
}
