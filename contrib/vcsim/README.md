# Build instructions for vCenter Server Simulator

To build `vcsim` from latest release, run

```
docker build -t vscim .
```

For testing with Swarm run:
```
docker run -p 8989:8989 vcsim
```