# parse-test-reports

Harness plugin for parsing test reports. The plugin will exit with `exit status 1` if there are any tests failing in directories matching the input globs. The plugin currently only supports JUnit XML test reports.
## Build

Build the binary with the following commands:

```sh
$ go build
```

## Docker

Build the Docker image with the following commands:

```sh
$ GOOS=linux GOARCH=amd64 go build

$ docker buildx build -f docker/Dockerfile -t harnesscommunity/parse-test-reports:latest --platform=linux/amd64 --load .
```

## Usage

Execute from the working directory:
```sh
$ docker run -e PLUGIN_TEST_GLOBS="folder1/*.xml, folder2/*.xml" harnesscommunity/parse-test-reports:latest
```

Execute the plugin in Harness pipeline:
```yaml
  - step:
      type: Plugin
      name: Parse Test Reports Plugin
      identifier: Parse_Test_Reports_Plugin
      spec:
        connectorRef: dockerConnector
        image: harnesscommunity/parse-test-reports:latest
        settings:
          test_globs: folder1/*.xml, folder2/*.xml
```