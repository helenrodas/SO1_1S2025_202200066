FROM rust:1.85.0 AS builder
WORKDIR /usr/src/app
COPY . .
RUN apt-get update && \
    apt-get install -y pkg-config libssl-dev && \
    rm -rf /var/lib/apt/lists/* && \
    cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y ca-certificates libssl3 && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /usr/src/app/target/release/rust-api /usr/local/bin/rust-api
EXPOSE 8080
CMD ["rust-api"]