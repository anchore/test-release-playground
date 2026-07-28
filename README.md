# traceart

> [!WARNING]
> This is a sandbox repo to test out release workflows (and other GitHub-isms) before we try them in production.
> The application in this repo is just for testing releasing an application.

Visualise a simulated packet's grand tour on an ASCII world map.

`traceart` invents a plausible traceroute from a random "home" city to a
random destination, geolocates every public hop, projects them onto a
low-resolution mercator world map, draws the path between consecutive
hops, and colours each hop and segment by round-trip time.

Everything is offline. There is no real traceroute, no DNS, no geoip
lookups — the route, the hostnames, the IPs and the RTTs are all
fabricated, but the geometry is real and the gradient is honest.

## Development

```
make help              # list all tasks
make unit              # run unit tests
make static-analysis   # run linters
make snapshot          # build local snapshot binaries with goreleaser
make format            # format source files
make lint-fix          # format + autofix lint issues
```
