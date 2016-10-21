## sample datacenter configuration

In a main git repository, datacenters/sunny/appConfig.json
```json
{
  "applications": [
    {
      "applicationId": "web2048",
      "repoUrl": "centralgit.cdnetworks.com/web2048Conf",
      "branch": "master"
      "rev": "latest",
    },
    {
      "applicationId": "web4096",
      "repoUrl": "centralgit.cdnetworks.com/4096Conf",
      "branch": "tag",
      "rev": "v1.2"
    },
    {
      "applicationId": "web8192",
      "repoUrl": "centralgit.cdnetworks.com/8192Conf",
      "branch": "topic/test8192",
      "rev": "54FA3A"
    },

  ]
}
```

In the central git server(centralgit.cdnetworks.com), contained

```
centralgit.cdnetworks.com/4096Conf/etc/a.conf
centralgit.cdnetworks.com/4096Conf/b.yml
```

b.yml contains
```
receipt:     Oz-Ware Purchase Invoice
date:        2012-08-06
customer:
    first_name:   Dorothy
    family_name:  Gale
```

## KV storage in datacenter cluster

references:
```
conf/web2048/repo 
            /branch
            /rev

conf/web4096/repo 
            /branch
            /rev
```

values:
```
appConfig/web4096/etc/a.conf
appConfig/web4096/b/receipt
appConfig/web4096/b/date
appConfig/web4096/b/customer
appConfig/web4096/b/customer/first_name
appConfig/web4096/b/customer/family_name
```
