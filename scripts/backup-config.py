#!/usr/bin/env python3
"""Project-owned configuration scope for the O machine's encrypted cloud backup.

Update this list when persistent configuration or application files change.
PostgreSQL uses its separate 1Panel database backup; raw database files are rejected.
"""
import sys

sys.path.insert(0, '/opt/ogg-backup')
import cloud_config_backup as backup

SPEC = {
    'name': 'O-NewAPI-Config-GoogleDrive',
    'helper': 'O-NewAPI-Config-Upload-GoogleDrive',
    'state': '/opt/1panel/backup/ogg-newapi-config',
    'stage': 'newapi-config',
    'locks': ('/opt/new-api/.deploy.lock',),
    'items': (
        '/opt/new-api/.env', '/opt/new-api/compose.yaml',
        '/opt/new-api/access.json', '/opt/new-api/current.json',
        '/opt/new-api/data/app',
    ),
}

if __name__ == '__main__':
    backup.main(SPEC)
