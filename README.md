# BreezSDK – Go Package

The [Breez SDK](https://github.com/breez/breez-sdk) enables mobile developers to integrate lightning and bitcoin payments into their apps with a very shallow learning curve. More information can be found [here](https://github.com/breez/breez-sdk).

## 👨‍🔧 Installation

To install the package:

```sh
$ go get github.com/breez/breez-sdk-go
```

### Supported platforms

This library embeds the BreezSDK runtime compiled as shared library
objects, and uses [`cgo`](https://golang.org/cmd/cgo/) to consume
it. A set of precompiled shared library objects are provided. Thus
this library works (and is tested) on the following platforms:

<table>
  <thead>
    <tr>
      <th>Platform</th>
      <th>Architecture</th>
      <th>Triple</th>
      <th>Status</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td rowspan="2">Linux</td>
      <td><code>amd64</code></td>
      <td><code>x86_64-unknown-linux-gnu</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td><code>aarch64</code></td>
      <td><code>aarch64-unknown-linux-gnu</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td rowspan="2">Darwin</td>
      <td><code>amd64</code></td>
      <td><code>x86_64-apple-darwin</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td><code>aarch64</code></td>
      <td><code>aarch64-apple-darwin</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td>Windows</td>
      <td><code>amd64</code></td>
      <td><code>x86_64-pc-windows-msvc</code></td>
      <td>⏳</td>
    </tr>
  </tbody>
</table>

## 📄 Usage

``` go
package main

import (
	"log"
	"github.com/breez/breez-sdk-go/breez_sdk"
)

func main() {
	seed, _ := breez_sdk.MnemonicToSeed("cruise clever syrup coil cute execute laundry general cover prevent law sheriff")

	log.Printf("Seed: %v", seed)
}
```

## 💡 Information for Maintainers and Contributors

This repository is used to publish a Go package providing Go bindings to the Breez SDK's [underlying Rust implementation](https://github.com/breez/breez-sdk). The Go bindings are generated using [UniFFi Bindgen Go](https://github.com/NordSecurity/uniffi-bindgen-go).

Any changes to the Breez SDK, the Go bindings, and the configuration of this Go package must be made via the [breez-sdk](https://github.com/breez/breez-sdk) repo.

To release a new version of this package, go to the Actions tab of this GitHub repository. Then select the *Publish Go Package* workflow and run it on the `main` branch. It will ask for a version as input. This allows you to specify the version (e.g., *0.0.1*) of the [breez-sdk](https://github.com/breez/breez-sdk) repository that should be released as a Go package.