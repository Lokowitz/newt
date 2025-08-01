# Newt

Newt is a fully user space [WireGuard](https://www.wireguard.com/) tunnel client and TCP/UDP proxy, designed to securely expose private resources controlled by Pangolin. By using Newt, you don't need to manage complex WireGuard tunnels and NATing.

### Installation and Documentation

Newt is used with Pangolin and Gerbil as part of the larger system. See documentation below:

-   [Full Documentation](https://docs.fossorial.io)

## Preview

<img src="public/screenshots/preview.png" alt="Preview"/>

_Sample output of a Newt container connected to Pangolin and hosting various resource target proxies._

## Key Functions

### Registers with Pangolin

Using the Newt ID and a secret, the client will make HTTP requests to Pangolin to receive a session token. Using that token, it will connect to a websocket and maintain that connection. Control messages will be sent over the websocket.

### Receives WireGuard Control Messages

When Newt receives WireGuard control messages, it will use the information encoded (endpoint, public key) to bring up a WireGuard tunnel using [netstack](https://github.com/WireGuard/wireguard-go/blob/master/tun/netstack/examples/http_server.go) fully in user space. It will ping over the tunnel to ensure the peer on the Gerbil side is brought up. 

### Receives Proxy Control Messages

When Newt receives WireGuard control messages, it will use the information encoded to create a local low level TCP and UDP proxies attached to the virtual tunnel in order to relay traffic to programmed targets.

## CLI Args

- `endpoint`: The endpoint where both Gerbil and Pangolin reside in order to connect to the websocket.
- `id`: Newt ID generated by Pangolin to identify the client.
- `secret`: A unique secret (not shared and kept private) used to authenticate the client ID with the websocket in order to receive commands. 
- `mtu`: MTU for the internal WG interface. Default: 1280
- `dns`: DNS server to use to resolve the endpoint
- `log-level` (optional): The log level to use. Default: INFO
- `updown` (optional): A script to be called when targets are added or removed.
- `tls-client-cert` (optional): Client certificate (p12 or pfx) for mTLS. See [mTLS](#mtls)
- `docker-socket` (optional): Set the Docker socket to use the container discovery integration
- `docker-enforce-network-validation` (optional): Validate the container target is on the same network as the newt process
- `health-file` (optional): Check if connection to WG server (pangolin) is ok. creates a file if ok, removes it if not ok. Can be used with docker healtcheck to restart newt 

- Example:

```bash
./newt \
--id 31frd0uzbjvp721 \
--secret h51mmlknrvrwv8s4r1i210azhumt6isgbpyavxodibx1k2d6 \
--endpoint https://example.com
```

You can also run it with Docker compose. For example, a service in your `docker-compose.yml` might look like this using environment vars (recommended):

```yaml
services:
  newt:
    image: fosrl/newt
    container_name: newt
    restart: unless-stopped
    environment:
      - PANGOLIN_ENDPOINT=https://example.com
      - NEWT_ID=2ix2t8xk22ubpfy 
      - NEWT_SECRET=nnisrfsdfc7prqsp9ewo1dvtvci50j5uiqotez00dgap0ii2
      - HEALTH_FILE=/tmp/healthy 
```

You can also pass the CLI args to the container:

```yaml
services:
  newt:
    image: fosrl/newt
    container_name: newt
    restart: unless-stopped
    command:
        - --id 31frd0uzbjvp721
        - --secret h51mmlknrvrwv8s4r1i210azhumt6isgbpyavxodibx1k2d6
        - --endpoint https://example.com
        - --health-file /tmp/healthy
```

### Docker Socket Integration

Newt can integrate with the Docker socket to provide remote inspection of Docker containers. This allows Pangolin to query and retrieve detailed information about containers running on the Newt client, including metadata, network configuration, port mappings, and more.

**Configuration:**

You can specify the Docker socket path using the `--docker-socket` CLI argument or by setting the `DOCKER_SOCKET` environment variable. On most linux systems the socket is `/var/run/docker.sock`. When deploying newt as a container, you need to mount the host socket as a volume for the newt container to access it. If the Docker socket is not available or accessible, Newt will gracefully disable Docker integration and continue normal operation.

```yaml
services:
  newt:
    image: fosrl/newt
    container_name: newt
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      - PANGOLIN_ENDPOINT=https://example.com
      - NEWT_ID=2ix2t8xk22ubpfy 
      - NEWT_SECRET=nnisrfsdfc7prqsp9ewo1dvtvci50j5uiqotez00dgap0ii2
      - DOCKER_SOCKET=/var/run/docker.sock
```

#### Hostnames vs IPs

When the Docker Socket Integration is used, depending on the network which Newt is run with, either the hostname (generally considered the container name) or the IP address of the container will be sent to Pangolin. Here are some of the scenarios where IPs or hostname of the container will be utilised:
- **Running in Network Mode 'host'**: IP addresses will be used
- **Running in Network Mode 'bridge'**: IP addresses will be used
- **Running in docker-compose without a network specification**: Docker compose creates a network for the compose by default, hostnames will be used
- **Running on docker-compose with defined network**: Hostnames will be used

### Docker Enforce Network Validation

When run as a Docker container, Newt can validate that the target being provided is on the same network as the Newt container and only return containers directly accessible by Newt. Validation will be carried out against either the hostname/IP Address and the Port number to ensure the running container is exposing the ports to Newt.

It is important to note that if the Newt container is run with a network mode of `host` that this feature will not work. Running in `host` mode causes the container to share its resources with the host machine, therefore making it so the specific host container information for Newt cannot be retrieved to be able to carry out network validation.

**Configuration:**

Validation is `false` by default. It can be enabled via setting the `--docker-enforce-network-validation` CLI argument or by setting the `DOCKER_ENFORCE_NETWORK_VALIDATION` environment variable.

If validation is enforced and the Docker socket is available, Newt will **not** add the target as it cannot be verified. A warning will be presented in the Newt logs.

### Updown

You can pass in a updown script for Newt to call when it is adding or removing a target:

`--updown "python3 test.py"`

It will get called with args when a target is added: 
`python3 test.py add tcp localhost:8556`
`python3 test.py remove tcp localhost:8556`

Returning a string from the script in the format of a target (`ip:dst` so `10.0.0.1:8080`) it will override the target and use this value instead to proxy.

You can look at updown.py as a reference script to get started!

### mTLS
Newt supports mutual TLS (mTLS) authentication, if the server has been configured to request a client certificate.
* Only PKCS12 (.p12 or .pfx) file format is accepted
* The PKCS12 file must contain: 
  * Private key
  * Public certificate
  * CA certificate
* Encrypted PKCS12 files are currently not supported

Examples:

```bash
./newt \
--id 31frd0uzbjvp721 \
--secret h51mmlknrvrwv8s4r1i210azhumt6isgbpyavxodibx1k2d6 \
--endpoint https://example.com \
--tls-client-cert ./client.p12
```

```yaml
services:
  newt:
    image: fosrl/newt
    container_name: newt
    restart: unless-stopped
    environment:
      - PANGOLIN_ENDPOINT=https://example.com
      - NEWT_ID=2ix2t8xk22ubpfy 
      - NEWT_SECRET=nnisrfsdfc7prqsp9ewo1dvtvci50j5uiqotez00dgap0ii2 
      - TLS_CLIENT_CERT=./client.p12 
```

## Build

### Container 

Ensure Docker is installed.

```bash
make
```

### Binary

Make sure to have Go 1.23.1 installed.

```bash
make local
```

### Nix Flake

```bash
nix build
```

Binary will be at `./result/bin/newt`

Development shell available with `nix develop`

## Licensing

Newt is dual licensed under the AGPLv3 and the Fossorial Commercial license. For inquiries about commercial licensing, please contact us.

## Contributions

Please see [CONTRIBUTIONS](./CONTRIBUTING.md) in the repository for guidelines and best practices.
