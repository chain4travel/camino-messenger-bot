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

Example launch json debug config can be found in `examples/debug-launch-configs/vs_code_launch.json`
