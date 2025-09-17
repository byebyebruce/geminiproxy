FROM golang:1.23-alpine AS builder

ENV GOPROXY='https://goproxy.cn'

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build .

FROM alpine

RUN apk --no-cache add tzdata

COPY --from=builder /src/geminiproxy /app/geminiproxy

WORKDIR /app

EXPOSE 8085

ENTRYPOINT ["./geminiproxy"]