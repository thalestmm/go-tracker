FROM ghcr.io/hybridgroup/opencv:4.13.0

RUN apt-get update && apt-get install -y --no-install-recommends \
    libgtk2.0-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /usr/local/bin/go-tracker .

WORKDIR /data
ENTRYPOINT ["go-tracker"]
