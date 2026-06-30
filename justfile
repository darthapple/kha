# List available recipes
default:
    @just --list

# ── Local CLI ─────────────────────────────────────────────────────────────────

# Build kha CLI for the current platform and install to ~/.kha/kha
install:
    make install

# Build all platform binaries to dist/
build:
    make all

# Build manager binary for linux/amd64 (used inside Docker image)
manager:
    make manager

# Remove dist/
clean:
    make clean

# ── Skill image (ghcr.io/darthapple/kha) ─────────────────────────────────────

IMAGE := "ghcr.io/darthapple/kha:latest"

# Build the skill image locally (pre-builds the linux binary first)
build-skill:
    make manager
    docker build -f Dockerfile -t {{IMAGE}} .

# Build and push the skill image to GHCR
push: build-skill
    docker push {{IMAGE}}

# ── Docker (manager + NATS) ───────────────────────────────────────────────────

# Build images and start NATS + manager in the background
up:
    docker compose up -d --build

# Stop all services
down:
    docker compose down

# Stream manager logs (Ctrl-C to exit)
logs:
    docker compose logs -f manager

# Show service status
ps:
    docker compose ps

# Restart manager (picks up .env.agents changes without rebuilding)
restart:
    docker compose restart manager
