#!/usr/bin/env python3

'''
Darknet Page Downloader
'''

import pandas as pd
from random import randint
import time
import os
import sys
import multiprocessing

# environment variables
download_directory = '../data/universe-labelled/'
process_name = 'tor'
bot_user_agent = 'Mozilla/5.0 (Windows NT 10.0; rv:68.0) Gecko/20100101 Firefox/68.0'
limit_rate_value = 70
max_redirect_value = 0
number_of_tries = 1

# checks if Tor process is active on system, if not it will activate it
def check_tor_process():
    # move processes variable here, otherwise it will initiate the variable
    # when the module is imported, rather than when the function is called.
    # this "could" cause issues
    processes = os.popen('ps -Af').read()
    if process_name not in processes[:]:
        new_process = 'nohup python %s &' % (process_name)
        os.system(new_process)
    return

# checks if directory for resulting files exists. Will auto-make it otherwise
def check_directory(download_directory):
    os.system('mkdir -p ' + download_directory)
    return

def wget_page(directory, url):
    wget_string = ('torify wget --user-agent="' + bot_user_agent + '" ' +
               '--retry-connrefused ' +
               '--content-on-error ' +
               '--tries=' + str(number_of_tries) + ' ' +
               '--limit-rate ' + str(limit_rate_value) + 'k ' +
               '--max-redirect ' + str(max_redirect_value) + ' ' +
               '--header="Referer: http://' + url + '" ' + ' ')

    os.system(wget_string +
              '"http://' + url + '" ' +
              '-O "' + directory + url + '.html"')

    time.sleep(randint(3, 6))

if __name__ == '__main__':

    check_directory(download_directory)
    check_tor_process()
    workers = multiprocessing.Pool(processes=int(multiprocessing.cpu_count()))

    df = pd.read_excel(download_directory + 'DUTA_10K.xls')
    urls = df['Onion_Address'].tolist()

    workers = multiprocessing.Pool(processes=int(multiprocessing.cpu_count()))
    out = [workers.apply_async(wget_page, args=(download_directory, url)) for url in urls]
    workers.close()
    workers.join()
