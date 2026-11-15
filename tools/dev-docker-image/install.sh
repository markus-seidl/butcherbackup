#!/bin/bash -xe

apt update
apt upgrade -y
apt install -y software-properties-common

# From https://mise.jdx.dev/installing-mise.html#apt
apt install -y extrepo
extrepo enable mise
apt update
apt install -y mise

echo "eval \"$(mise activate bash)\" >> ~/.bashrc"

cd /workspace

mise install

