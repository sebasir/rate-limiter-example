FROM golang:1.21.3 AS builder

ARG TARGET_APP

WORKDIR /app

COPY ../.. ./

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build cmd/${TARGET_APP}/main.go

FROM gcr.io/distroless/static-debian11 AS build-release-stage

WORKDIR /

COPY --from=builder /app/main /main

USER nonroot:nonroot
ENTRYPOINT ["/main"]