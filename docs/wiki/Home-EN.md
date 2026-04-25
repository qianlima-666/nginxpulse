# NginxPulse Wiki

The README stays short (fast start + critical notices). This Wiki holds the details.

## Language
- English (this page)
- 中文: [Home](Home)

## Start here
1. [Quick Start](Quick-Start-EN)
2. [Deployment](Deployment-EN)
3. [SQLite -> PostgreSQL Migration](Migration-SQLite-to-Postgres-EN)
4. [Configuration](Configuration-EN)
5. [Log Sources](Log-Sources-EN)
6. [Supported Log Formats](Supported-Log-Formats-EN)
7. [Agent Collection](Agent-EN)
8. [Log Parsing](Log-Parsing-EN)
9. [IP Geo](IP-Geo-EN)
10. [Database Schema](Database-Schema-EN)
11. [FAQ](FAQ-EN)

## Quick reminders
- Version > 1.5.3 requires PostgreSQL (SQLite is dropped).
- Logs are parsed in system timezone. Make sure the host timezone is correct.
- Website ID is derived from `websites[].name`. Renaming creates a new site.
- Multi-file mounts and remote logs are documented in [Log Sources](Log-Sources-EN); Push Agent is documented in [Agent Collection](Agent-EN).

## Common paths
- Config file: `configs/nginxpulse_config.json`
- Data dir: `var/nginxpulse_data`
- Scan state: `var/nginxpulse_data/nginx_scan_state.json`
- App log: `var/nginxpulse_data/nginxpulse.log`
