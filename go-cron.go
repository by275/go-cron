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

func execute(ctx context.Context, command string, args []string) {
	log.Printf("executing: %s %s", command, strings.Join(args, " "))

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
		log.Printf("stopping command process group: %s %s", command, strings.Join(args, " "))
		killProcessGroup(cmd.Process.Pid, syscall.SIGTERM)

		select {
		case err := <-done:
			if err != nil && !isShutdownError(ctx, err) {
				log.Printf("command failed: %v", err)
			}
		case <-time.After(5 * time.Second):
			log.Printf("forcing command process group to exit: %s %s", command, strings.Join(args, " "))
			killProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
			if err := <-done; err != nil && !isShutdownError(ctx, err) {
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

func isShutdownError(ctx context.Context, err error) bool {
	if !errors.Is(ctx.Err(), context.Canceled) {
		return false
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		status, ok := exitErr.Sys().(syscall.WaitStatus)
		return ok && (status.Signaled() || status.ExitStatus() != 0)
	}

	return errors.Is(err, context.Canceled)
}

func create(ctx context.Context) *cron.Cron {
	var schedule string = os.Args[1]
	var command string = os.Args[2]
	var args []string = os.Args[3:len(os.Args)]

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
	os.Exit(0)
}

func main() {

	if len(os.Args) < 3 {
		log.Fatalln("Not enough arguments: Usage: go-cron SCHEDULE COMMAND [ARGS]")
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := create(ctx)

	c.Start()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("received signal: %s", <-ch)

	stop(cancel, c)
}
