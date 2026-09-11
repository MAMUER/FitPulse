# syntax=docker/dockerfile:1
FROM postgres@sha256:d3e1620b530c944afa6e887d22eb899824da68e19c52024bf98f5220c88a65b2

WORKDIR /tmp/pgsodium

# hadolint ignore=DL3003,DL3018
RUN apk add --no-cache --virtual .build-deps \
        build-base \
        git \
        libsodium-dev \
        postgresql-dev \
    && git clone https://github.com/michelp/pgsodium.git . \
    && make -f Makefile \
    && make install \
    && rm -rf . \
    && apk del .build-deps

HEALTHCHECK CMD pg_isready -U postgres

USER postgres
