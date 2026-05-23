.PHONY: build run test lint docker-build

SERVICE := simulation-agent
IMAGE := antigravity/$(SERVICE):latest

build:
	cargo build --release

run:
	cargo run

test:
	cargo test -- --nocapture

lint:
	cargo clippy -- -D warnings

fmt:
	cargo fmt

docker-build:
	docker build -t $(IMAGE) .

docker-run:
	docker run --rm -p 8090:8090 $(IMAGE)
