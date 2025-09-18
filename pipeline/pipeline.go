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
// It walks the directory, starts a pool of workers, and returns a channel of results.
func Run(path, filePattern string, numWorkers int, hf hasher.Func) <-chan Result {
	results := make(chan Result)
	jobs := make(chan string)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(&wg, jobs, results, hf)
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
					jobs <- p
				}
			}
			return nil
		}); err != nil {
			results <- Result{Error: fmt.Errorf("error walking path %s: %w", path, err)}
		}
	}()

	// Wait for all workers to finish, then close results channel.
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// worker is a goroutine that receives file paths from the jobs channel,
// hashes the files, and sends the results to the results channel.
func worker(wg *sync.WaitGroup, jobs <-chan string, results chan<- Result, hf hasher.Func) {
	defer wg.Done()
	for filePath := range jobs {
		hash, err := hashFile(filePath, hf)
		results <- Result{FilePath: filePath, Hash: hash, Error: err}
	}
}

// hashFile opens a file and computes its hash using the provided hasher function.
func hashFile(filePath string, hf hasher.Func) (hash string, err error) {
	file, err := os.Open(filePath)
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
