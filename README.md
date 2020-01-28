# Simple Grayscale Proxy

A simple proxy server which proxies calls to an origin host and converts valid png/jpeg responses to grayscale. Requests timeout after five seconds. This server has been tested with https://maps.wikimedia.org/, https://secure.gravatar.com/ and https://i.redd.it/ explicitly. No guarantees that every server works correctly, especially if they have bizarre HTTP semantics around their response codes. Does not yet support query parameter passthrough, but this would be a simple feature to add.

## Development

### Configuration

This project relies on the following environment variables:

* `ORIGIN_SERVER`: the host to use as the origin server for requesting images
* `APP_HOST`: the host for the proxy server to listen on
* `APP_PORT`: the port for the proxy server to listen on 

Try the following example configuration:

```bash
export ORIGIN_SERVER=https://maps.wikimedia.org/
export APP_HOST=localhost
export APP_PORT=3000
```

### Build and Run

```bash
git clone git@github.com:dylrich/image-proxy.git && cd image-proxy
go build && ./image-proxy
```