install:
	go mod download; \
    go install github.com/daixiang0/gci@v0.13.0; \
    go install github.com/matryer/moq@v0.3.3; \
    go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.16.2; \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0; \

format:
	$(GOPATH)/bin/gci write $$(find . -type f -name '*.go' -not -path "./pkg/proto/*" -not -name "*.gen.go" -not -path "*/mock/*") -s standard -s default

build_worker_docker:
	docker build --tag=safe-ai-coder/worker:latest --build-arg SERVICE_NAME=worker --file=docker/Dockerfile .

build_worker_mac:
	go build -o ./bin/worker ./applications/worker/cmd/