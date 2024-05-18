FROM golang:1.22.3-alpine3.19

WORKDIR /app/bb

COPY go.* ./

RUN go mod download
RUN go mod verify
RUN go clean -modcache

COPY . .

RUN go build -o ./bin/main ./cmd/api/

EXPOSE 8080

# I really think this is a stupid step by me :3
CMD [ "bin/main" ] 