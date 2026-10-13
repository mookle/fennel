# Builds the product service image. compose.yaml at the repository root
# builds it with apps/product as the context.

FROM golang:1.26 AS build
WORKDIR /src

# The module download sits in its own layer, so a source change does not
# fetch the dependencies again.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# A static binary needs no libc, so it runs on the distroless base below.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/product ./cmd/product

# distroless/static holds CA certificates and time zone data and nothing
# else: no shell and no package manager. The nonroot tag runs as an
# unprivileged user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/product /product
EXPOSE 8080
ENTRYPOINT ["/product"]
