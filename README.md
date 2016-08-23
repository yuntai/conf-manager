# Components
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

## Other Issues
- backup Conf-Master
- Load balance & fault tolerant scheme for Conf-Slave
