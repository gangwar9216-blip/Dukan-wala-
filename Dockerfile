FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/healthcare-api ./cmd/server

FROM alpine:3.21
RUN adduser -D app
USER app
COPY --from=build /out/healthcare-api /healthcare-api
EXPOSE 8080
ENTRYPOINT ["/healthcare-api"]
