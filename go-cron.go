package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
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

func create(ctx context.Context) (cr *cron.Cron, wgr *sync.WaitGroup) {
	var schedule string = os.Args[1]
	var command string = os.Args[2]
	var args []string = os.Args[3:len(os.Args)]

	wg := &sync.WaitGroup{}

	c := cron.New(
		cron.WithParser(
			cron.NewParser(
				cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
			),
		),
	)
	println("new cron:", schedule)

	if _, err := c.AddFunc(schedule, func() {
		wg.Add(1)
		execute(ctx, command, args)
		wg.Done()
	}); err != nil {
		log.Fatalf("invalid schedule %q: %v", schedule, err)
	}

	return c, wg
}

func start(c *cron.Cron, wg *sync.WaitGroup) {
	c.Start()
}

func stop(cancel context.CancelFunc, c *cron.Cron, wg *sync.WaitGroup) {
	println("Stopping")
	cancel()
	c.Stop()
	println("Waiting")
	wg.Wait()
	println("Exiting")
	os.Exit(0)
}

func main() {

	if len(os.Args) < 3 {
		log.Fatalln("Not enough arguments: Usage: go-cron SCHEDULE COMMAND [ARGS]")
	}

	ctx, cancel := context.WithCancel(context.Background())

	c, wg := create(ctx)

	go start(c, wg)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	println(<-ch)

	stop(cancel, c, wg)
}
