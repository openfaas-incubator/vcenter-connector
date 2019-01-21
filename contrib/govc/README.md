# Build instructions for govc CLI

To build the `govc` CLI tool from latest release, run

```
docker build -t govc .
```

For testing with Swarm run:
```
docker run -p 8989:8989 govc
```