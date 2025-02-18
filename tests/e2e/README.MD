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
{ // e2e
    "name": "e2e",
    "type": "go",
    "request": "launch",
    "mode": "test",
    "program": "tests/e2e/e2e_test.go",
    "buildFlags": [
        "-tags=e2e",
    ],
    "args": [
        // "-test.parallel=8", // number of parallel test runs
        // "-existing-network-node-uri=http://127.0.0.1:19651", // local node endpoint, if need to use pre-existing local network
        // "-existing-network-admin-key=PrivateKey-vmRQiZeXEXYMyJhEiqdC2z5JhuDbxL8ix9UVvjgMu2Er1NepE",
        "-node=../../../caminogo/build/caminogo", // path to caminogo binary
        "-matrix=../../../camino-conduit/target/release/conduit", // path to conduit binary
        "-partner-plugin=../../build/pp-mock", // path to partner-plugin binary
        "-cmb=../../build/camino-messenger-bot", // path to cmb binary
        "-migration=../../migrations", // path to sql migrations directory
        "-tests-data-dir=/tmp/cmb-e2e", // path to e2e tests data directory
        "-max-parallel-tests=10", // maximum number of parallel tests run
    ]
}
```
