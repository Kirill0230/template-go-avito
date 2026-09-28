FROM golang:1.27 AS build
WORKDIR /src
COPY .. .
RUN CGO_ENABLED=0 go build -o /trip-service ./cmd/trip-service

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /trip-service /trip-service
USER nonroot
ENTRYPOINT ["/trip-service"]