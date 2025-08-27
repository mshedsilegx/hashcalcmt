# Hash MT Generator

## Overview

Hash MT Generator is a high-performance, command-line tool for concurrently hashing files. It can recursively scan a directory, filter files by a pattern, and compute hashes using a variety of fast and secure hashing algorithms. The tool is designed for efficiency, using a worker pool to process files in parallel and a streaming approach to handle large files with low memory usage.

## Features

- **Concurrent Processing**: Utilizes a worker pool to hash multiple files in parallel, significantly speeding up the process on multi-core systems.
- **Multiple Hash Algorithms**: Supports a range of hashing algorithms, including MD5, SHA1, SHA256, XXHASH64, and BLAKE3.
- **Memory Efficient**: Uses a streaming approach to hash files, which means it can handle very large files without consuming a large amount of memory.
- **Flexible File Discovery**: Can recursively search directories and filter files based on a specified pattern.
- **Multiple Output Options**:
    - Display hash values directly to the console.
    - Store the results in an output file.
    - Rename files to their corresponding hash values.
- **Configurable Concurrency**: The number of concurrent workers can be configured to optimize performance for your specific hardware.

## Command Line Usage

The tool is configured via command-line flags:

| Flag             | Description                                              | Default Value      |
|------------------|----------------------------------------------------------|--------------------|
| `--file-pattern` | File pattern to search for.                              | `*` (all files)    |
| `--path`         | The directory to search in.                              | `.` (current dir)  |
| `--hash`         | The hash algorithm to use. (MD5, SHA1, SHA256, XXHASH64, BLAKE3) | `MD5`              |
| `--out-file`     | The file to store the results in.                        | (none)             |
| `--rename`       | Rename files to their hash value.                        | `false`            |
| `--display`      | Display hash values to the user.                         | `true`             |
| `--workers`      | The number of worker goroutines to use.                  | (number of CPUs)   |
| `--version`      | Display the version information.                         | `false`            |

## Examples

### Basic Usage

To compute the MD5 hashes of all files in the current directory and its subdirectories:

```bash
./hash-tool
```

### Using a Different Hash Algorithm

To compute SHA256 hashes for all `.jpg` files in the `/home/user/pictures` directory:

```bash
./hash-tool --hash=SHA256 --path=/home/user/pictures --file-pattern="*.jpg"
```

### Saving Results to a File

To compute BLAKE3 hashes for all files and save the results to a file named `hashes.txt`:

```bash
./hash-tool --hash=BLAKE3 --out-file=hashes.txt --display=false
```

### Renaming Files to Their Hashes

To rename all `.txt` files in the `documents` directory to their XXHASH64 hash values (the original file extension is preserved):

```bash
./hash-tool --hash=XXHASH64 --path=documents --file-pattern="*.txt" --rename --display=false
```
