import os

def url_to_file_name(url):
    return url.replace('http://', '').replace('https://', '').replace('/', '_')

import pickle
def get_source_using_curl(url):
    file_name = url_to_file_name(url)+'.pkl'
    file_path = os.path.join('url_cache', file_name)
    if os.path.exists(file_path):
        # print('Using cached file: {}'.format(file_path))
        with open(file_path, 'rb') as f:
            return pickle.load(f)
    # if False:
    #     pass
    else:
        if not os.path.exists('url_cache'):
            os.mkdir('url_cache')
        # print('Downloading: {}'.format(url))
        try:
            import subprocess
            html = subprocess.check_output(['curl', url])
        except:
            html = ''
        with open(file_path, 'wb') as f:
            pickle.dump(html, f)
        return html
        

download_completed = []
def load_downloaded_urls(major_url):
    global download_completed

    db_path = os.path.join('./downloaded_db', url_to_file_name(major_url)+'.pkl')
    if os.path.exists(db_path):
        with open(db_path, 'rb') as f:
            download_completed += pickle.load(f)

def download_complete(major_url, url):
    global download_completed
    download_completed.append(url)
    if not os.path.exists('./downloaded_db'):
        os.mkdir('./downloaded_db')
    db_path = os.path.join('./downloaded_db', url_to_file_name(major_url)+'.pkl')
    with open(db_path, 'wb') as f:
        pickle.dump(download_completed, f)

# downloadable_urls = []

def get_target_domain(url):
    import re
    match = re.search(r'(http[s]?://[a-zA-Z0-9.-]+)', url)

    if match:
        return match.group(1)
    return None

def crawl_h5ai(target_domain, url, recursion, max_depth):
    downloadable_urls = []
    def inner_crawl(target_domain, url, recursion, max_depth):
        if recursion > max_depth:
            return
        html = get_source_using_curl(url)
        from bs4 import BeautifulSoup
        soup = BeautifulSoup(html, 'html.parser')
        
        for link in soup.find_all('a'):
            href = link.get('href')
            if href.startswith('..'):
                continue
            if href.endswith('/'):
                url = target_domain + href
                inner_crawl(target_domain, url, recursion+1, max_depth)
            else:
                url = target_domain + href
                downloadable_urls.append(url)
    inner_crawl(target_domain, url, recursion, max_depth)
    return downloadable_urls

def url_decode(url):
    import urllib.parse
    return urllib.parse.unquote(url)

def download_url_to_path(target_domain, url):
    path = url.replace(target_domain, '.')
    path = url_decode(path)
    
    return path

def download_urls(target_domain, major_url, urls):
    for url in urls:
        path = download_url_to_path(target_domain, url)
        directory = os.path.dirname(path)
        if not os.path.exists(directory):
            os.makedirs(directory)
        if os.path.exists(path) and url in download_completed:
            print('Skipping: {}'.format(path))
            continue
        print('Downloading: {}'.format(path))
        import subprocess
        subprocess.call(['wget', url, '-O', path, '-q', '--show-progress'])  
        # wget.download(url, out=path)
        download_complete(major_url, url)
        
# def get_downloaded_count(target_domain, major_url, urls):
#     count = 0
#     for url in urls:
#         load_downloaded_urls(major_url)
#         path = download_url_to_path(target_domain, url)
#         if os.path.exists(path) and url in download_completed:
#             count += 1
#     return count


def get_urls_from_file(path, default_depth):
    # is path is to a txt file, read the urls from the file
    if path.endswith('.txt'):
        if not os.path.exists(path):
            print('>>>> File not found: {}'.format(path))
            sys.exit(1)
        with open(path, 'r') as f:
            lines = f.read().splitlines()
            segments = []
            for line in lines:
                splitted = line.split(' ')
                if len(splitted) > 1:
                    segments.append((splitted[0], int(splitted[1])))
                else:
                    segments.append((splitted[0], default_depth))
            return segments
    
    # return [(path, default_depth)]
    print('>>>> Invalid file format: {}'.format(path))
    sys.exit(1)

import argparse
import sys
if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Scrapper for h5ai')
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('-u', '--url', type=str, help='URL to scrape')
    group.add_argument('-f', '--file', type=str, help='File path to save the scraped data')
    parser.add_argument('-d', '--depth', type=int, default=4, help='Max depth for scraping')
    
    args = parser.parse_args()
    url = args.url
    file = args.file
    max_depth = args.depth
    
    if url:
        to_work_urls = [(url, max_depth)]
    elif file:
        to_work_urls = get_urls_from_file(file, max_depth)
    else:
        print('>>>> Usage: python dl.py -u <url> -d <max_depth>')
        print('>>>> Usage: python dl.py -f <file> -d <max_depth>')
        sys.exit(1)
          
    
    # to_work_urls = get_urls(url, max_depth)
    if (len(to_work_urls) < 1):
        print("No URL Detected")
        sys.exit(1)
    if (len(to_work_urls) > 1):
        print("Detected {} URLs".format(len(to_work_urls)))
        # print("urls: ")
        # for url, max_depth in to_work_urls:
            # print(">>>> {} : depth: {}".format(url, max_depth))
        # print("\n\n\n\n------Processing----------------------------------- \n\n\n\n")        
        
    d_url = {}
    total_downloadable_urls = 0

    print("\nScrapping and finding download urls: ")
    import tqdm
    for url, max_depth in tqdm.tqdm(to_work_urls):
        target_download_domain = get_target_domain(url)
        if target_download_domain is None:
            print('>>> Invalid URL. Please enter with http:// or https://')
            sys.exit(1)

        # print('>>>> Target Domain Found: {}'.format(target_download_domain))
        
        urls = crawl_h5ai(target_download_domain, url, 0, max_depth)
        d_url[url] = urls
        total_downloadable_urls += len(urls)
        

    if (total_downloadable_urls == 0):
        print(">>>> No Downloadbale files Found")
        sys.exit(1)
    print()
    print(">>>> Total Downloadable Files: {}".format(total_downloadable_urls))
    # print(">>>> Total Downloaded Files: {}".format(get_downloaded_count(target_download_domain, url, urls)))
    # print(">>>> Total Remaining Files: {}".format(total_downloadable_urls - get_downloaded_count(target_download_domain, url, urls)))
    print()
    
    # ask for confirmation only for single url download
    continue_download = input('Press y to continue: ')
    if (continue_download != 'y'):
        print('>>>> Aborting...')
        sys.exit(1)

    for url, downloadable_urls in d_url.items():        
        load_downloaded_urls(url)
        download_urls(target_download_domain, url, downloadable_urls)
