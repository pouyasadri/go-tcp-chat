FROM golang:1.26 AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/chat-server ./cmd/chat-server

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=builder /out/chat-server /chat-server

EXPOSE 8080

ENTRYPOINT ["/chat-server"]
