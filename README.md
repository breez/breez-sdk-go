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
      <td rowspan="4">Android</td>
      <td><code>amd64</code></td>
      <td><code>x86_64-linux-android</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td><code>aarch64</code></td>
      <td><code>aarch64-linux-android</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td><code>aarch</code></td>
      <td><code>armv7-linux-androideabi</code></td>
      <td>✅</td>
    </tr>
    <tr>
      <td><code>386</code></td>
      <td><code>i686-linux-android</code></td>
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
      <td>Windows</td>
      <td><code>amd64</code></td>
      <td><code>x86_64-pc-windows-msvc</code></td>
      <td>✅</td>
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

## Bundling

For some platforms the provided binding libraries need to be copied into a location where they need to be found during runtime.

### Android

Copy the binding libraries into the jniLibs directory of your app
```bash
cp vendor/github.com/breez/breez-sdk-go/breez_sdk/lib/android-386/*.so android/app/src/main/jniLibs/x86/
cp vendor/github.com/breez/breez-sdk-go/breez_sdk/lib/android-aarch/*.so android/app/src/main/jniLibs/armeabi-v7a/
cp vendor/github.com/breez/breez-sdk-go/breez_sdk/lib/android-aarch64/*.so android/app/src/main/jniLibs/arm64-v8a/
cp vendor/github.com/breez/breez-sdk-go/breez_sdk/lib/android-amd64/*.so android/app/src/main/jniLibs/x86_64/
```
So they are in the following structure
```
└── android
    ├── app
        └── src
            └── main
                └── jniLibs
                    ├── arm64-v8a
                        ├── libbreez_sdk_bindings.so
                        └── libc++_shared.so
                    ├── armeabi-v7a
                        ├── libbreez_sdk_bindings.so
                        └── libc++_shared.so
                    ├── x86
                        ├── libbreez_sdk_bindings.so
                        └── libc++_shared.so
                    └── x86_64
                        ├── libbreez_sdk_bindings.so
                        └── libc++_shared.so
                └── AndroidManifest.xml
        └── build.gradle
    └── build.gradle
```

### Windows

Copy the binding library to the same directory as the executable file or include the library into the windows install packager.
```bash
cp vendor/github.com/breez/breez-sdk-go/breez_sdk/lib/windows-amd64/*.dll build/windows/
```

## 💡 Information for Maintainers and Contributors

This repository is used to publish a Go package providing Go bindings to the Breez SDK's [underlying Rust implementation](https://github.com/breez/breez-sdk). The Go bindings are generated using [UniFFi Bindgen Go](https://github.com/NordSecurity/uniffi-bindgen-go).

Any changes to the Breez SDK, the Go bindings, and the configuration of this Go package must be made via the [breez-sdk](https://github.com/breez/breez-sdk) repo.

To release a new version of this package, go to the Actions tab of this GitHub repository. Then select the *Publish Go Package* workflow and run it on the `main` branch. It will ask for a version as input. This allows you to specify the version (e.g., *0.0.1*) of the [breez-sdk](https://github.com/breez/breez-sdk) repository that should be released as a Go package.
