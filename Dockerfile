FROM heroiclabs/nakama-pluginbuilder:2.11.1 AS builder

ENV GO111MODULE on
ENV CGO_ENABLED 1

WORKDIR /backend
COPY . .

RUN go build --trimpath --mod=vendor --buildmode=plugin -o ./backend.so

FROM heroiclabs/nakama:2.11.1

COPY --from=builder /backend/backend.so /nakama/data/modules
