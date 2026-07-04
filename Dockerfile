FROM golang:1.25.6 AS builder

WORKDIR /build

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /taskflow_build ./cmd/server/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /taskflow_build ./taskflow_server

CMD [ "./taskflow_server" ]