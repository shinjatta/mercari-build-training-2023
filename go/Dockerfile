FROM golang:latest

# Set the working directory inside the container
WORKDIR /go/app

# Copy the go.mod and go.sum files to the container
COPY go.mod go.sum ./

# Copy the database to the container
COPY mercari.db ./

# Set the target architecture for cross-compilation
ENV GOOS=linux
ENV GOARCH=amd64

# Download and cache Go dependencies
RUN go mod download

# Copy the source code into the container
COPY ./app ./

# Build the Go application
RUN go build -o main

# Set the command to run the Go application
CMD ["./main"]
