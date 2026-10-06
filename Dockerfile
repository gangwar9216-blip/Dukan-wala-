FROM golang:1.24-alpine

RUN apk add --no-cache unzip
WORKDIR /build

COPY Healthcare_Backend_Render_Upload.zip /tmp/backend.zip
RUN unzip -q /tmp/backend.zip -d /build

RUN go mod download
RUN go build -o /app/healthcare-backend ./cmd/server

EXPOSE 10000
CMD ["/app/healthcare-backend"]
