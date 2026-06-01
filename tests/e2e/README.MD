## Setting up environment

```sh
./scripts/build.sh
./scripts/build_partner_plugin_mock.sh
```

## Running tests

### Dependencies

#### Linux
```sh
sudo apt install libclang-dev build-essential
```

#### macOS
```sh
brew install rust
brew install llvm
```

### Running from terminal

The e2e.sh script does:

* Download the latest released dependencies
* Build and execute the end2end tests

Usage:

```sh
./scripts/e2e.sh [--clean] [--debug] [--filter "TESTNAME"] [--caminogo "CAMINOGO_VERSION"] [--camino-conduit "CAMINO_CONDUIT_VERSION"] [--asb "ASB_VERSION"]
```
e.g.
```sh
./scripts/e2e.sh --clean --debug --caminogo "latest" --camino-conduit "latest" --asb "latest"
```
or to download from the dev branch
```sh
./scripts/e2e.sh --clean --debug --caminogo "dev" --camino-conduit "dev" --asb "dev"
```
Different dependencies might require different go versions or otherwise will fail to build. In this case, manually delete the dependency from `build/dependencies` folder, use gvm to adjust the version and re-run the script with just the dependency.
```
gvm use go1.20
./scripts/e2e.sh --debug --caminogo "dev"
```
```
gvm use go1.25
./scripts/e2e.sh --debug --asb "dev"
```

Flags (All optional):

* `--clean` Always re-download dependencies instead of re-using already existing ones
* `--debug` Adds extensive log output for the end-to-end tests, including JSON requests and responses for each test-case
* `--filter` Comma-separated list of top-level test names to execute instead of all e.g. "PingV1,AccommodationV3".
* `--caminogo` Overrides the version to download; if unspecified, the latest version will be used.
* `--camino-conduit` Overrides the version to download; if unspecified, the latest version will be used.
* `--asb` Overrides the version to download; if unspecified, the latest version will be used.
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

Example launch json debug config can be found [here](../../examples/debug-launch-configs/vs_code_launch.json).


### Running from GoLand


Example runConfiguration .xml file can be found [here](../../examples/debug-launch-configs/goland_e2e.xml)


## Troubleshooting
The logs of tests are stored inside `/tmp/cmb-e2e` folder. 
```sh
find /tmp/cmb-e2e -name "node-*.log" 2>/dev/null
```