# O machine configuration backup

`backup-config.py` owns the list of persistent configuration and ordinary files in
`/opt/new-api/data/app`. Update it with changes to the application's storage layout.
The server's deployment tool installs the script from the fixed release source only
after a successful deployment. The 1Panel command remains:

```sh
/usr/bin/python3 -B /opt/new-api/tools/backup-config.py
```

The shared uploader is maintained in the ogg operations repository and installed as
`/opt/ogg-backup/cloud_config_backup.py`. Both scripts are included in the separate
infrastructure recovery archive. The project script itself contains no credentials.

The main task runs at 03:55 Asia/Shanghai daily. Its native directory upload helper
stays disabled and is invoked by the script. Archives are encrypted into the existing
Google Drive account and retained for 14 runs. The recovery password is stored in a
protected Mac file, separately from Git. A downloaded copy is decrypted and verified
after each upload, including file contents, permissions and empty directories.

The PostgreSQL database has a separate scheduled backup. This configuration backup
does not include raw PostgreSQL/Redis storage, logs, old backups or the whole `/opt`.
Unexpected SQLite files are rejected and require a reviewed online snapshot path.
Source changes during capture abort the run. Pending uploads are never blindly retried.

Restore the encrypted archive into an isolated directory first. Its `newapi-config/`
root contains paths relative to `/`, plus a manifest. Preserve the original `.env`
cryptographic keys, combine it with a compatible PostgreSQL backup, and review queued
external tasks before starting the application. Never overwrite the live service as
part of a restore test. Disable the main task to roll back the schedule; preserve the
password, helper task and valid cloud copies.
