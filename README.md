# Components
## Conf-Serv (Central Configution Server)
### Git Orgarnization
- datacentes/<datacenter name>/version
- services/<service name>/version
- applications/<application name>/version

## Conf-Local (Surrogate Configuration Server for each datacenter)
- Serve as cache for datacenter
- Git repository serving datacenter
### Configuration fetcher 
- bootup configuration
  - url of central repository
  - credential
- process
  - pull datacenter branch
  - pull dependent branches
- Populate Consul K/V storage
  - configurations/service/name/branch/commit

## Local Host Configuration
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

** Use cases
  - (Imaginary) Nomad Job Manager detects configuration and restart jobs
  - Local script detects configuratin chagnes and HUP
  - Local go library detects configuratin chagnes and HUP

** Consul K/V storage orgarnization
  - configurations/services/name/version: branch/commitId

# POC setup
- One AWS host (US-WEST) as a central repository
- A job in main cluster as a surrogate git repository (peristent storage desirable)
- Multiple test jobs to fetch configuration and show configuration changes (HUP signal)

