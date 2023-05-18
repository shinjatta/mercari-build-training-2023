FROM golang:alpine

RUN apk update && apk add --no-cache gcc musl-dev sqlite sqlite-dev

RUN addgroup -S mercari && adduser -S trainee -G mercari

USER trainee

# Set the working directory inside the container
WORKDIR /go/app

# Copy the go.mod and go.sum files to the container
COPY go.mod go.sum ./

# Copy the database to the container
COPY mercari.db /go/app/mercari.db

# Adjust ownership of the working directory recursively
USER root
RUN chown -R trainee:mercari /go/app

# Switch back to the non-root user
USER trainee

# Set the target architecture for cross-compilation
ENV GOOS=linux
ENV GOARCH=amd64

# Enable CGO
ENV CGO_ENABLED=1

# Download and cache Go dependencies
RUN go mod download

# Copy the source code into the container
COPY ./app ./

# Build the Go application
RUN go build -o main

# Set the command to run the Go application
CMD ["./main"]
