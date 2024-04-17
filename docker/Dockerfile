FROM ubuntu:22.04

ARG UID
ARG USERNAME
ARG GID
ARG GROUPNAME
ARG GO_VERSION

WORKDIR /app

RUN apt update && \
    apt install -y curl && \
    curl -OL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz && \
    groupadd -g $GID $GROUPNAME && \
    useradd -m -s /bin/bash -u $UID -g $GID $USERNAME && \
    chown -R $USERNAME:$GROUPNAME /app

USER $USERNAME
ENV PATH $PATH:/usr/local/go/bin