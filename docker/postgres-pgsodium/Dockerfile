# syntax=docker/dockerfile:1
# Тег указан явно вместе с digest: без тега Renovate (dockerfile-менеджер) не может
# понять, какой тег искать, и никогда не предлагает обновление digest.
# Sonar docker:S8431 (тег+digest одновременно) погашен в sonar-project.properties (e37).
FROM postgres:18-alpine@sha256:d3e1620b530c944afa6e887d22eb899824da68e19c52024bf98f5220c88a65b2

LABEL maintainer="FitPulse <dev@fitpulse.app>"
LABEL description="PostgreSQL with pgsodium extension for FitPulse"

# Работаем в /tmp, чтобы безопасно удалить папку потом
WORKDIR /tmp

# Build pgsodium, then re-install the libsodium runtime *after*
# `apk del .build-deps` — that cleanup used to take the shared library with it,
# which produced an image that builds fine but dies at boot with
# "could not load library pgsodium.so: libsodium.so.26: No such file or
# directory" and takes the whole stack down. The final `apk info -L` assertion
# makes such an image impossible to build.
#
# apk upgrade --no-cache в начале: база запинена по digest, поэтому патчи Alpine
# (например libssl3 3.5.9-r1 с пятью CVE из 3.5.9-r0) иначе не попадают в образ
# до следующего обновления digest через Renovate.
# hadolint ignore=DL3003,DL3018
RUN apk upgrade --no-cache \
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
    && apk del .build-deps \
    && apk add --no-cache libsodium \
    && apk info -L libsodium > /tmp/libsodium-files \
    && grep -q 'libsodium\.so' /tmp/libsodium-files \
    && rm -f /tmp/libsodium-files

# pgsodium refuses to start when shared_preload_libraries includes it but this
# script is missing. It must print exactly 64 hex characters (the DB_ENCRYPTION_KEY
# from app-secrets, mounted at /etc/secrets in Kubernetes).
COPY docker/postgres-pgsodium/pgsodium_getkey /usr/local/share/postgresql/extension/pgsodium_getkey
RUN chmod 0755 /usr/local/share/postgresql/extension/pgsodium_getkey

HEALTHCHECK CMD ["pg_isready", "-U", "postgres"]

USER 999

EXPOSE 5432
