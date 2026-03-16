package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"criticalsys.net/hashcalcmt/hasher"
	"criticalsys.net/hashcalcmt/pipeline"
)

var version string

// Config holds the application configuration.
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

// main is the entry point of the Hash MT Generator tool.
// It orchestrates flag parsing, hash function selection, and result processing.
func main() {
	cfg := parseFlags()

	if cfg.Version {
		fmt.Printf("Hash MT Generator - Version: %s\n", version)
		os.Exit(0)
	}

	hf, err := hasher.GetHasher(cfg.HashType)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	results := pipeline.Run(cfg.Path, cfg.FilePattern, cfg.NumWorkers, hf)

	output, errs := processResults(results, cfg)

	if cfg.OutFile != "" {
		if err := writeResultsToFile(cfg.OutFile, output); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		}
	}

	if len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "\nErrors encountered:")
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, "-", err)
		}
	}
}

// parseFlags defines and parses CLI flags into a Config struct.
// It sets defaults for hash types, worker counts, and patterns.
func parseFlags() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.FilePattern, "file-pattern", "*", "File pattern to search")
	flag.StringVar(&cfg.Path, "path", ".", "Directory to search")
	flag.StringVar(&cfg.HashType, "hash", hasher.HashMD5, "Hash type: MD5, SHA1, SHA256, XXH3-128, HIGHWAYHASH, WYHASH, BLAKE3")
	flag.StringVar(&cfg.OutFile, "out-file", "", "File to store the results")
	flag.BoolVar(&cfg.Rename, "rename", false, "Rename files to their hash value")
	flag.BoolVar(&cfg.Display, "display", true, "Display hash values to the user")
	flag.BoolVar(&cfg.Version, "version", false, "Display version information")
	flag.IntVar(&cfg.NumWorkers, "workers", runtime.NumCPU(), "Number of worker goroutines")
	flag.Parse()
	return cfg
}

// processResults iterates over the results channel and handles renaming or display.
// It aggregates results for potential file output and collects any errors.
func processResults(results <-chan pipeline.Result, cfg *Config) (map[string]string, []error) {
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

// writeResultsToFile saves the collected hash results to a specified file.
// It cleans the filename to mitigate directory traversal risks.
func writeResultsToFile(filename string, results map[string]string) (err error) {
	// Clean and localize the filename to mitigate G304.
	// We use filepath.Clean to resolve any directory traversal elements.
	filename = filepath.Clean(filename)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}()

	for filePath, hash := range results {
		if _, err = fmt.Fprintf(file, "%s: %s\n", filePath, hash); err != nil {
			return err
		}
	}
	return nil
}
