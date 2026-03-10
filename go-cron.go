package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/robfig/cron/v3"
)

func execute(ctx context.Context, command string, args []string) {
	println("executing:", command, strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, command, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("command failed: %v", err)
	}
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
	println("new cron:", schedule)

	if _, err := c.AddFunc(schedule, func() {
		execute(ctx, command, args)
	}); err != nil {
		log.Fatalf("invalid schedule %q: %v", schedule, err)
	}

	return c
}

func start(c *cron.Cron) {
	c.Start()
}

func stop(cancel context.CancelFunc, c *cron.Cron) {
	println("Stopping")
	cancel()
	stopCtx := c.Stop()
	println("Waiting")
	<-stopCtx.Done()
	println("Exiting")
	os.Exit(0)
}

func main() {

	if len(os.Args) < 3 {
		log.Fatalln("Not enough arguments: Usage: go-cron SCHEDULE COMMAND [ARGS]")
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := create(ctx)

	go start(c)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	println(<-ch)

	stop(cancel, c)
}
