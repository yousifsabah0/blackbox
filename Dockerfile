FROM golang:1.22.3-alpine3.19

WORKDIR /app/bb

COPY go.* ./
RUN go mod download && go mod verify

COPY . .

RUN go build -o ./bin/main ./cmd/api/

EXPOSE 8080

CMD ["./bin/main"]
