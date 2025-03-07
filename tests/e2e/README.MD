## Setting up environment

```sh
./scripts/build.sh
./scripts/build_partner_plugin.sh
```

## Running tests

### Running from terminal

The e2e.sh script does:

* Download the latest released dependencies
* Build and execute the end2end tests

Usage:

```sh
./scripts/e2e.sh [--clean] [--debug] [--filter "TESTNAME"] [--caminogo "CAMINOGO_VERSION"] [--camino-conduit "CAMINO_CONDUIT_VERSION"]
```

Flags (All optional):

* `--clean` Always re-download dependencies instead of re-using already existing ones
* `--debug` Adds a lot of log output by the end2end test including JSON requests and responses for ech test-case
* `--filter` Comma-separated list of top-level test names to execute instead of all e.g. "PingV1,AccommodationV3".
* `--caminogo` Overwrites the version to download instead of latest
* `--camino-conduit` Overwrites the version to download instead of latest

### Running from VS Code

For VS Code to work correctly with e2e, following settings must be added because of build constraints:

```json
    "gopls": {
         "build.buildFlags": [
             "-tags=e2e"
         ]
     },
    "go.useLanguageServer": true,
```

Example launch json debug config:

```jsonc
{
  // e2e
  "name": "e2e",
  "type": "go",
  "request": "launch",
  "mode": "test",
  "program": "${workspaceFolder}/tests/e2e/e2e_test.go",
  "buildFlags": ["-tags=e2e"],
  "args": [
    // "-test.parallel=8", // number of parallel test runs
    // "-existing-network-node-uri=http://127.0.0.1:19651", // local node endpoint, if need to use pre-existing local network
    // "-existing-network-admin-key=PrivateKey-vmRQiZeXEXYMyJhEiqdC2z5JhuDbxL8ix9UVvjgMu2Er1NepE",
    "-node=${workspaceFolder}/caminogo/build/dependencies/caminogo", // path to caminogo binary
    "-matrix=${workspaceFolder}/build/dependencies/camino-conduit/camino-conduit-v1.0.0/camino-conduit", // path to conduit binary
    "-partner-plugin=${workspaceFolder}/build/pp-mock", // path to partner-plugin binary
    "-cmb=${workspaceFolder}/build/camino-messenger-bot", // path to cmb binary
    "-migration=${workspaceFolder}/migrations", // path to sql migrations directory
    "-tests-data-dir=/tmp/cmb-e2e", // path to e2e tests data directory
    "-debug" // print requests and responses of e2e tests
  ]
}
```
