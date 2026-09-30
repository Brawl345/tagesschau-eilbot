FROM golang:1.26-trixie AS build-stage
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -buildid=" -o /tagesschau-eilbot

FROM gcr.io/distroless/static-debian13:nonroot AS release-stage
WORKDIR /app
COPY --from=build-stage /tagesschau-eilbot /app/tagesschau-eilbot
USER nonroot:nonroot
ENTRYPOINT ["/app/tagesschau-eilbot"]
