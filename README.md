go-cron
=========

Simple golang wrapper over `github.com/robfig/cron` and `os/exec` as a cron replacement

## Usage

```bash
$ go-cron "* * * * * *" /bin/bash -c "echo 1"
new cron: * * * * * *
executing: /bin/bash -c echo 1
1
executing: /bin/bash -c echo 1
1
executing: /bin/bash -c echo 1
1
executing: /bin/bash -c echo 1
1
^C(0x4ed9b0,0x56ebf0)
Stopping
Waiting
Exiting
```
