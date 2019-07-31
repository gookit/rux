# Benchmarks

Higher is better.

## Results

- install bombardier: 

```bash
go get -u github.com/codesenberg/bombardier
```

### Static Path

```bash
bombardier -c 125 -n 1000000 http://localhost:3000
```

### Parameterized (dynamic) Path

```bash
bombardier -c 125 -n 1000000 http://localhost:3000/user/42
```
