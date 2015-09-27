# proclink-api
API Server for proc.link website

The server returns oembed information for ANY url ( if they respond within provided time limit :) )

To run it and disable access to private ip ranges, do:

`./server -blacklist_ranges "10.0.0.0/8 172.16.0.0/12 192.168.0.0/16"`

Try it online:

A normal website
```
http://api.proc.link/oembed?url=http://techcrunch.com
```

A slow loading image
```
http://api.proc.link/oembed?url=http://deelay.me/1000/http://deelay.me/img/1000ms.gif
```

A very slow image
```
http://api.proc.link/oembed?url=http://deelay.me/5000/http://deelay.me/img/5000ms.gif
```
