# Define project variables (change as needed)
PROJECT_NAME := pipi
GO_SRC := ./  # Path to your Go source code (usually the root directory)

all: clean build run

# Target to build the executable
build:
	go build -o $(PROJECT_NAME) $(GO_SRC)
# CGO_ENABLED=1 go build -o $(PROJECT_NAME) $(GO_SRC)

# Target to run the project
run: build
	./$(PROJECT_NAME)

# Target for code cleanup (optional)
clean:
	rm -rf $(PROJECT_NAME)
