# Build stage — compile a static binary.
FROM golang:1.26 AS build

WORKDIR /src

# Download dependencies first so they are cached across source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO is disabled so the binary runs on a minimal distroless base image.
# Migration SQL is embedded into the binary, so no extra files are needed.
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

# Run stage — a minimal image containing just the binary.
FROM gcr.io/distroless/static-debian12

COPY --from=build /bin/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
