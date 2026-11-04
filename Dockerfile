FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gobpmbank ./cmd/gobpmbank && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mock-core-banking ./cmd/mock-core-banking

FROM gcr.io/distroless/static-debian12:nonroot AS bank
COPY --from=build /out/gobpmbank /gobpmbank
ENV BANK_ADDR=:8090
EXPOSE 8090
ENTRYPOINT ["/gobpmbank"]

FROM gcr.io/distroless/static-debian12:nonroot AS mock
COPY --from=build /out/mock-core-banking /mock-core-banking
ENV MOCK_ADDR=:8091
EXPOSE 8091
ENTRYPOINT ["/mock-core-banking"]
