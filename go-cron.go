package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
)

const shutdownKillTimeout = 5 * time.Second

func execute(ctx context.Context, command string, args []string) {
	commandLine := strings.TrimSpace(command + " " + strings.Join(args, " "))
	log.Printf("executing: %s", commandLine)

	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Printf("command failed: %v", err)
		return
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("command failed: %v", err)
		}
	case <-ctx.Done():
		log.Printf("stopping command process group: %s", commandLine)
		killProcessGroup(cmd.Process.Pid, syscall.SIGTERM)

		select {
		case err := <-done:
			if err != nil && !isShutdownError(err) {
				log.Printf("command failed: %v", err)
			}
		case <-time.After(shutdownKillTimeout):
			log.Printf("forcing command process group to exit: %s", commandLine)
			killProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
			if err := <-done; err != nil && !isShutdownError(err) {
				log.Printf("command failed: %v", err)
			}
		}
	}
}

func killProcessGroup(pid int, sig syscall.Signal) {
	if err := syscall.Kill(-pid, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
		log.Printf("failed to send %v to process group %d: %v", sig, pid, err)
	}
}

func isShutdownError(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return errors.Is(err, context.Canceled)
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	return ok && (status.Signal() == syscall.SIGTERM || status.Signal() == syscall.SIGKILL)
}

func newCron(ctx context.Context, schedule string, command string, args []string) *cron.Cron {
	c := cron.New(
		cron.WithParser(
			cron.NewParser(
				cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
			),
		),
	)
	log.Printf("new cron: %s", schedule)

	if _, err := c.AddFunc(schedule, func() {
		execute(ctx, command, args)
	}); err != nil {
		log.Fatalf("invalid schedule %q: %v", schedule, err)
	}

	return c
}

func stop(cancel context.CancelFunc, c *cron.Cron) {
	log.Printf("stopping")
	cancel()
	stopCtx := c.Stop()
	log.Printf("waiting")
	<-stopCtx.Done()
	log.Printf("exiting")
}

func main() {

	if len(os.Args) < 3 {
		log.Fatalln("Not enough arguments: Usage: go-cron SCHEDULE COMMAND [ARGS]")
	}

	schedule := os.Args[1]
	command := os.Args[2]
	args := os.Args[3:]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newCron(ctx, schedule, command, args)

	c.Start()

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	<-signalCtx.Done()
	log.Printf("received shutdown signal")

	stop(cancel, c)
}
