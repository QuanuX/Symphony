FROM ubuntu:26.04
RUN apt-get update && apt-get install -y --no-install-recommends \
    clang-21 libstdc++-15-dev cmake ninja-build pkg-config libsecret-1-dev \
    openssh-client openssh-server dbus-daemon gnome-keyring libglib2.0-bin \
    golang-go libjson-c-dev attr passwd util-linux && rm -rf /var/lib/apt/lists/*
