# H5ai Downloader (Go Version)

A Go rewrite of the h5ai downloader with concurrent download support and additional features.

## Features

- **Concurrent Downloads**: Use multiple goroutines for parallel file downloads
- **Export Only Mode**: Save URLs to file instead of downloading
- **Flexible Output**: Control directory structure and output files
- **Caching**: HTTP response caching to avoid redundant requests
- **Progress Tracking**: Track download progress and resume interrupted downloads
- **Multiple URL Support**: Process single URLs or files containing multiple URLs

## Installation

```bash
# Build from source
go build -o h5ai_downloader

# Or run directly
go run main.go [options]
```

## Usage

### Basic Usage

```bash
# Download from a single URL to default directory (./files)
./h5ai_downloader -url "http://example.com/files/" -depth 3 -workers 8

# Download to custom directory
./h5ai_downloader -url "http://example.com/files/" -output "./downloads" -workers 4

# Download from multiple URLs in a file
./h5ai_downloader -file urls.txt -depth 2 -workers 4
```

### Export Only Mode

```bash
# Export URLs to default file (urls.txt)
./h5ai_downloader -url "http://example.com/files/" -export-only

# Export URLs to custom file with flat structure
./h5ai_downloader -url "http://example.com/files/" -export-only -flat -output my_urls.txt

# Export with directory structure preserved
./h5ai_downloader -url "http://example.com/files/" -export-only -output detailed_urls.txt
```

## Command Line Options

| Option          | Short | Description                                           | Default                                    |
| --------------- | ----- | ----------------------------------------------------- | ------------------------------------------ |
| `--url`         | `-u`  | Single URL to scrape                                  | -                                          |
| `--file`        | `-f`  | File containing URLs to scrape                        | -                                          |
| `--depth`       | `-d`  | Maximum depth for scraping                            | 4                                          |
| `--workers`     |       | Number of concurrent download workers                 | 4                                          |
| `--export-only` |       | Save URLs to file instead of downloading              | false                                      |
| `--flat`        |       | Skip directory structure                              | false                                      |
| `--output`      |       | Output directory for downloads OR filename for export | `./files` (download) / `urls.txt` (export) |

## Input File Format

When using the `--file` option, create a text file with one URL per line. Optionally specify custom depth:

```
http://example1.com/files/
http://example2.com/data/ 5
http://example3.com/docs/ 2
```

## Features Comparison

| Feature                  | Python Version | Go Version |
| ------------------------ | -------------- | ---------- |
| Basic h5ai crawling      | ✅             | ✅         |
| Download tracking        | ✅             | ✅         |
| HTTP caching             | ✅             | ✅         |
| Multiple URLs            | ✅             | ✅         |
| **Concurrent downloads** | ❌             | ✅         |
| **Export-only mode**     | ❌             | ✅         |
| **Flat export option**   | ❌             | ✅         |
| **Custom output file**   | ❌             | ✅         |
| **Worker pool control**  | ❌             | ✅         |

## Performance

The Go version offers significant performance improvements:

- **Concurrent Downloads**: Download multiple files simultaneously using configurable worker pools
- **Better Memory Usage**: More efficient memory management compared to Python
- **Faster Startup**: No interpreter overhead
- **Built-in HTTP Client**: Optimized HTTP handling without external dependencies

## Architecture

### Core Components

1. **Cache System**: Stores HTTP responses in `.gob` files for quick retrieval
2. **URL Collector**: Thread-safe collection of downloadable URLs during crawling
3. **Download Tracker**: Persistent tracking of completed downloads to enable resuming
4. **Worker Pool**: Configurable number of goroutines for concurrent downloads

### Directory Structure

```
├── main.go              # Main application code
├── go.mod              # Go module definition
├── url_cache/          # HTTP response cache (created automatically)
├── downloaded_db/      # Download completion tracking (created automatically)
└── [downloaded files]  # Downloaded content preserving directory structure
```

## Examples

### Example 1: Basic Download to Custom Directory

```bash
./h5ai_downloader -url "http://files.example.com/" -depth 2 -workers 8 -output "./my_downloads"
```

### Example 2: Export URLs Only

```bash
./h5ai_downloader -url "http://files.example.com/" -export-only -output backup_urls.txt
```

### Example 3: Multiple URLs with Different Depths

Create `sites.txt`:

```
http://site1.com/files/ 3
http://site2.com/data/ 5
http://site3.com/docs/
```

Run:

```bash
./h5ai_downloader -file sites.txt -workers 6 -output "./downloads"
```

### Example 4: Flat Export (URLs only, no directory info)

```bash
./h5ai_downloader -url "http://files.example.com/" -export-only -flat -output flat_urls.txt
```

## Notes

### Output Parameter Behavior

The `-output` parameter has dual functionality:

- **Download Mode** (default): Specifies the output directory where files will be downloaded
  - Default: `./files`
  - Example: `-output "./my_downloads"` creates directory structure under `my_downloads/`
- **Export Mode** (`-export-only`): Specifies the filename for the exported URL list
  - Default: `urls.txt`
  - Example: `-output "backup_urls.txt"` creates a file named `backup_urls.txt`

### Directory Structure

- When `flat=false` (default): Maintains the original directory structure from the server
- When `flat=true`: Downloads all files to the output directory root (no subdirectories)

- The Go version maintains compatibility with the Python version's cache and download tracking
- Default worker count is 4, but can be adjusted based on your system and network capacity
- Export-only mode is useful for creating backup lists or processing URLs with external tools
- The flat option in export mode outputs just the URLs without directory structure information
