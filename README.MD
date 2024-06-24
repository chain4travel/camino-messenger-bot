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

## Dependencies
This application depends on the [camino-messenger-protocol](https://github.com/chain4travel/camino-messenger-protocol), which is a separate repository. The protocol defines the protobuf messages and services used for communication between the gRPC server and client.

Version used: [release-6](https://github.com/chain4travel/camino-messenger-protocol/releases/tag/release-6)
## Examples and Docker Support

The repository includes examples demonstrating the setup of gRPC servers and clients under the `examples` folder. Additionally, Dockerfiles and a docker-compose.yml file are provided to simplify the deployment of a provider, distributor bot, and a partner plugin application that mocks responses.

## Getting Started

### Install required olm library

Certainly! The installation of the `olm` library may vary depending on the operating system. Here are instructions for Linux (Debian and Ubuntu) and Windows:

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

1. **Install Homebrew:**
   If you don't have Homebrew installed, you can install it by following the instructions on [https://brew.sh/](https://brew.sh/).

2. **Update Homebrew:**
   ```bash
   brew update
   ```

3. **Install libolm:**
   ```bash
   brew install libolm
   ```

#### Verify Installation

After installing `olm`, you can verify its installation by checking the library version:

```bash
olm --version
```

This command should print the installed `olm` version.

Note: The provided instructions assume a certain level of system familiarity. Adjustments might be necessary based on your specific setup or if you encounter any issues during the installation process.

## Running a single bot

### Build

```bash
./scripts/build.sh
```

### Create Configuration

Create a configuration YAML file (e.g. `camino-messenger-bot.yaml`) based on the provided example `camino-messenger-bot-supplier.yaml.example` or `camino-messenger-bot-distributor.yaml.example`.



### Start the Application

```bash
./build/bot --config camino-messenger-bot.yaml
```

Feel free to explore and customize the application based on your specific requirements. If you encounter any issues or have questions, please refer to the documentation or open an issue in the repository.

## Running both Supplier and Distributor at the same time
For development proposes, you may want to run both Supplier and Distributor bot at the same time.  
### Build 
```bash
./scripts/build_both.sh
```
to build bots to different folders. 

### Create Configuration

Use `camino-messenger-bot-distributor.yaml.example` and `camino-messenger-bot-supplier.yaml.example` files to create configuration YAML files (e.g. `camino-messenger-bot-distributor.yaml` and `camino-messenger-bot-supplier.yaml`). 

### Start the Application

To run the bots, in 2 separate terminals run:  
```bash 
./build/distributor/bot --config camino-messenger-bot-distributor.yaml
```  

```bash 
./build/supplier/bot --config camino-messenger-bot-supplier.yaml
```

Or configure the launching file of your preferred IDE to do the same.

⚠️ WARNING:  
 YAML **configuration files** contain private keys and thus **should never be committed**. We suggest adding any configuration file names to [.git/info/exclude](`camino-messenger-bot-distributor.yaml.example`) to have git ignore them without modifying the .gitignore.


### Running partner plugin example
Run the partner plugin, setting the `PORT` env as defined in your supplier bot configuration  `partner_plugin_port`  e.g.:  
`PORT=50052 go run examples/rpc/partner-plugin/server.go`



## Tracing
The application supports tracing by providing an otel (OpenTelemetry) sdk implementation adding traces to all requests and responses. 
Traces cross the boundaries of the application by using traceIDs and context propagation.
The tracing configuration can be set in the configuration file.
By default, tracing is disabled. For the moment the application uses a Jaeger exporter to send the traces to a Jaeger instance. 
A future improvement could include an otel collector as the middle stop for traces and feed to them different tracing systems.