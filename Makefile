PROJECT_NAME := sql-proxy
BUILD_VERSION := 1.4.2
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_DIR := build
GO_FILES := src/main.go

# Build with SQL drivers, comment out if unused:
BUILD_WITH_POSTGRES_TAG := postgres
BUILD_WITH_MSSQL_TAG := sqlserver
BUILD_WITH_MYSQL_TAG := mysql

# Go compiler basic settings
GOOS := linux
GOARCH := amd64
GOAMD64 := v2

# Application settings to run:
BIND_PORT := 8080
BIND_ADDR := localhost
MAX_ROWS := 10000
DEBUG_LOG := true
#TLS_CERT := $(BUILD_DIR)/server.crt
#TLS_KEY := $(BUILD_DIR)/server.key

TAGS := -tags=$(BUILD_WITH_POSTGRES_TAG),$(BUILD_WITH_MSSQL_TAG),$(BUILD_WITH_MYSQL_TAG)

# Default
all: prod

clean:
	rm -f $(BUILD_DIR)/$(PROJECT_NAME)
	rm -f $(BUILD_DIR)/$(PROJECT_NAME)-debug

# Build for production
prod: clean
	GOOS=${GOOS} GOARCH=${GOARCH} GOAMD64=${GOAMD64} go build $(TAGS) \
		-ldflags="-s -w \
		-X ${PROJECT_NAME}/src/app.BuildVersion=${BUILD_VERSION} \
		-X ${PROJECT_NAME}/src/app.BuildTime=${BUILD_TIME}" -o $(BUILD_DIR)/$(PROJECT_NAME) $(GO_FILES)
	@echo "Production build completed."

# Build for debugging
debug: clean
	GOOS=${GOOS} GOARCH=${GOARCH} go build $(TAGS) \
		-ldflags="\
		-X ${PROJECT_NAME}/src/app.BuildVersion=${BUILD_VERSION}-debug \
		-X ${PROJECT_NAME}/src/app.BuildTime=${BUILD_TIME}" -o $(BUILD_DIR)/$(PROJECT_NAME)-debug $(GO_FILES)
	@echo "Debug build completed."

# Run
run: debug
	@echo "Running $(PROJECT_NAME) in debug mode..."
	BIND_ADDR=$(BIND_ADDR) BIND_PORT=$(BIND_PORT) MAX_ROWS=$(MAX_ROWS) TLS_CERT=$(TLS_CERT) TLS_KEY=$(TLS_KEY) DEBUG_LOG=$(DEBUG_LOG) $(BUILD_DIR)/$(PROJECT_NAME)-debug

# Run test
test:
	@echo "Running tests..."
	@go test ./... -v
