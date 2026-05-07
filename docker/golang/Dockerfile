FROM ubuntu:bionic AS brotli-builder

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get -y install \
        build-essential git wget && \
    rm -rf /var/lib/apt/lists/*

RUN wget https://github.com/Kitware/CMake/releases/download/v3.31.0/cmake-3.31.0-linux-x86_64.tar.gz && \
    echo "0fcb338b4515044f9ac77543550ac92c314c58f6f95aafcac5cd36aa75db6924  cmake-3.31.0-linux-x86_64.tar.gz" | sha256sum -c && \
    tar xf cmake-3.31.0-linux-x86_64.tar.gz -C /usr --strip-components=1 && rm cmake-3.31.0-linux-x86_64.tar.gz

RUN git clone https://github.com/google/brotli.git /tmp/brotli && \
    git -C /tmp/brotli checkout 664952333f675eb189b9d8dfefb1d33538cc41f6

RUN cd /tmp/brotli && \
    mkdir out && cd out && \
    cmake -DCMAKE_INSTALL_PREFIX=/brotli-install .. && \
    make -j$(nproc) && make install

FROM ubuntu:bionic

# custom golang image with CMake and Brotli support (compatible with ubuntu 18.04..22.04)

ENV DEBIAN_FRONTEND=noninteractive
ENV TERM=xterm-256color

RUN apt-get update && \
    apt-get -y install \
        build-essential cmake \
        iputils-ping \
        net-tools dnsutils \
        wget

RUN wget -O - "https://go.dev/dl/go1.25.2.linux-amd64.tar.gz" | tar -C /usr/local -xz && \
	export PATH="/usr/local/go/bin:$PATH" && \
	go version

ENV GOPATH=/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

RUN wget https://github.com/Kitware/CMake/releases/download/v3.31.0/cmake-3.31.0-linux-x86_64.tar.gz && \
    echo "0fcb338b4515044f9ac77543550ac92c314c58f6f95aafcac5cd36aa75db6924  cmake-3.31.0-linux-x86_64.tar.gz" | sha256sum -c && \
    tar xf cmake-3.31.0-linux-x86_64.tar.gz -C /usr --strip-components=1 && rm cmake-3.31.0-linux-x86_64.tar.gz

COPY --from=brotli-builder /brotli-install/ /usr/local/
