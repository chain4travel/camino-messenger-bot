# Camino-Messenger-Bot

Camino-Messenger-Bot is designed to facilitate communication through various messaging platforms. The application is modularized into distinct components, each serving a specific purpose.

## Components

1. **Messenger**

   - The Messenger component declares a decoupled messenger interface with the functionalities that include sending and receiving messages asynchronously.

   Currently, the only concrete implementation is the `matrix-messenger`, which utilizes the [Mautrix Golang SDK](https://github.com/mautrix/go) for the Matrix client. Note that the `mautrix` dependency requires the `olm` C library during both build time and runtime.

2. **Message Processor**

   - The Message Processor utilizes the Messenger to receive and process messages.

3. **gRPC Server**

   - This component acts as a Message Request gateway, accepting protobuf messages. These requests are then forwarded to the Message Processor.

4. **gRPC Client**
   - The gRPC Client communicates with a middleware or "partner-plugin" via gRPC, exchanging protobuf messages. The middleware handles necessary conversions and interacts with the external system of the partner. It's important to note that the middleware is not part of this application.

## Camino Messenger Protocol (CMP)

This application depends on the [camino-messenger-protocol](https://github.com/chain4travel/camino-messenger-protocol), which is a separate repository. The protocol defines the protobuf messages and services used for communication between the gRPC server and client.

Currently [release-10](https://github.com/chain4travel/camino-messenger-protocol/releases/tag/release-10) of Camino Messenger Protocol is used.

## Camino Messenger Account (CM Account)

To run the camino-bot-messenger as intended you will need a **CM Account** with a **bot** added to it and **registered services**.

### Creating the CM Account
If you don't already have a **CM Account**, for development purposes, a CM Account can be created via [camino messenger contracts](https://github.com/chain4travel/camino-messenger-contracts) `hardhat` tool or via [suite](https://suite.camino.network) as explained [here](https://docs.camino.network/partners/partner-config/#step-by-step-configuration-guide).

To create a **CM Account** and add the bot to it - you will need **two** (test) **wallets** - one for the **CM Account**, and one for the **bot**. The CM Account wallet has to be **KYC verified**. Creating and KYC verifying the test wallets can be done via [suite](https://suite.camino.network/create) as explained [here](https://docs.camino.network/guides/how-to-create-a-wallet) and [here](https://docs.camino.network/guides/kyc#kyc-verification-process).

> [!IMPORTANT]
> Remember to ensure the suite is connected to testnet Columbus if you are doing this for development purposes. 

### Creating the CM Account via hardhat
After you have your two wallets ready, clone the contracts repo, position inside the repo, and run:
```
yarn install
```
And then
```
yarn hardhat account --help
yarn hardhat account create --help
yarn hardhat account bot:add --help
yarn hardhat account service:add --help  
```

To see how to use the tool.

1. To **create the CM Account** run:
```
yarn hardhat account create --private-key <0xstatic-private-key-of-the-cm-account-wallet> --network columbus
```
The command will output your new **CM Account Address** in format 0x<\...\>. Store it and add it to your `.yml` configuration under `cm_account_addess`.
To retrieve your bot wallet static private key, follow the instructions [here](https://docs.camino.network/guides/how-to-login-to-your-wallet#how-to-retrieve-your-private-key).  

2. To **add the bot** to the CM Account run:
```
yarn hardhat account add:bot --bot <0xc-chain-address-of-the-bot-wallet> --cm-account  <0xCMAccount-address> --private-key <0xstatic-private-key-of-the-cm-account-wallet> --network columbus
```
Add this bot's private key to your `.yml` configuration under `bot_key`.

3. To **register a service** supported by your CM Account e.g.
```
yarn hardhat account service:add --cm-account <0xCMAccount-address> --private-key <0xstatic-private-key-of-the-cm-account-wallet>  --service-name cmp.services.transport.v3.TransportSearchService --fee 3 --network Columbus
```
To see all the services you can support, run:
```
yarn hardhat manager services:list --network columbus
```

## Examples and Docker Support

The repository includes examples demonstrating the setup of gRPC servers and clients under the `examples` folder. Additionally, Dockerfiles and a docker-compose.yml file are provided to simplify the deployment of a provider, distributor bot, and a partner plugin application that mocks responses.

> [!Note]
> If docker-compose.yml is used, please check volumes. The configuration filepath should be `cmb-config/config.yaml`. An example can be copied over from examples/config/ and modified.


## Getting Started

### Install required `olm` library

Below are instructions for installing the `olm` library for Linux (Debian and Ubuntu) and macOS:

#### Linux (Debian/Ubuntu)

1. **Update Package Lists:**

   ```bash
   sudo apt update
   ```

2. **Install Dependencies:**
   ```bash
   sudo apt install -y libolm-dev
   ```

#### macOS

1. **Update Homebrew:**

   ```bash
   brew update
   ```

   > [!TIP]
   > If you don't have Homebrew installed, you can install it by following
   > the instructions on [https://brew.sh/](https://brew.sh/).

2. **Install libolm:**
   ```bash
   brew install libolm
   ```

## Build

Use the provided build script to build the application. This will create a binary in the `./build` directory.

```bash
./scripts/build.sh
```

## Run

### Configuration

Use the provided example configuration files for suppliers
(`examples/config/camino-messenger-bot-supplier.yaml.example`) and distributors
(`examples/config/camino-messenger-bot-distributor.yaml.example`).

Copy them accordingly and edit fields, mainly the private keys and CM Account address. Most of the default
values should be fine for testing.

> [!IMPORTANT]
> Keep in mind that the addresses of `bot_key` in the config file needs to
> have funds on C-Chain to pay for the transaction fees. (If you are using **MintRequest**)

### Start the Application

Start the bot using the related config file depending on your need.

**Supplier Bot:**

```bash
./build/bot --config camino-messenger-bot-supplier.yaml
```

**Distributor Bot:**

```bash
./build/bot --config camino-messenger-bot-distributor.yaml
```

> [!TIP]
> For testing and development, you can also start two bots, one supplier and one distributor.

Feel free to explore and customize the application based on your specific requirements. If you encounter any issues or have questions, please refer to the documentation or open an issue in the repository.

### Running partner plugin example

Run the partner plugin, setting the `PORT` env as defined in your supplier bot
configuration file's `partner_plugin_port` field, e.g.:

```bash
PORT=50051 go run examples/rpc/partner-plugin/server.go
```

## Tracing

The application supports tracing by providing an otel (OpenTelemetry) sdk
implementation adding traces to all requests and responses.

Traces cross the boundaries of the application by using traceIDs and context propagation.

The tracing configuration can be set in the configuration file. By default, tracing
is disabled.

For the moment the application uses a Jaeger exporter to send the traces to a Jaeger
instance. A future improvement could include an otel collector as the middle stop
for traces and feed to them different tracing systems.
