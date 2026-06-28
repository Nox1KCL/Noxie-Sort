FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
ADD go.mod go.sum ./
RUN go mod download
ADD . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/nxe-sort ./cmd/

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/nxe-sort .

CMD ["./nxe-sort"]