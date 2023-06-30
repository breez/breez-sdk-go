// See https://github.com/golang/go/issues/26366.
package lib

import (
	_ "github.com/breez/breez-sdk-go/breezsdk/lib/darwin-aarch64"
	_ "github.com/breez/breez-sdk-go/breezsdk/lib/darwin-amd64"
	_ "github.com/breez/breez-sdk-go/breezsdk/lib/linux-aarch64"
	_ "github.com/breez/breez-sdk-go/breezsdk/lib/linux-amd64"
)
