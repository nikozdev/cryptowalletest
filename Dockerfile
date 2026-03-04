FROM golang:1.24 AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/server cmd/server/main.go

FROM scratch AS runtime

COPY --from=builder /bin/server /bin/server
COPY --from=builder /src/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/bin/server"]
