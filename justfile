# GoTracker task runner
# Install just: https://github.com/casey/just

default: build

# Build the binary
build:
    go build -o go-tracker .

# Run all tests
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Format code
fmt:
    golangci-lint fmt

# Run linter
lint:
    golangci-lint run

# Run fmt + lint + test
check: fmt lint test

# Build and run with a video
run video:
    go build -o go-tracker . && ./go-tracker -video {{video}}

# Clean build artifacts
clean:
    rm -f go-tracker
