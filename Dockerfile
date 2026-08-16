# syntax=docker/dockerfile:1

ARG DEBIAN_VERSION=bookworm-slim

FROM debian:${DEBIAN_VERSION} AS builder

ARG MAGPIE_TTS_REF=3008ff73fc2d2da9e4d743b09350aa7023e8980c
ARG MODEL_URL=https://huggingface.co/mudler/magpie-tts.cpp-gguf/resolve/main/magpie-tts-multilingual-357m-f16.gguf

RUN apt-get update \
    && apt-get install --no-install-recommends --yes \
        build-essential \
        ca-certificates \
        cmake \
        curl \
        git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

RUN git clone --filter=blob:none --no-checkout https://github.com/mudler/magpie-tts.cpp.git . \
    && git checkout "${MAGPIE_TTS_REF}" \
    && git submodule update --init --depth 1 \
    && cmake -S . -B build \
        -DCMAKE_BUILD_TYPE=Release \
        -DGGML_NATIVE=OFF \
        -DMAGPIE_BUILD_TESTS=OFF \
    && cmake --build build --config Release --parallel

RUN mkdir -p /model \
    && curl --fail --location --retry 3 --output /model/model.gguf "${MODEL_URL}" \
    && test -s /model/model.gguf \
    && /src/build/examples/cli/magpie-cli info --model /model/model.gguf

FROM debian:${DEBIAN_VERSION}

RUN apt-get update \
    && apt-get install --no-install-recommends --yes \
        ffmpeg \
        libgomp1 \
        libstdc++6 \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir /output

COPY --from=builder /src/build/examples/cli/magpie-cli /usr/local/bin/magpie-cli
COPY --from=builder /model/model.gguf /opt/magpie/model.gguf
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh

RUN chmod +x /usr/local/bin/entrypoint.sh

WORKDIR /output
VOLUME ["/output"]

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
