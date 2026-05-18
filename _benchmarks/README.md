# Benchmarks

Higher is better.

install all:

```bash
go mod tidy

# update all
go get -u -v ./...
```

## Run target server

```bash
go run ./chi
go run ./echo
go run ./fasthttp
go run ./gin
go run ./gorilla-mux
go run ./httprouter
go run ./muxie
go run ./raw-mux
go run ./rux
```

## Results

- install bombardier:

```bash
go get -u github.com/codesenberg/bombardier
```

### Static Path

```bash
bombardier -c 125 -n 1000000 http://localhost:3000
bombardier -c 200 -n 1000000 http://localhost:3000
```

### Parameterized (dynamic) Path

```bash
bombardier -c 125 -n 1000000 http://localhost:3000/user/42
bombardier -c 200 -n 1000000 http://localhost:3000/user/42
```

### Details

run serve: `go run ./rux`

- [2020.04.08](./testdata/2020.0418.md)
