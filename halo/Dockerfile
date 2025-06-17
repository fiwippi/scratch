# -- Build
FROM golang:1.24-alpine AS builder

RUN apk add gcc g++ musl-dev

WORKDIR /app
COPY . .

RUN CGO_ENABLED=1 go build -o ./bin/halo cmd/*.go

# -- Run
FROM alpine:latest

RUN apk add libheif-dev libde265-dev

COPY --from=builder /app/bin/halo /bin/halo

ENTRYPOINT ["/bin/halo"]
