# syntax=docker/dockerfile:1
FROM postgres@sha256:d3e1620b530c944afa6e887d22eb899824da68e19c52024bf98f5220c88a65b2

LABEL maintainer="FitPulse <dev@fitpulse.app>"
LABEL description="PostgreSQL with pgsodium extension for FitPulse"

# Работаем в /tmp, чтобы безопасно удалить папку потом
WORKDIR /tmp

# Keep the libsodium runtime library: pgsodium.so links against it at runtime.
# Installing only libsodium-dev as a virtual build dependency removed the shared
# library again during `apk del .build-deps`, leaving pgsodium unusable at runtime.
# hadolint ignore=DL3003,DL3018
RUN apk add --no-cache libsodium \
    && apk add --no-cache --virtual .build-deps \
    build-base \
    clang \
    git \
    libsodium-dev \
    postgresql-dev \
    && git clone https://github.com/michelp/pgsodium.git pgsodium \
    && cd pgsodium \
    && ln -sf "$(which clang)" /usr/bin/clang-21 \
    && mkdir -p /usr/lib/llvm21/bin \
    && echo '#!/bin/sh' > /usr/lib/llvm21/bin/llvm-lto \
    && echo 'exit 0' >> /usr/lib/llvm21/bin/llvm-lto \
    && chmod +x /usr/lib/llvm21/bin/llvm-lto \
    && make -f Makefile \
    && make install \
    && cd /tmp \
    && rm -rf pgsodium \
    && apk del .build-deps

# pgsodium refuses to start when shared_preload_libraries includes it but this
# script is missing. It must print exactly 64 hex characters (the DB_ENCRYPTION_KEY
# from app-secrets, mounted at /etc/secrets in Kubernetes).
COPY docker/postgres-pgsodium/pgsodium_getkey /usr/local/share/postgresql/extension/pgsodium_getkey
RUN chmod 0755 /usr/local/share/postgresql/extension/pgsodium_getkey

HEALTHCHECK CMD ["pg_isready", "-U", "postgres"]

USER 999

EXPOSE 5432
