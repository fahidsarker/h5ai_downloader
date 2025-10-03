package main

import (
	"bufio"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Config holds all command line configuration
type Config struct {
	URL        string
	File       string
	Depth      int
	Workers    int
	ExportOnly bool
	Flat       bool
	Output     string
}

// URLTask represents a URL to be processed with its depth
type URLTask struct {
	URL   string
	Depth int
}

// DownloadTask represents a file to be downloaded
type DownloadTask struct {
	URL          string
	Path         string
	TargetDomain string
	MajorURL     string
}

// Cache manages HTTP response caching
type Cache struct {
	dir string
	mu  sync.RWMutex
}

// DownloadTracker tracks completed downloads
type DownloadTracker struct {
	completed map[string]bool
	mu        sync.RWMutex
	dbPath    string
}

// URLCollector collects URLs during crawling
type URLCollector struct {
	urls []string
	mu   sync.Mutex
}

func main() {
	config := parseFlags()

	if err := validateConfig(config); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	tasks, err := getURLTasks(config)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(tasks) == 0 {
		fmt.Println("No URLs detected")
		os.Exit(1)
	}

	if len(tasks) > 1 {
		fmt.Printf("Detected %d URLs\n", len(tasks))
	}

	// Initialize cache
	cache := &Cache{dir: "url_cache"}
	if err := os.MkdirAll(cache.dir, 0755); err != nil {
		fmt.Printf("Error creating cache directory: %v\n", err)
		os.Exit(1)
	}

	// Crawl and collect URLs
	fmt.Println("\nScraping and finding download URLs:")
	allDownloadableURLs := make(map[string][]string)
	totalURLs := 0

	for i, task := range tasks {
		fmt.Printf("Processing %d/%d: %s\n", i+1, len(tasks), task.URL)
		targetDomain := getTargetDomain(task.URL)
		if targetDomain == "" {
			fmt.Printf("Invalid URL. Please enter with http:// or https://: %s\n", task.URL)
			os.Exit(1)
		}

		collector := &URLCollector{}
		crawlH5AI(cache, targetDomain, task.URL, 0, task.Depth, collector)

		allDownloadableURLs[task.URL] = collector.urls
		totalURLs += len(collector.urls)
	}

	if totalURLs == 0 {
		fmt.Println("No downloadable files found")
		os.Exit(1)
	}

	fmt.Printf("\nTotal Downloadable Files: %d\n", totalURLs)

	if config.ExportOnly {
		exportURLs(allDownloadableURLs, config)
	} else {
		// Ask for confirmation
		fmt.Print("Press y to continue: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		if strings.TrimSpace(response) != "y" {
			fmt.Println("Aborting...")
			os.Exit(1)
		}

		downloadFiles(allDownloadableURLs, config)
	}
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.URL, "u", "", "URL to scrape")
	flag.StringVar(&config.URL, "url", "", "URL to scrape")
	flag.StringVar(&config.File, "f", "", "File containing URLs to scrape")
	flag.StringVar(&config.File, "file", "", "File containing URLs to scrape")
	flag.IntVar(&config.Depth, "d", 4, "Maximum depth for scraping")
	flag.IntVar(&config.Depth, "depth", 4, "Maximum depth for scraping")
	flag.IntVar(&config.Workers, "workers", 4, "Number of concurrent download workers")
	flag.BoolVar(&config.ExportOnly, "export-only", false, "Save URLs to file instead of downloading")
	flag.BoolVar(&config.Flat, "flat", false, "Skip directory structure in export")
	flag.StringVar(&config.Output, "output", "", "Output directory for downloads or filename for export")

	flag.Parse()

	return config
}

func validateConfig(config *Config) error {
	if config.URL == "" && config.File == "" {
		return fmt.Errorf("either -url or -file must be specified")
	}

	if config.URL != "" && config.File != "" {
		return fmt.Errorf("cannot specify both -url and -file")
	}

	if config.Workers < 1 {
		return fmt.Errorf("workers must be at least 1")
	}

	if config.Depth < 0 {
		return fmt.Errorf("depth must be non-negative")
	}

	// Set default values based on mode
	if config.Output == "" {
		if config.ExportOnly {
			config.Output = "urls.txt"
		} else {
			config.Output = "./files"
		}
	}

	return nil
}

func getURLTasks(config *Config) ([]URLTask, error) {
	if config.URL != "" {
		return []URLTask{{URL: config.URL, Depth: config.Depth}}, nil
	}

	return getURLsFromFile(config.File, config.Depth)
}

func getURLsFromFile(filePath string, defaultDepth int) ([]URLTask, error) {
	if !strings.HasSuffix(filePath, ".txt") {
		return nil, fmt.Errorf("invalid file format: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}
	defer file.Close()

	var tasks []URLTask
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) > 1 {
			depth, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid depth in line: %s", line)
			}
			tasks = append(tasks, URLTask{URL: parts[0], Depth: depth})
		} else {
			tasks = append(tasks, URLTask{URL: parts[0], Depth: defaultDepth})
		}
	}

	return tasks, scanner.Err()
}

func urlToFileName(url string) string {
	url = strings.ReplaceAll(url, "http://", "")
	url = strings.ReplaceAll(url, "https://", "")
	url = strings.ReplaceAll(url, "/", "_")
	return url
}

func getTargetDomain(urlStr string) string {
	re := regexp.MustCompile(`(https?://[a-zA-Z0-9.-]+)`)
	matches := re.FindStringSubmatch(urlStr)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (c *Cache) getSourceUsingHTTP(urlStr string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fileName := urlToFileName(urlStr) + ".gob"
	filePath := filepath.Join(c.dir, fileName)

	// Check cache
	if data, err := os.ReadFile(filePath); err == nil {
		var result []byte
		if err := gob.NewDecoder(strings.NewReader(string(data))).Decode(&result); err == nil {
			return result, nil
		}
	}

	// Download
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Cache result
	var buf strings.Builder
	if err := gob.NewEncoder(&buf).Encode(data); err == nil {
		os.WriteFile(filePath, []byte(buf.String()), 0644)
	}

	return data, nil
}

func crawlH5AI(cache *Cache, targetDomain, urlStr string, recursion, maxDepth int, collector *URLCollector) {
	if recursion > maxDepth {
		return
	}

	data, err := cache.getSourceUsingHTTP(urlStr)
	if err != nil {
		return
	}

	// Simple HTML parsing using regex to find href attributes
	hrefRegex := regexp.MustCompile(`href="([^"]*)"`)
	matches := hrefRegex.FindAllStringSubmatch(string(data), -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		href := match[1]
		if strings.HasPrefix(href, "..") {
			continue
		}

		if strings.HasSuffix(href, "/") {
			// Directory - recurse
			newURL := targetDomain + href
			crawlH5AI(cache, targetDomain, newURL, recursion+1, maxDepth, collector)
		} else {
			// File - add to download list
			fileURL := targetDomain + href
			collector.mu.Lock()
			collector.urls = append(collector.urls, fileURL)
			collector.mu.Unlock()
		}
	}
}

func downloadURLToPath(targetDomain, urlStr, outputDir string, flat bool) string {
	pathStr := strings.TrimPrefix(urlStr, targetDomain)
	if flat {
		pathStr = path.Base(pathStr)
	} else {
		pathStr = strings.TrimPrefix(pathStr, "/")
	}

	// Combine output directory with path
	fullPath := filepath.Join(outputDir, pathStr)

	decoded, err := url.QueryUnescape(fullPath)
	if err != nil {
		return fullPath
	}
	return decoded
}

func exportURLs(allURLs map[string][]string, config *Config) {
	fmt.Printf("Exporting URLs to %s...\n", config.Output)

	file, err := os.Create(config.Output)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for majorURL, urls := range allURLs {
		targetDomain := getTargetDomain(majorURL)

		for _, urlStr := range urls {
			if config.Flat {
				writer.WriteString(urlStr + "\n")
			} else {
				// For export, we don't use the output directory as prefix, just show the structure
				pathStr := strings.TrimPrefix(urlStr, targetDomain)
				pathStr = strings.TrimPrefix(pathStr, "/")
				decoded, err := url.QueryUnescape(pathStr)
				if err != nil {
					decoded = pathStr
				}
				writer.WriteString(fmt.Sprintf("%s -> %s\n", urlStr, decoded))
			}
		}
	}

	fmt.Printf("Successfully exported %d URLs\n", getTotalURLCount(allURLs))
}

func downloadFiles(allURLs map[string][]string, config *Config) {
	for majorURL, urls := range allURLs {
		targetDomain := getTargetDomain(majorURL)
		tracker := newDownloadTracker(majorURL)
		tracker.load()

		// Create download tasks
		var tasks []DownloadTask
		for _, urlStr := range urls {
			pathStr := downloadURLToPath(targetDomain, urlStr, config.Output, config.Flat)

			if tracker.isCompleted(urlStr) && fileExists(pathStr) {
				continue
			}

			tasks = append(tasks, DownloadTask{
				URL:          urlStr,
				Path:         pathStr,
				TargetDomain: targetDomain,
				MajorURL:     majorURL,
			})
		}

		if len(tasks) == 0 {
			fmt.Println("All files already downloaded")
			continue
		}

		fmt.Printf("Downloading %d files with %d workers...\n", len(tasks), config.Workers)
		downloadWithWorkers(tasks, tracker, config.Workers)
	}
}

func downloadWithWorkers(tasks []DownloadTask, tracker *DownloadTracker, numWorkers int) {
	taskChan := make(chan DownloadTask, len(tasks))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go downloadWorker(taskChan, tracker, &wg)
	}

	// Send tasks
	for _, task := range tasks {
		taskChan <- task
	}
	close(taskChan)

	wg.Wait()
}

func downloadWorker(taskChan <-chan DownloadTask, tracker *DownloadTracker, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range taskChan {
		if err := downloadFile(task); err != nil {
			fmt.Printf("Error downloading %s: %v\n", task.URL, err)
			continue
		}

		tracker.markCompleted(task.MajorURL, task.URL)
		fmt.Printf("Downloaded: %s\n", task.Path)
	}
}

func downloadFile(task DownloadTask) error {
	// Create directory if needed
	dir := filepath.Dir(task.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Download file
	resp, err := http.Get(task.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(task.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func newDownloadTracker(majorURL string) *DownloadTracker {
	dbDir := "downloaded_db"
	os.MkdirAll(dbDir, 0755)

	return &DownloadTracker{
		completed: make(map[string]bool),
		dbPath:    filepath.Join(dbDir, urlToFileName(majorURL)+".gob"),
	}
}

func (dt *DownloadTracker) load() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	data, err := os.ReadFile(dt.dbPath)
	if err != nil {
		return
	}

	var urls []string
	if err := gob.NewDecoder(strings.NewReader(string(data))).Decode(&urls); err != nil {
		return
	}

	for _, url := range urls {
		dt.completed[url] = true
	}
}

func (dt *DownloadTracker) save() {
	dt.mu.RLock()
	urls := make([]string, 0, len(dt.completed))
	for url := range dt.completed {
		urls = append(urls, url)
	}
	dt.mu.RUnlock()

	var buf strings.Builder
	if err := gob.NewEncoder(&buf).Encode(urls); err != nil {
		return
	}

	os.WriteFile(dt.dbPath, []byte(buf.String()), 0644)
}

func (dt *DownloadTracker) markCompleted(majorURL, url string) {
	dt.mu.Lock()
	dt.completed[url] = true
	dt.mu.Unlock()
	dt.save()
}

func (dt *DownloadTracker) isCompleted(url string) bool {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.completed[url]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getTotalURLCount(allURLs map[string][]string) int {
	total := 0
	for _, urls := range allURLs {
		total += len(urls)
	}
	return total
}
