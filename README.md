go-cron
=========

Simple golang wrapper over `github.com/robfig/cron` and `os/exec` as a cron replacement

## Usage

```bash
$ go-cron "* * * * * *" /bin/bash -c "echo 1"
2026/03/10 11:44:57 new cron: * * * * * *
2026/03/10 11:44:58 executing: /bin/bash -c echo 1
1
2026/03/10 11:44:59 executing: /bin/bash -c echo 1
1
2026/03/10 11:45:00 executing: /bin/bash -c echo 1
1
2026/03/10 11:45:01 executing: /bin/bash -c echo 1
1
^C
2026/03/10 11:45:02 received shutdown signal
2026/03/10 11:45:02 stopping
2026/03/10 11:45:02 waiting
2026/03/10 11:45:02 exiting
```

## Testing

Basic scheduling:

```bash
go run . "* * * * * *" /bin/bash -c 'echo tick'
```

Command failure logging:

```bash
go run . "* * * * * *" /bin/bash -c 'echo before-fail; exit 1'
```

Shutdown while a command is still running:

```bash
go run . "* * * * * *" /bin/bash -c 'echo start; sleep 30; echo end'
```

Press `Ctrl+C` after `start` is printed. If shutdown works correctly, `end` should not be printed.

Process group shutdown, including child processes spawned by the shell:

```bash
go run . "* * * * * *" /bin/bash -c 'sleep 1000 & child=$!; echo "child=$child"; wait'
```

After pressing `Ctrl+C`, check whether the child process is gone:

```bash
ps -fp <child_pid>
```
