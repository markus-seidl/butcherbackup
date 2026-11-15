FROM ubuntu:26.10

RUN mkdir /workspace

COPY mise.toml /workspace
COPY install.sh /

RUN chmod +x /install.sh && /install.sh
