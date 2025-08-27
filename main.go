// This program is a command-line tool for hashing files concurrently.
// It can recursively scan a directory, filter files by a pattern, and compute hashes
// using various algorithms (MD5, SHA1, SHA256, XXHASH64, BLAKE3).
// The tool uses a worker pool to process files in parallel for efficiency.
package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/zeebo/blake3"
)

// version is set at build time.
var version string

// Hash types constants.
const (
	HashMD5    = "MD5"
	HashSHA1   = "SHA1"
	HashSHA256 = "SHA256"
	HashXXHash = "XXHASH64"
	HashBlake3 = "BLAKE3"
)

// hashFunc is a function type that takes a reader and returns a hash string or an error.
type hashFunc func(io.Reader) (string, error)

// Config holds the application configuration provided via command-line flags.
type Config struct {
	FilePattern string
	Path        string
	HashType    string
	OutFile     string
	Rename      bool
	Display     bool
	Version     bool
	NumWorkers  int
}

// Result represents a single file hashing result, including any error that occurred.
type Result struct {
	FilePath string
	Hash     string
	Error    error
}

// main is the entry point of the application.
// It parses flags, sets up the processing pipeline, and handles results.
func main() {
	cfg := parseFlags()

	if cfg.Version {
		fmt.Printf("Hash MT Generator - Version: %s\n", version)
		os.Exit(0)
	}

	hasher, err := getHasher(cfg.HashType)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// jobs channel to send file paths from the walker to the workers.
	jobs := make(chan string)
	// results channel to send hashing results from the workers to the main goroutine.
	results := make(chan Result)
	var wg sync.WaitGroup

	// Start a pool of worker goroutines.
	for i := 0; i < cfg.NumWorkers; i++ {
		wg.Add(1)
		go worker(&wg, jobs, results, hasher)
	}

	// Start a goroutine to walk the directory and send file paths to the jobs channel.
	go func() {
		filepath.Walk(cfg.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				results <- Result{FilePath: path, Error: err}
				return nil
			}

			if !info.IsDir() {
				if match, _ := filepath.Match(cfg.FilePattern, info.Name()); match {
					jobs <- path
				}
			}
			return nil
		})
		close(jobs) // Close the jobs channel to signal that no more jobs will be sent.
	}()

	// Start a goroutine to wait for all workers to finish and then close the results channel.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process the results from the results channel.
	output, errs := processResults(results, cfg)

	// Write results to a file if specified.
	if cfg.OutFile != "" {
		if err := writeResultsToFile(cfg.OutFile, output); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		}
	}

	// Report any errors encountered during processing.
	if len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "\nErrors encountered:")
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, "-", err)
		}
	}
}

// worker is a goroutine that receives file paths from the jobs channel,
// hashes the files, and sends the results to the results channel.
func worker(wg *sync.WaitGroup, jobs <-chan string, results chan<- Result, hasher hashFunc) {
	defer wg.Done()
	for filePath := range jobs {
		hash, err := hashFile(filePath, hasher)
		results <- Result{FilePath: filePath, Hash: hash, Error: err}
	}
}

// parseFlags parses command-line flags and returns a Config struct.
func parseFlags() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.FilePattern, "file-pattern", "*", "File pattern to search")
	flag.StringVar(&cfg.Path, "path", ".", "Directory to search")
	flag.StringVar(&cfg.HashType, "hash", HashMD5, "Hash type: MD5, SHA1, SHA256, XXHASH64, BLAKE3")
	flag.StringVar(&cfg.OutFile, "out-file", "", "File to store the results")
	flag.BoolVar(&cfg.Rename, "rename", false, "Rename files to their hash value")
	flag.BoolVar(&cfg.Display, "display", true, "Display hash values to the user")
	flag.BoolVar(&cfg.Version, "version", false, "Display version information")
	flag.IntVar(&cfg.NumWorkers, "workers", runtime.NumCPU(), "Number of worker goroutines")
	flag.Parse()
	return cfg
}

// getHasher returns the appropriate hash function based on the hash type string.
func getHasher(hashType string) (hashFunc, error) {
	switch hashType {
	case HashMD5:
		return newHashStreamFunc(md5.New), nil
	case HashSHA1:
		return newHashStreamFunc(sha1.New), nil
	case HashSHA256:
		return newHashStreamFunc(sha256.New), nil
	case HashXXHash:
		return hashXXHashStream, nil
	case HashBlake3:
		return newHashStreamFunc(func() hash.Hash { return blake3.New() }), nil
	default:
		return nil, fmt.Errorf("unsupported hash type: %s", hashType)
	}
}

// newHashStreamFunc creates a hashFunc from a function that returns a new hash.Hash.
// This pattern ensures that a new hash object is created for each file.
func newHashStreamFunc(newHasher func() hash.Hash) hashFunc {
	return func(r io.Reader) (string, error) {
		h := newHasher()
		if _, err := io.Copy(h, r); err != nil {
			return "", err
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	}
}

// hashXXHashStream creates a new xxhash.Digest and computes the hash.
// It's a special case because the xxhash library has a slightly different API.
func hashXXHashStream(r io.Reader) (string, error) {
	h := xxhash.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum64()), nil
}

// hashFile opens a file and computes its hash using the provided hasher function.
func hashFile(filePath string, hasher hashFunc) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return hasher(file)
}

// processResults consumes results from the results channel, handles file renaming,
// displays output, and collects errors.
func processResults(results <-chan Result, cfg *Config) (map[string]string, []error) {
	output := make(map[string]string)
	var errs []error

	for result := range results {
		if result.Error != nil {
			errs = append(errs, fmt.Errorf("error processing file %s: %w", result.FilePath, result.Error))
			continue
		}

		output[result.FilePath] = result.Hash

		if cfg.Rename {
			newPath := filepath.Join(filepath.Dir(result.FilePath), result.Hash+filepath.Ext(result.FilePath))
			if _, err := os.Stat(newPath); err == nil {
				errs = append(errs, fmt.Errorf("could not rename %s to %s: file already exists", result.FilePath, newPath))
				continue
			}
			if err := os.Rename(result.FilePath, newPath); err != nil {
				errs = append(errs, fmt.Errorf("error renaming file %s: %w", result.FilePath, err))
			}
		}

		if cfg.Display && cfg.OutFile == "" {
			fmt.Printf("%s: %s\n", result.FilePath, result.Hash)
		}
	}
	return output, errs
}

// writeResultsToFile writes the computed hashes to a file.
func writeResultsToFile(filename string, results map[string]string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for filePath, hash := range results {
		if _, err := file.WriteString(fmt.Sprintf("%s: %s\n", filePath, hash)); err != nil {
			return err
		}
	}
	return nil
}
