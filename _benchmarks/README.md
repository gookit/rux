# Benchmarks

Higher is better.

install all:

```bash
go get ./...
# update all
go get -u -v ./...
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
