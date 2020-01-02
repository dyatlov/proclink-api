# proclink-api

API Server to process URLs and retrieve oEmbed info out of target pages.

The server returns oembed information for ANY url ( if they respond within provided time limit :) )

To run it and disable access to private ip ranges, do:

`./server -blacklist_ranges "10.0.0.0/8 172.16.0.0/12 192.168.0.0/16"`

### Demo server & request examples:

* Normal site info retrieval:
  - https://proclink.now.sh/?url=https://techcrunch.com
  
* Info retrieval from url which responds slowly (3 seconds delay):
  - https://proclink.now.sh/?url=http://www.deelay.me/3000/http://placehold.it/300x500
