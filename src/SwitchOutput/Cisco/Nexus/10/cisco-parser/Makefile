BINARY_NAME=cisco-parser
VERSION?=v1.0.0
BUILD_DIR=build
DIST_DIR=dist
GO=go
MAIN_PATH=cmd/main.go

.PHONY: all build clean package

all: clean build package

build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-w -s -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

clean:
	@rm -rf $(BUILD_DIR) $(DIST_DIR) *.tar.gz

package: build
	@mkdir -p $(DIST_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(DIST_DIR)/
	@cp ../show-*.txt $(DIST_DIR)/ 2>/dev/null || true
	@cp ../commands.json $(DIST_DIR)/ 2>/dev/null || true
	@tar -czf cisco-nexus-unified-parser-$(VERSION)-linux-amd64.tar.gz -C $(DIST_DIR) .
	@echo "Package created: cisco-nexus-unified-parser-$(VERSION)-linux-amd64.tar.gz"
