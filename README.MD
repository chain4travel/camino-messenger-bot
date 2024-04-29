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

Version used: [release-2](https://github.com/chain4travel/camino-messenger-protocol/releases/tag/release-2)

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

### Build

```bash
go build -o bot cmd/camino-messenger-bot/main.go
```

### Create Configuration

Create a configuration YAML file based on the provided example "camino-messenger-bot.yaml.example".

### Start the Application

```bash
./bot --config camino-messenger-bot.yaml
```

Feel free to explore and customize the application based on your specific requirements. If you encounter any issues or have questions, please refer to the documentation or open an issue in the repository.

## Tracing
The application supports tracing by providing an otel (OpenTelemetry) sdk implementation adding traces to all requests and responses. 
Traces cross the boundaries of the application by using traceIDs and context propagation.
The tracing configuration can be set in the configuration file.
By default, tracing is disabled. For the moment the application uses a Jaeger exporter to send the traces to a Jaeger instance. 
A future improvement could include an otel collector as the middle stop for traces and feed to them different tracing systems.