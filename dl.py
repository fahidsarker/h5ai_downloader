import os

def url_to_file_name(url):
    return url.replace('http://', '').replace('https://', '').replace('/', '_')

import pickle
def get_source_using_curl(url):
    file_name = url_to_file_name(url)+'.pkl'
    file_path = os.path.join('url_cache', file_name)
    if os.path.exists(file_path):
        print('Using cached file: {}'.format(file_path))
        with open(file_path, 'rb') as f:
            return pickle.load(f)
    else:
        if not os.path.exists('url_cache'):
            os.mkdir('url_cache')
        print('Downloading: {}'.format(url))
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
            download_completed = pickle.load(f)

def download_complete(major_url, url):
    global download_completed
    download_completed.append(url)
    if not os.path.exists('./downloaded_db'):
        os.mkdir('./downloaded_db')
    db_path = os.path.join('./downloaded_db', url_to_file_name(major_url)+'.pkl')
    with open(db_path, 'wb') as f:
        pickle.dump(download_completed, f)

downloadable_urls = []

def get_target_domain(url):
    import re
    match = re.search(r'(http[s]?://[a-zA-Z0-9.-]+)', url)

    if match:
        return match.group(1)
    return None

def crawl_h5ai(target_domain, url, recursion, max_depth):
    if recursion > max_depth:
        return
    print('Crawling: {}'.format(url))
    html = get_source_using_curl(url)
    from bs4 import BeautifulSoup
    soup = BeautifulSoup(html, 'html.parser')
    
    for link in soup.find_all('a'):
        href = link.get('href')
        if href.startswith('..'):
            continue
        if href.endswith('/'):
            url = target_domain + href
            crawl_h5ai(target_domain, url, recursion+1, max_depth)
        else:
            # print('Downloading: {}'.format(href))
            url = target_domain + href
            downloadable_urls.append(url)

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
        subprocess.call(['wget', url, '-O', path])  
        download_complete(major_url, url)

if __name__ == '__main__':
    
    import sys
    if len(sys.argv) > 1:
        url = sys.argv[1]
    else:
        print('>>>> Usage: python dl.py <url>')
        sys.exit(1)

    max_depth = sys.argv[2] if len(sys.argv) > 2 else 4    

    target_download_domain = get_target_domain(url)
    if target_download_domain is None:
        print('>>> Invalid URL. Please enter with http:// or https://')
        sys.exit(1)

    print('>>>> Target Domain Found: {}'.format(target_download_domain))
    
    crawl_h5ai(target_download_domain, url, 0, max_depth)
    
    if (len(downloadable_urls) == 0):
        print(">>>> No Downloadbale files Found")
        sys.exit(1)
    print()
    print(">>>> Total Downloadable Files: {}".format(len(downloadable_urls)))
    print()
    continue_download = input('Press y to continue: ')
    if (continue_download != 'y'):
        print('>>>> Aborting...')
        sys.exit(1)
    load_downloaded_urls(url)
    download_urls(target_download_domain, url, downloadable_urls)
