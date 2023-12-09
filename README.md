# h5ai Downloader
## Download contents from a h5ai website with deep scraping and crawling
### Run -
- `python dl.py <url> <max_depth default=4>`

The crawler will search (including sib dir) to find the downloadable URLs and confirm before starting to download.
Features:
- Depth of recursion Control
- Url caching
- Download status tracking
- - If the download is cancelled, it will skip the downloaded files when re-run
