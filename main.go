package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/zeebo/blake3"
)

var Version string

type hashFunc func(data []byte) string

// Hash functions
func hashMD5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

func hashSHA1(data []byte) string {
	hash := sha1.Sum(data)
	return hex.EncodeToString(hash[:])
}

func hashSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hashXXHash(data []byte) string {
	hash := xxhash.Sum64(data)
	return fmt.Sprintf("%x", hash)
}

func hashBlake3(data []byte) string {
	hash := blake3.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func main() {
	// Command-line flags
	filePattern := flag.String("file-pattern", "*", "File pattern to search")
	path := flag.String("path", ".", "Directory to search")
	hashType := flag.String("hash", "MD5", "Hash type: MD5, SHA1, SHA256, XXHASH64, BLAKE3")
	outFile := flag.String("out-file", "", "File to store the results")
	rename := flag.Bool("rename", false, "Rename files to their hash value")
	display := flag.Bool("display", true, "Display hash values to the user")
	versionFlag := flag.Bool("version", false, "Display version information")

	flag.Parse()

	// Version
	if *versionFlag {
		fmt.Printf("Application Version: %s\n", Version)
		os.Exit(0)
	}

	// Select hash function
	var hasher hashFunc
	switch *hashType {
	case "MD5":
		hasher = hashMD5
	case "SHA1":
		hasher = hashSHA1
	case "SHA256":
		hasher = hashSHA256
	case "XXHASH64":
		hasher = hashXXHash
	case "BLAKE3":
		hasher = hashBlake3
	default:
		fmt.Printf("Unsupported hash type: %s\n", *hashType)
		os.Exit(1)
	}

	// Channel to collect results
	results := make(chan [2]string)
	var wg sync.WaitGroup

	// Walk the directory recursively
	wg.Add(1)
	go func() {
		defer wg.Done()
		filepath.Walk(*path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("Error accessing file %s: %v\n", path, err)
				return nil
			}

			if !info.IsDir() {
				match, _ := filepath.Match(*filePattern, info.Name())
				if match {
					wg.Add(1)
					go func(filePath string) {
						defer wg.Done()

						file, err := os.Open(filePath)
						if err != nil {
							fmt.Printf("Error opening file %s: %v\n", filePath, err)
							return
						}
						defer file.Close()

						data, err := io.ReadAll(file)
						if err != nil {
							fmt.Printf("Error reading file %s: %v\n", filePath, err)
							return
						}

						hash := hasher(data)
						results <- [2]string{filePath, hash}
					}(path)
				}
			}
			return nil
		})
	}()

	// Close results channel after processing
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	output := make(map[string]string)
	for result := range results {
		filePath, hash := result[0], result[1]
		output[filePath] = hash

		if *rename {
			newPath := filepath.Join(filepath.Dir(filePath), hash+filepath.Ext(filePath))
			os.Rename(filePath, newPath)
		}

		if *display && *outFile == "" {
			fmt.Printf("%s: %s\n", filePath, hash)
		}
	}

	// Write results to a file if specified
	if *outFile != "" {
		file, err := os.Create(*outFile)
		if err != nil {
			fmt.Printf("Error creating output file: %v\n", err)
			return
		}
		defer file.Close()

		for filePath, hash := range output {
			file.WriteString(fmt.Sprintf("%s: %s\n", filePath, hash))
		}
	}
}
