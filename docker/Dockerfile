FROM golang:1.22.0-alpine

WORKDIR /app

COPY .env .
COPY .auth.env .
COPY . .

RUN go mod download
RUN go build -o main ./cmd/main.go

EXPOSE 5641

CMD ["./main"]