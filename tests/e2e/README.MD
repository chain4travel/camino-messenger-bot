## Setting up environment

```
./scripts/build.sh
./scripts/build_partner_plugin.sh
```

## Running tests

### Running from terminal

In terminal (with `--debug` flag specified if you want to see each request and response message printed out)

```
./scripts/e2e.sh --debug
```

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
