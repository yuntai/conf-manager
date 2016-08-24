## Issues

- snapshoting
  - disk size
  - traffic load (chattiness vs. throughput requriement)
  - bootup time (initial replication of repository)
  - backup size
  - mechanism

- backup Conf-Master
- Load balance & fault tolerant scheme for Conf-Slave
- Granularity of file and changes

- Support for templating and variable (on which level? support for predicates?)

- Updates strategy
  - Granteeness of simultaneous updates of configuration for multiple clusters (User cases?)
  - Strict or leanient
  - Automatic rollback?
  - Staggering

- Existing usage or system?

- Membership of participants & Fault Detection
  Very well be outside of the scope of the project
  - with asusmtion - given inventory
  - Serf or other gosship protocols?

# Initial Design
Initial idea is to have we have a central repository set up in a redundant way with one slave repository for each datacenter. The slave repository will mirror the master repository and serves as a cache for the data center.

## Conf-Master (Central Configution Git Server)

### Master Gir Repository Orgarnization
- datacenter-configuration/<datacenter>
  master repository for datacenter configuration
  {
    services: [ {
      name: test_service0, 
      branch: experimental_branch,
      tag: latest
    }],
    services: [ {
      name: service0, 
      branch: master,
      tag: v1.0
    }],
  }

- datacenter/<datacenter id>
  specific instance of configuration with inventories(?)
  {
    conf: datacenter-configuration/bigdatacenter/v1.1
    inventories: {
    }
  }

- services/<service name>/version
  {
  }
- applications/<application name>/version
- devices/<device name>/version

## Conf-Slave (Surrogate Configuration Server for each datacenter)
Conf-Slave serves as a cache for each datacenter and contains Git repository. It monitors dataceters branch 'datacenters/<datacenter ID>/version' to detect any changes in configuration for the datacenter. Instead of period pulling changes from the Conf-Slave, we could employ separate notification methanism such as using consul or other method to mitigate a possible risk of overloading the Conf-Master(Git Server) with Gil request.

### Configuration fetcher 
- bootstrap configuration
  Minimally the process requires followin parameters

  - datacenter, datacenter Version
  - The url of central repository
  - credential

- process
  - pull datacenter branch
  - pull dependent branches
- Populate Consul K/V storage
  - configurations/service/name/branch/commit

## Conf-Local (Local Host Configuration Service)
- Serve as local cache
### Fetcher
- configuration
  - url of surrogate repositories, central repositories
  - credential
  - local git directory (possibly reusable)
    - local git repository integrity checker if necessary
      - process
        - Watch consul changes
        - pull datacenter branch
        - pull dependent branches

    - Local configuration service
      - output json
      - export configuration? how? 

## Performance estimation
- Assumption
  - Number of datacenters: 100
  - Number of applications: 100 * 30
  - Frequency of configruation changes: 3 per day
  - Avergage size of configuration file
  - Average size of configuration diff

## Load
- Load of Conf-Master is proporitional to the number of datacenters
- Load of Conf-Master is proportional to the number of jobs in the datacenter

## Use cases
  - (Imaginary) Nomad Job Manager detects configuration and restart jobs
  - Local script detects configuratin chagnes and HUP
  - Local go library detects configuratin chagnes and HUP

## Consul K/V storage orgarnization
  - configurations/services/name/version: branch/commitId

# POC setup
- One AWS host (US-WEST) hosting a central Git repository
- One Conf-Slave in main cluster (optinoal peristent storage)
- Multiple test jobs to fetch configuration and show configuration changes (HUP signal)


## Sample configuration
From Jay's configuration (https://bitbucket.org/cdnetworks/gslb-msa/src/dfd6f207f0a66834861aef82eaa8a27936efd9b2/cache_mgmt/cache_conf.py?at=master&fileviewer=file-view-default)
### nginx upstream
```
server {
    listen       8082;
    server_name  $cache_domain_name;

    set $host_name $org_domain_name;
    set $host_port 80;
    set $x_host "";
    location / {
        include logic/cdns_query.conf;

        proxy_pass http://next;
        proxy_http_versio 1.1;
        proxy_set_header Host $host_name;
        proxy_set_header X-Host $x_host;
    }

    error_page   500 502 503 504  /50x.html;
    location = /50x.html {
        root   html;
    }
}
```

### nginx downstream
```
server {
    listen       80;
    server_name  $cache_domain_name;

    location / {
        proxy_pass http://cache_ats;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
    }

    error_page   500 502 503 504  /50x.html;
    location = /50x.html {
        root   html;
    }
}
```

### ATS HDREWRITE
```
# default max-age
cond %{READ_RESPONSE_HDR_HOOK} [AND]
cond %{HEADER:Cache-Control} /max-age/ [NOT]
add-header Cache-Control "max-age=1111"
```

### ADS REMAP
```
map http://$cache_domain_name/ http://$org_domain_name/ @plugin=header_rewrite.so @pparam=hdr_rewrite/$cache_domain_name.conf
map http://$cache_domain_name/inspect/ http://{cache}/
map http://$cache_domain_name/internal/ http://{cache-internal}/
map http://$cache_domain_name/stat/ http://{stat}/
map http://$cache_domain_name/test/ http://{test}/
map http://$cache_domain_name/hostdb/ http://{hostdb}/
map http://$cache_domain_name/net/ http://{net}/
map http://$cache_domain_name/http/ http://{http}/
```
