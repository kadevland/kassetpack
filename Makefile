# Variables
BINARY_NAME=kassetpack
CMD_PATH=./cmd/kassetpack
VERSION ?= dev
BUILD_DIR=build

# Check code quality.
vet:
	go vet ./...

# Default build for the current platform.
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

# Run all tests.
test:
	go test -v ./...

# Cross-compile release binaries for all supported platforms.
release:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)_windows_amd64.exe $(CMD_PATH)
	GOOS=linux GOARCH=amd64 go build -ldflags="-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)_linux_amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_arm64 $(CMD_PATH)

# Clean the build directory.
clean:
	rm -rf $(BUILD_DIR)