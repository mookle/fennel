# Builds the cart service image. compose.yaml at the repository root builds
# it with apps/cart as the context.

# The builder and the runtime below share Debian trixie, so the release's
# ERTS links against the same glibc and OpenSSL it finds at run time.
FROM hexpm/elixir:1.20.0-erlang-27.3.4.13-debian-trixie-20260610-slim AS build
WORKDIR /src
ENV MIX_ENV=prod

RUN mix local.hex --force && mix local.rebar --force

# The dependency fetch and compile sit in their own layers, so a source
# change does not fetch or build the dependencies again. config.exs is
# compile-time config, which the dependencies may read.
COPY mix.exs mix.lock ./
COPY config/config.exs config/
RUN mix deps.get --only prod
RUN mix deps.compile

COPY config config
COPY lib lib
# The release bundles ERTS, so the runtime stage needs no Erlang or Elixir.
RUN mix release

FROM debian:trixie-20260610-slim
# ERTS links against libstdc++ and ncurses, and the crypto application loads
# OpenSSL when the release starts.
RUN apt-get update \
 && apt-get install -y --no-install-recommends libstdc++6 libncurses6 openssl ca-certificates \
 && rm -rf /var/lib/apt/lists/*
# The BEAM warns at start without a UTF-8 locale. C.UTF-8 ships with glibc,
# so it needs no locales package.
ENV LANG=C.UTF-8

# An unprivileged user owns the release, which writes its tmp directory at
# start.
RUN useradd --system --create-home --uid 10001 cart
WORKDIR /app
COPY --from=build --chown=cart:cart /src/_build/prod/rel/cart ./
USER cart

EXPOSE 4000
CMD ["/app/bin/cart", "start"]
