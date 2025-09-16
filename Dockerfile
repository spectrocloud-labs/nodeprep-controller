# Build
FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/nodeprep-controller main.go

# Run
FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot
COPY --from=build /out/nodeprep-controller /nodeprep-controller
ENTRYPOINT ["/nodeprep-controller"]
