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

func parseFlags() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.FilePattern, "file-pattern", "*", "File pattern to search")
	flag.StringVar(&cfg.Path, "path", ".", "Directory to search")
	flag.StringVar(&cfg.HashType, "hash", hasher.HashMD5, "Hash type: MD5, SHA1, SHA256, XXHASH64, BLAKE3")
	flag.StringVar(&cfg.OutFile, "out-file", "", "File to store the results")
	flag.BoolVar(&cfg.Rename, "rename", false, "Rename files to their hash value")
	flag.BoolVar(&cfg.Display, "display", true, "Display hash values to the user")
	flag.BoolVar(&cfg.Version, "version", false, "Display version information")
	flag.IntVar(&cfg.NumWorkers, "workers", runtime.NumCPU(), "Number of worker goroutines")
	flag.Parse()
	return cfg
}

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
