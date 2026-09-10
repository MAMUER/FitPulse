# syntax=docker/dockerfile:1
FROM postgres@sha256:d3e1620b530c944afa6e887d22eb899824da68e19c52024bf98f5220c88a65b2

RUN apk add --no-cache --virtual .build-deps \
        build-base \
        git \
        libsodium-dev \
        postgresql-dev \
    && git clone https://github.com/michelp/pgsodium.git /tmp/pgsodium \
    && cd /tmp/pgsodium \
    && make -f Makefile \
    && make install \
    && rm -rf /tmp/pgsodium \
    && apk del .build-deps

USER postgres
