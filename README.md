# Components
## Conf-Master (Central Configution Server)
### Git Orgarnization
- datacenters/<datacenter name>/version
- services/<service name>/version
- applications/<application name>/version

## Conf-Slave (Surrogate Configuration Server for each datacenter)
Conf-Slave serves as a cache for each datacenter and contains Git repository. It monitors dataceters branch 'datacenters/<datacenter ID>/<datacenter version>' to detect any changes in configuration for the datacenter. Notification in changes can be done in a sperate channel -- using consul or other method to mitigate a possible issue with load in the master server
### Configuration fetcher 
- bootup configuration
  - datacenter ID, datacenter Version
  - url of central repository
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
- One AWS host (US-WEST) as a central repository
- A job in main cluster as a surrogate git repository (peristent storage desirable)
- Multiple test jobs to fetch configuration and show configuration changes (HUP signal)

## Other Issues
- backup Conf-Master
- Load balance & fault tolerant scheme for Conf-Slave
