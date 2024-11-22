# Build stage
FROM golang:1.23.3 AS builder

RUN mkdir /app
COPY . /app
RUN go build -o gitsvc .

# Final stage
FROM scratch

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/gitsvc /gitsvc

# Command to run the executable
ENTRYPOINT ["/gitsvc"]