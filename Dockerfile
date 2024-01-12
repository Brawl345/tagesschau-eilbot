FROM golang:1.21 AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /tagesschau-eilbot

FROM gcr.io/distroless/base-debian12 AS release-stage
WORKDIR /app
COPY --from=build-stage /tagesschau-eilbot /app/tagesschau-eilbot
USER nonroot:nonroot
ENTRYPOINT ["/app/tagesschau-eilbot"]
