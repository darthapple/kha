package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/darthapple/kha/internal/executor"
	"github.com/darthapple/kha/internal/manager"
	"github.com/darthapple/kha/internal/slots"
)

func main() {
	cfg, err := manager.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	nc, err := nats.Connect(cfg.NATSUrl)
	if err != nil {
		log.Fatalf("connect NATS: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("init jetstream: %v", err)
	}

	store, err := slots.New(js, cfg.SlotTTL)
	if err != nil {
		log.Fatalf("init slot store: %v", err)
	}

	exec := executor.NewDockerExecutor(executor.DockerConfig{
		Image:              cfg.SkillImage,
		ClickUpAPIKey:      cfg.ClickUpAPIKey,
		AnthropicAPIKey:    cfg.AnthropicAPIKey,
		ClaudeOAuthToken:   cfg.ClaudeOAuthToken,
		RepoURL:            cfg.RepoURL,
		GitToken:           cfg.GitToken,
		SocketPath:         os.Getenv("DOCKER_SOCKET"),
	})

	sched := manager.NewScheduler(cfg, store, exec)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sched.Run(ctx)
	log.Println("manager stopped")
}
