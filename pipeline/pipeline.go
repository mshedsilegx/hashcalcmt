// Package pipeline implements a multi-threaded file processing pipeline.
// It scans directories, distributes file paths to worker goroutines,
// and collects hashing results concurrently.
package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"criticalsys.net/hashcalcmt/hasher"
)

// Result represents a single file hashing result.
type Result struct {
	FilePath string
	Hash     string
	Error    error
}

// Run starts the file processing pipeline.
// It performs the following steps:
// 1. Opens the target path as an os.Root to prevent directory traversal.
// 2. Starts a pool of worker goroutines.
// 3. Walks the directory tree and sends matching file paths to the workers.
// 4. Closes all resources and channels once processing is complete.
// It returns a read-only channel of Result objects.
func Run(path, filePattern string, numWorkers int, hf hasher.Func) <-chan Result {
	results := make(chan Result)
	jobs := make(chan string)
	var wg sync.WaitGroup

	root, err := os.OpenRoot(path)
	if err != nil {
		results <- Result{Error: fmt.Errorf("error opening root %s: %w", path, err)}
		close(results)
		return results
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(&wg, root, jobs, results, hf)
	}

	// Walk the directory and send jobs.
	go func() {
		defer close(jobs)
		if err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				results <- Result{FilePath: p, Error: err}
				return nil
			}

			if !info.IsDir() {
				if match, _ := filepath.Match(filePattern, info.Name()); match {
					// jobs channel expects path relative to root for os.Root access
					rel, err := filepath.Rel(path, p)
					if err != nil {
						results <- Result{FilePath: p, Error: err}
						return nil
					}
					jobs <- rel
				}
			}
			return nil
		}); err != nil {
			results <- Result{Error: fmt.Errorf("error walking path %s: %w", path, err)}
		}
	}()

	// Wait for all workers to finish, then close results channel and root.
	go func() {
		wg.Wait()
		_ = root.Close() // #nosec G104 -- closing root at the end of processing, error is secondary to completion
		close(results)
	}()

	return results
}

// worker is a goroutine that processes jobs from the jobs channel.
// It uses the provided os.Root to safely open files and the hasher.Func to compute hashes.
// Results are sent to the results channel.
func worker(wg *sync.WaitGroup, root *os.Root, jobs <-chan string, results chan<- Result, hf hasher.Func) {
	defer wg.Done()
	for filePath := range jobs {
		hash, err := hashFile(root, filePath, hf)
		results <- Result{FilePath: filePath, Hash: hash, Error: err}
	}
}

// hashFile opens a file safely via the os.Root and computes its hash.
// It ensures the file is closed correctly and handles any errors during the process.
func hashFile(root *os.Root, filePath string, hf hasher.Func) (hash string, err error) {
	file, err := root.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("could not open file: %w", err)
	}
	defer func() {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}()

	return hf(file)
}
