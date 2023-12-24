# h5ai Downloader
## Download contents from a h5ai website with deep scraping and crawling
### Run -
- install dependency `pip install -r requirements.txt`
- `usage: python dl.py [-h] (-u URL | -f FILE) [-d DEPTH]`
- url can be a h5ai directory url or a txt file which contains multiple urls
- format of txt file:
```
<url> <optional depth>
<url> ...
...
```


The crawler will search (including sib dir) to find the downloadable URLs and confirm before starting to download.
Features:
- Download any files from the websiite
- Depth of recursion Control
- Url caching
- Download status tracking
- - If the download is cancelled, it will skip the downloaded files when re-run
