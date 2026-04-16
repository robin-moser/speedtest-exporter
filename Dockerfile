FROM golang:1.24 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/speedtest-exporter ./cmd/speedtest-exporter

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /out/speedtest-exporter /speedtest-exporter

EXPOSE 9090

ENTRYPOINT ["/speedtest-exporter"]
