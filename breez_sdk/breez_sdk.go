package breez_sdk

// #include <breez_sdk.h>
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// This is needed, because as of go 1.24
// type RustBuffer C.RustBuffer cannot have methods,
// RustBuffer is treated as non-local type
type GoRustBuffer struct {
	inner C.RustBuffer
}

type RustBufferI interface {
	AsReader() *bytes.Reader
	Free()
	ToGoBytes() []byte
	Data() unsafe.Pointer
	Len() uint64
	Capacity() uint64
}

func RustBufferFromExternal(b RustBufferI) GoRustBuffer {
	return GoRustBuffer{
		inner: C.RustBuffer{
			capacity: C.uint64_t(b.Capacity()),
			len:      C.uint64_t(b.Len()),
			data:     (*C.uchar)(b.Data()),
		},
	}
}

func (cb GoRustBuffer) Capacity() uint64 {
	return uint64(cb.inner.capacity)
}

func (cb GoRustBuffer) Len() uint64 {
	return uint64(cb.inner.len)
}

func (cb GoRustBuffer) Data() unsafe.Pointer {
	return unsafe.Pointer(cb.inner.data)
}

func (cb GoRustBuffer) AsReader() *bytes.Reader {
	b := unsafe.Slice((*byte)(cb.inner.data), C.uint64_t(cb.inner.len))
	return bytes.NewReader(b)
}

func (cb GoRustBuffer) Free() {
	rustCall(func(status *C.RustCallStatus) bool {
		C.ffi_breez_sdk_bindings_rustbuffer_free(cb.inner, status)
		return false
	})
}

func (cb GoRustBuffer) ToGoBytes() []byte {
	return C.GoBytes(unsafe.Pointer(cb.inner.data), C.int(cb.inner.len))
}

func stringToRustBuffer(str string) C.RustBuffer {
	return bytesToRustBuffer([]byte(str))
}

func bytesToRustBuffer(b []byte) C.RustBuffer {
	if len(b) == 0 {
		return C.RustBuffer{}
	}
	// We can pass the pointer along here, as it is pinned
	// for the duration of this call
	foreign := C.ForeignBytes{
		len:  C.int(len(b)),
		data: (*C.uchar)(unsafe.Pointer(&b[0])),
	}

	return rustCall(func(status *C.RustCallStatus) C.RustBuffer {
		return C.ffi_breez_sdk_bindings_rustbuffer_from_bytes(foreign, status)
	})
}

type BufLifter[GoType any] interface {
	Lift(value RustBufferI) GoType
}

type BufLowerer[GoType any] interface {
	Lower(value GoType) C.RustBuffer
}

type BufReader[GoType any] interface {
	Read(reader io.Reader) GoType
}

type BufWriter[GoType any] interface {
	Write(writer io.Writer, value GoType)
}

func LowerIntoRustBuffer[GoType any](bufWriter BufWriter[GoType], value GoType) C.RustBuffer {
	// This might be not the most efficient way but it does not require knowing allocation size
	// beforehand
	var buffer bytes.Buffer
	bufWriter.Write(&buffer, value)

	bytes, err := io.ReadAll(&buffer)
	if err != nil {
		panic(fmt.Errorf("reading written data: %w", err))
	}
	return bytesToRustBuffer(bytes)
}

func LiftFromRustBuffer[GoType any](bufReader BufReader[GoType], rbuf RustBufferI) GoType {
	defer rbuf.Free()
	reader := rbuf.AsReader()
	item := bufReader.Read(reader)
	if reader.Len() > 0 {
		// TODO: Remove this
		leftover, _ := io.ReadAll(reader)
		panic(fmt.Errorf("Junk remaining in buffer after lifting: %s", string(leftover)))
	}
	return item
}

func rustCallWithError[E any, U any](converter BufReader[*E], callback func(*C.RustCallStatus) U) (U, *E) {
	var status C.RustCallStatus
	returnValue := callback(&status)
	err := checkCallStatus(converter, status)
	return returnValue, err
}

func checkCallStatus[E any](converter BufReader[*E], status C.RustCallStatus) *E {
	switch status.code {
	case 0:
		return nil
	case 1:
		return LiftFromRustBuffer(converter, GoRustBuffer{inner: status.errorBuf})
	case 2:
		// when the rust code sees a panic, it tries to construct a rustBuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(GoRustBuffer{inner: status.errorBuf})))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		panic(fmt.Errorf("unknown status code: %d", status.code))
	}
}

func checkCallStatusUnknown(status C.RustCallStatus) error {
	switch status.code {
	case 0:
		return nil
	case 1:
		panic(fmt.Errorf("function not returning an error returned an error"))
	case 2:
		// when the rust code sees a panic, it tries to construct a C.RustBuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(GoRustBuffer{
				inner: status.errorBuf,
			})))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		return fmt.Errorf("unknown status code: %d", status.code)
	}
}

func rustCall[U any](callback func(*C.RustCallStatus) U) U {
	returnValue, err := rustCallWithError[error](nil, callback)
	if err != nil {
		panic(err)
	}
	return returnValue
}

type NativeError interface {
	AsError() error
}

func writeInt8(writer io.Writer, value int8) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint8(writer io.Writer, value uint8) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt16(writer io.Writer, value int16) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint16(writer io.Writer, value uint16) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt32(writer io.Writer, value int32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint32(writer io.Writer, value uint32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeInt64(writer io.Writer, value int64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeUint64(writer io.Writer, value uint64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeFloat32(writer io.Writer, value float32) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func writeFloat64(writer io.Writer, value float64) {
	if err := binary.Write(writer, binary.BigEndian, value); err != nil {
		panic(err)
	}
}

func readInt8(reader io.Reader) int8 {
	var result int8
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint8(reader io.Reader) uint8 {
	var result uint8
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt16(reader io.Reader) int16 {
	var result int16
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint16(reader io.Reader) uint16 {
	var result uint16
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt32(reader io.Reader) int32 {
	var result int32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint32(reader io.Reader) uint32 {
	var result uint32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readInt64(reader io.Reader) int64 {
	var result int64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readUint64(reader io.Reader) uint64 {
	var result uint64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readFloat32(reader io.Reader) float32 {
	var result float32
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func readFloat64(reader io.Reader) float64 {
	var result float64
	if err := binary.Read(reader, binary.BigEndian, &result); err != nil {
		panic(err)
	}
	return result
}

func init() {

	FfiConverterCallbackInterfaceEventListenerINSTANCE.register()
	FfiConverterCallbackInterfaceLogStreamINSTANCE.register()
	uniffiCheckChecksums()
}

func uniffiCheckChecksums() {
	// Get the bindings contract version from our ComponentInterface
	bindingsContractVersion := 26
	// Get the scaffolding contract version by calling the into the dylib
	scaffoldingContractVersion := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint32_t {
		return C.ffi_breez_sdk_bindings_uniffi_contract_version()
	})
	if bindingsContractVersion != int(scaffoldingContractVersion) {
		// If this happens try cleaning and rebuilding your project
		panic("breez_sdk: UniFFI contract version mismatch")
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_connect()
		})
		if checksum != 2295 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_connect: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_default_config()
		})
		if checksum != 38312 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_default_config: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_mnemonic_to_seed()
		})
		if checksum != 19340 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_mnemonic_to_seed: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_parse_input()
		})
		if checksum != 46630 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_parse_input: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_parse_invoice()
		})
		if checksum != 4794 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_parse_invoice: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_service_health_check()
		})
		if checksum != 1079 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_service_health_check: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_set_log_stream()
		})
		if checksum != 25613 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_set_log_stream: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_static_backup()
		})
		if checksum != 34455 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_static_backup: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_backup()
		})
		if checksum != 36004 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_backup: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_backup_status()
		})
		if checksum != 51417 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_backup_status: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_buy_bitcoin()
		})
		if checksum != 36346 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_buy_bitcoin: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_check_message()
		})
		if checksum != 43483 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_check_message: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_claim_reverse_swap()
		})
		if checksum != 5979 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_claim_reverse_swap: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_close_lsp_channels()
		})
		if checksum != 37352 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_close_lsp_channels: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_configure_node()
		})
		if checksum != 40371 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_configure_node: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_connect_lsp()
		})
		if checksum != 41822 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_connect_lsp: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_disconnect()
		})
		if checksum != 25385 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_disconnect: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_execute_dev_command()
		})
		if checksum != 16243 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_execute_dev_command: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_fiat_rates()
		})
		if checksum != 38513 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_fiat_rates: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_lsp_info()
		})
		if checksum != 40985 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_lsp_info: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_reverse_swap_fees()
		})
		if checksum != 56465 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_reverse_swap_fees: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_generate_diagnostic_data()
		})
		if checksum != 22880 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_generate_diagnostic_data: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_in_progress_onchain_payments()
		})
		if checksum != 39619 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_in_progress_onchain_payments: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_in_progress_swap()
		})
		if checksum != 64161 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_in_progress_swap: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_fiat_currencies()
		})
		if checksum != 54166 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_fiat_currencies: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_lsps()
		})
		if checksum != 30571 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_lsps: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_payments()
		})
		if checksum != 44520 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_payments: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_refundables()
		})
		if checksum != 53017 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_refundables: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_swaps()
		})
		if checksum != 14075 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_swaps: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lnurl_auth()
		})
		if checksum != 23740 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lnurl_auth: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lsp_id()
		})
		if checksum != 53031 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lsp_id: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lsp_info()
		})
		if checksum != 38310 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lsp_info: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_node_credentials()
		})
		if checksum != 5169 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_node_credentials: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_node_info()
		})
		if checksum != 17807 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_node_info: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_onchain_payment_limits()
		})
		if checksum != 58677 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_onchain_payment_limits: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_open_channel_fee()
		})
		if checksum != 65044 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_open_channel_fee: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_pay_lnurl()
		})
		if checksum != 59900 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_pay_lnurl: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_pay_onchain()
		})
		if checksum != 34167 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_pay_onchain: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_payment_by_hash()
		})
		if checksum != 8063 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_payment_by_hash: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_onchain_payment()
		})
		if checksum != 38151 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_onchain_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_redeem_onchain_funds()
		})
		if checksum != 23808 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_redeem_onchain_funds: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_refund()
		})
		if checksum != 19067 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_refund: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_receive_onchain()
		})
		if checksum != 38436 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_receive_onchain: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_receive_payment()
		})
		if checksum != 65361 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_receive_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_recommended_fees()
		})
		if checksum != 46238 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_recommended_fees: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_redeem_onchain_funds()
		})
		if checksum != 25372 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_redeem_onchain_funds: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_redeem_swap()
		})
		if checksum != 31523 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_redeem_swap: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_refund()
		})
		if checksum != 28853 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_refund: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_register_webhook()
		})
		if checksum != 51745 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_register_webhook: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_report_issue()
		})
		if checksum != 20233 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_report_issue: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_rescan_swaps()
		})
		if checksum != 30273 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_rescan_swaps: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_send_payment()
		})
		if checksum != 21112 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_send_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_send_spontaneous_payment()
		})
		if checksum != 62139 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_send_spontaneous_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_set_payment_metadata()
		})
		if checksum != 64161 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_set_payment_metadata: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_sign_message()
		})
		if checksum != 27140 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_sign_message: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_sync()
		})
		if checksum != 37323 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_sync: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_unregister_webhook()
		})
		if checksum != 13931 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_unregister_webhook: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_withdraw_lnurl()
		})
		if checksum != 44837 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_withdraw_lnurl: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_eventlistener_on_event()
		})
		if checksum != 53633 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_eventlistener_on_event: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(_uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_logstream_log()
		})
		if checksum != 5129 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_logstream_log: UniFFI API checksum mismatch")
		}
	}
}

type FfiConverterUint8 struct{}

var FfiConverterUint8INSTANCE = FfiConverterUint8{}

func (FfiConverterUint8) Lower(value uint8) C.uint8_t {
	return C.uint8_t(value)
}

func (FfiConverterUint8) Write(writer io.Writer, value uint8) {
	writeUint8(writer, value)
}

func (FfiConverterUint8) Lift(value C.uint8_t) uint8 {
	return uint8(value)
}

func (FfiConverterUint8) Read(reader io.Reader) uint8 {
	return readUint8(reader)
}

type FfiDestroyerUint8 struct{}

func (FfiDestroyerUint8) Destroy(_ uint8) {}

type FfiConverterUint16 struct{}

var FfiConverterUint16INSTANCE = FfiConverterUint16{}

func (FfiConverterUint16) Lower(value uint16) C.uint16_t {
	return C.uint16_t(value)
}

func (FfiConverterUint16) Write(writer io.Writer, value uint16) {
	writeUint16(writer, value)
}

func (FfiConverterUint16) Lift(value C.uint16_t) uint16 {
	return uint16(value)
}

func (FfiConverterUint16) Read(reader io.Reader) uint16 {
	return readUint16(reader)
}

type FfiDestroyerUint16 struct{}

func (FfiDestroyerUint16) Destroy(_ uint16) {}

type FfiConverterUint32 struct{}

var FfiConverterUint32INSTANCE = FfiConverterUint32{}

func (FfiConverterUint32) Lower(value uint32) C.uint32_t {
	return C.uint32_t(value)
}

func (FfiConverterUint32) Write(writer io.Writer, value uint32) {
	writeUint32(writer, value)
}

func (FfiConverterUint32) Lift(value C.uint32_t) uint32 {
	return uint32(value)
}

func (FfiConverterUint32) Read(reader io.Reader) uint32 {
	return readUint32(reader)
}

type FfiDestroyerUint32 struct{}

func (FfiDestroyerUint32) Destroy(_ uint32) {}

type FfiConverterUint64 struct{}

var FfiConverterUint64INSTANCE = FfiConverterUint64{}

func (FfiConverterUint64) Lower(value uint64) C.uint64_t {
	return C.uint64_t(value)
}

func (FfiConverterUint64) Write(writer io.Writer, value uint64) {
	writeUint64(writer, value)
}

func (FfiConverterUint64) Lift(value C.uint64_t) uint64 {
	return uint64(value)
}

func (FfiConverterUint64) Read(reader io.Reader) uint64 {
	return readUint64(reader)
}

type FfiDestroyerUint64 struct{}

func (FfiDestroyerUint64) Destroy(_ uint64) {}

type FfiConverterInt64 struct{}

var FfiConverterInt64INSTANCE = FfiConverterInt64{}

func (FfiConverterInt64) Lower(value int64) C.int64_t {
	return C.int64_t(value)
}

func (FfiConverterInt64) Write(writer io.Writer, value int64) {
	writeInt64(writer, value)
}

func (FfiConverterInt64) Lift(value C.int64_t) int64 {
	return int64(value)
}

func (FfiConverterInt64) Read(reader io.Reader) int64 {
	return readInt64(reader)
}

type FfiDestroyerInt64 struct{}

func (FfiDestroyerInt64) Destroy(_ int64) {}

type FfiConverterFloat64 struct{}

var FfiConverterFloat64INSTANCE = FfiConverterFloat64{}

func (FfiConverterFloat64) Lower(value float64) C.double {
	return C.double(value)
}

func (FfiConverterFloat64) Write(writer io.Writer, value float64) {
	writeFloat64(writer, value)
}

func (FfiConverterFloat64) Lift(value C.double) float64 {
	return float64(value)
}

func (FfiConverterFloat64) Read(reader io.Reader) float64 {
	return readFloat64(reader)
}

type FfiDestroyerFloat64 struct{}

func (FfiDestroyerFloat64) Destroy(_ float64) {}

type FfiConverterBool struct{}

var FfiConverterBoolINSTANCE = FfiConverterBool{}

func (FfiConverterBool) Lower(value bool) C.int8_t {
	if value {
		return C.int8_t(1)
	}
	return C.int8_t(0)
}

func (FfiConverterBool) Write(writer io.Writer, value bool) {
	if value {
		writeInt8(writer, 1)
	} else {
		writeInt8(writer, 0)
	}
}

func (FfiConverterBool) Lift(value C.int8_t) bool {
	return value != 0
}

func (FfiConverterBool) Read(reader io.Reader) bool {
	return readInt8(reader) != 0
}

type FfiDestroyerBool struct{}

func (FfiDestroyerBool) Destroy(_ bool) {}

type FfiConverterString struct{}

var FfiConverterStringINSTANCE = FfiConverterString{}

func (FfiConverterString) Lift(rb RustBufferI) string {
	defer rb.Free()
	reader := rb.AsReader()
	b, err := io.ReadAll(reader)
	if err != nil {
		panic(fmt.Errorf("reading reader: %w", err))
	}
	return string(b)
}

func (FfiConverterString) Read(reader io.Reader) string {
	length := readInt32(reader)
	buffer := make([]byte, length)
	read_length, err := reader.Read(buffer)
	if err != nil {
		panic(err)
	}
	if read_length != int(length) {
		panic(fmt.Errorf("bad read length when reading string, expected %d, read %d", length, read_length))
	}
	return string(buffer)
}

func (FfiConverterString) Lower(value string) C.RustBuffer {
	return stringToRustBuffer(value)
}

func (FfiConverterString) Write(writer io.Writer, value string) {
	if len(value) > math.MaxInt32 {
		panic("String is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	write_length, err := io.WriteString(writer, value)
	if err != nil {
		panic(err)
	}
	if write_length != len(value) {
		panic(fmt.Errorf("bad write length when writing string, expected %d, written %d", len(value), write_length))
	}
}

type FfiDestroyerString struct{}

func (FfiDestroyerString) Destroy(_ string) {}

// Below is an implementation of synchronization requirements outlined in the link.
// https://github.com/mozilla/uniffi-rs/blob/0dc031132d9493ca812c3af6e7dd60ad2ea95bf0/uniffi_bindgen/src/bindings/kotlin/templates/ObjectRuntime.kt#L31

type FfiObject struct {
	pointer       unsafe.Pointer
	callCounter   atomic.Int64
	cloneFunction func(unsafe.Pointer, *C.RustCallStatus) unsafe.Pointer
	freeFunction  func(unsafe.Pointer, *C.RustCallStatus)
	destroyed     atomic.Bool
}

func newFfiObject(
	pointer unsafe.Pointer,
	cloneFunction func(unsafe.Pointer, *C.RustCallStatus) unsafe.Pointer,
	freeFunction func(unsafe.Pointer, *C.RustCallStatus),
) FfiObject {
	return FfiObject{
		pointer:       pointer,
		cloneFunction: cloneFunction,
		freeFunction:  freeFunction,
	}
}

func (ffiObject *FfiObject) incrementPointer(debugName string) unsafe.Pointer {
	for {
		counter := ffiObject.callCounter.Load()
		if counter <= -1 {
			panic(fmt.Errorf("%v object has already been destroyed", debugName))
		}
		if counter == math.MaxInt64 {
			panic(fmt.Errorf("%v object call counter would overflow", debugName))
		}
		if ffiObject.callCounter.CompareAndSwap(counter, counter+1) {
			break
		}
	}

	return rustCall(func(status *C.RustCallStatus) unsafe.Pointer {
		return ffiObject.cloneFunction(ffiObject.pointer, status)
	})
}

func (ffiObject *FfiObject) decrementPointer() {
	if ffiObject.callCounter.Add(-1) == -1 {
		ffiObject.freeRustArcPtr()
	}
}

func (ffiObject *FfiObject) destroy() {
	if ffiObject.destroyed.CompareAndSwap(false, true) {
		if ffiObject.callCounter.Add(-1) == -1 {
			ffiObject.freeRustArcPtr()
		}
	}
}

func (ffiObject *FfiObject) freeRustArcPtr() {
	rustCall(func(status *C.RustCallStatus) int32 {
		ffiObject.freeFunction(ffiObject.pointer, status)
		return 0
	})
}

type BlockingBreezServicesInterface interface {
	Backup() *SdkError
	BackupStatus() (BackupStatus, *SdkError)
	BuyBitcoin(req BuyBitcoinRequest) (BuyBitcoinResponse, *ReceiveOnchainError)
	CheckMessage(req CheckMessageRequest) (CheckMessageResponse, *SdkError)
	ClaimReverseSwap(lockupAddress string) *SdkError
	CloseLspChannels() *SdkError
	ConfigureNode(req ConfigureNodeRequest) *SdkError
	ConnectLsp(lspId string) *SdkError
	Disconnect() *SdkError
	ExecuteDevCommand(command string) (string, *SdkError)
	FetchFiatRates() ([]Rate, *SdkError)
	FetchLspInfo(lspId string) (*LspInformation, *SdkError)
	FetchReverseSwapFees(req ReverseSwapFeesRequest) (ReverseSwapPairInfo, *SdkError)
	GenerateDiagnosticData() (string, *SdkError)
	InProgressOnchainPayments() ([]ReverseSwapInfo, *SdkError)
	InProgressSwap() (*SwapInfo, *SdkError)
	ListFiatCurrencies() ([]FiatCurrency, *SdkError)
	ListLsps() ([]LspInformation, *SdkError)
	ListPayments(req ListPaymentsRequest) ([]Payment, *SdkError)
	ListRefundables() ([]SwapInfo, *SdkError)
	ListSwaps(req ListSwapsRequest) ([]SwapInfo, *SdkError)
	LnurlAuth(reqData LnUrlAuthRequestData) (LnUrlCallbackStatus, *LnUrlAuthError)
	LspId() (*string, *SdkError)
	LspInfo() (LspInformation, *SdkError)
	NodeCredentials() (*NodeCredentials, *SdkError)
	NodeInfo() (NodeState, *SdkError)
	OnchainPaymentLimits() (OnchainPaymentLimitsResponse, *SdkError)
	OpenChannelFee(req OpenChannelFeeRequest) (OpenChannelFeeResponse, *SdkError)
	PayLnurl(req LnUrlPayRequest) (LnUrlPayResult, *LnUrlPayError)
	PayOnchain(req PayOnchainRequest) (PayOnchainResponse, *SendOnchainError)
	PaymentByHash(hash string) (*Payment, *SdkError)
	PrepareOnchainPayment(req PrepareOnchainPaymentRequest) (PrepareOnchainPaymentResponse, *SendOnchainError)
	PrepareRedeemOnchainFunds(req PrepareRedeemOnchainFundsRequest) (PrepareRedeemOnchainFundsResponse, *RedeemOnchainError)
	PrepareRefund(req PrepareRefundRequest) (PrepareRefundResponse, *SdkError)
	ReceiveOnchain(req ReceiveOnchainRequest) (SwapInfo, *ReceiveOnchainError)
	ReceivePayment(req ReceivePaymentRequest) (ReceivePaymentResponse, *ReceivePaymentError)
	RecommendedFees() (RecommendedFees, *SdkError)
	RedeemOnchainFunds(req RedeemOnchainFundsRequest) (RedeemOnchainFundsResponse, *RedeemOnchainError)
	RedeemSwap(swapAddress string) *SdkError
	Refund(req RefundRequest) (RefundResponse, *SdkError)
	RegisterWebhook(webhookUrl string) *SdkError
	ReportIssue(req ReportIssueRequest) *SdkError
	RescanSwaps() *SdkError
	SendPayment(req SendPaymentRequest) (SendPaymentResponse, *SendPaymentError)
	SendSpontaneousPayment(req SendSpontaneousPaymentRequest) (SendPaymentResponse, *SendPaymentError)
	SetPaymentMetadata(hash string, metadata string) *SdkError
	SignMessage(req SignMessageRequest) (SignMessageResponse, *SdkError)
	Sync() *SdkError
	UnregisterWebhook(webhookUrl string) *SdkError
	WithdrawLnurl(request LnUrlWithdrawRequest) (LnUrlWithdrawResult, *LnUrlWithdrawError)
}
type BlockingBreezServices struct {
	ffiObject FfiObject
}

func (_self *BlockingBreezServices) Backup() *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_backup(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) BackupStatus() (BackupStatus, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_backup_status(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue BackupStatus
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterBackupStatusINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) BuyBitcoin(req BuyBitcoinRequest) (BuyBitcoinResponse, *ReceiveOnchainError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[ReceiveOnchainError](FfiConverterReceiveOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_buy_bitcoin(
				_pointer, FfiConverterBuyBitcoinRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue BuyBitcoinResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterBuyBitcoinResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) CheckMessage(req CheckMessageRequest) (CheckMessageResponse, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_check_message(
				_pointer, FfiConverterCheckMessageRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue CheckMessageResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterCheckMessageResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ClaimReverseSwap(lockupAddress string) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_claim_reverse_swap(
			_pointer, FfiConverterStringINSTANCE.Lower(lockupAddress), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) CloseLspChannels() *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_close_lsp_channels(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) ConfigureNode(req ConfigureNodeRequest) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_configure_node(
			_pointer, FfiConverterConfigureNodeRequestINSTANCE.Lower(req), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) ConnectLsp(lspId string) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_connect_lsp(
			_pointer, FfiConverterStringINSTANCE.Lower(lspId), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) Disconnect() *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_disconnect(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) ExecuteDevCommand(command string) (string, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_execute_dev_command(
				_pointer, FfiConverterStringINSTANCE.Lower(command), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterStringINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) FetchFiatRates() ([]Rate, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_fetch_fiat_rates(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []Rate
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceRateINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) FetchLspInfo(lspId string) (*LspInformation, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_fetch_lsp_info(
				_pointer, FfiConverterStringINSTANCE.Lower(lspId), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *LspInformation
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalLspInformationINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) FetchReverseSwapFees(req ReverseSwapFeesRequest) (ReverseSwapPairInfo, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_fetch_reverse_swap_fees(
				_pointer, FfiConverterReverseSwapFeesRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ReverseSwapPairInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterReverseSwapPairInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) GenerateDiagnosticData() (string, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_generate_diagnostic_data(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterStringINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) InProgressOnchainPayments() ([]ReverseSwapInfo, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_in_progress_onchain_payments(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []ReverseSwapInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceReverseSwapInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) InProgressSwap() (*SwapInfo, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_in_progress_swap(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *SwapInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalSwapInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ListFiatCurrencies() ([]FiatCurrency, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_list_fiat_currencies(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []FiatCurrency
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceFiatCurrencyINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ListLsps() ([]LspInformation, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_list_lsps(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []LspInformation
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceLspInformationINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ListPayments(req ListPaymentsRequest) ([]Payment, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_list_payments(
				_pointer, FfiConverterListPaymentsRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []Payment
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequencePaymentINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ListRefundables() ([]SwapInfo, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_list_refundables(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []SwapInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceSwapInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ListSwaps(req ListSwapsRequest) ([]SwapInfo, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_list_swaps(
				_pointer, FfiConverterListSwapsRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []SwapInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceSwapInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) LnurlAuth(reqData LnUrlAuthRequestData) (LnUrlCallbackStatus, *LnUrlAuthError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[LnUrlAuthError](FfiConverterLnUrlAuthError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_lnurl_auth(
				_pointer, FfiConverterLnUrlAuthRequestDataINSTANCE.Lower(reqData), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlCallbackStatus
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterLnUrlCallbackStatusINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) LspId() (*string, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_lsp_id(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalStringINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) LspInfo() (LspInformation, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_lsp_info(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LspInformation
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterLspInformationINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) NodeCredentials() (*NodeCredentials, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_node_credentials(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *NodeCredentials
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalNodeCredentialsINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) NodeInfo() (NodeState, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_node_info(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue NodeState
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterNodeStateINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) OnchainPaymentLimits() (OnchainPaymentLimitsResponse, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_onchain_payment_limits(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue OnchainPaymentLimitsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOnchainPaymentLimitsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) OpenChannelFee(req OpenChannelFeeRequest) (OpenChannelFeeResponse, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_open_channel_fee(
				_pointer, FfiConverterOpenChannelFeeRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue OpenChannelFeeResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOpenChannelFeeResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PayLnurl(req LnUrlPayRequest) (LnUrlPayResult, *LnUrlPayError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[LnUrlPayError](FfiConverterLnUrlPayError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_pay_lnurl(
				_pointer, FfiConverterLnUrlPayRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlPayResult
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterLnUrlPayResultINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PayOnchain(req PayOnchainRequest) (PayOnchainResponse, *SendOnchainError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SendOnchainError](FfiConverterSendOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_pay_onchain(
				_pointer, FfiConverterPayOnchainRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PayOnchainResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterPayOnchainResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PaymentByHash(hash string) (*Payment, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_payment_by_hash(
				_pointer, FfiConverterStringINSTANCE.Lower(hash), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *Payment
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalPaymentINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PrepareOnchainPayment(req PrepareOnchainPaymentRequest) (PrepareOnchainPaymentResponse, *SendOnchainError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SendOnchainError](FfiConverterSendOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_prepare_onchain_payment(
				_pointer, FfiConverterPrepareOnchainPaymentRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareOnchainPaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterPrepareOnchainPaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PrepareRedeemOnchainFunds(req PrepareRedeemOnchainFundsRequest) (PrepareRedeemOnchainFundsResponse, *RedeemOnchainError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[RedeemOnchainError](FfiConverterRedeemOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_prepare_redeem_onchain_funds(
				_pointer, FfiConverterPrepareRedeemOnchainFundsRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareRedeemOnchainFundsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterPrepareRedeemOnchainFundsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PrepareRefund(req PrepareRefundRequest) (PrepareRefundResponse, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_prepare_refund(
				_pointer, FfiConverterPrepareRefundRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareRefundResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterPrepareRefundResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ReceiveOnchain(req ReceiveOnchainRequest) (SwapInfo, *ReceiveOnchainError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[ReceiveOnchainError](FfiConverterReceiveOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_receive_onchain(
				_pointer, FfiConverterReceiveOnchainRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SwapInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSwapInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ReceivePayment(req ReceivePaymentRequest) (ReceivePaymentResponse, *ReceivePaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[ReceivePaymentError](FfiConverterReceivePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_receive_payment(
				_pointer, FfiConverterReceivePaymentRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ReceivePaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterReceivePaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) RecommendedFees() (RecommendedFees, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_recommended_fees(
				_pointer, _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue RecommendedFees
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterRecommendedFeesINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) RedeemOnchainFunds(req RedeemOnchainFundsRequest) (RedeemOnchainFundsResponse, *RedeemOnchainError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[RedeemOnchainError](FfiConverterRedeemOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_redeem_onchain_funds(
				_pointer, FfiConverterRedeemOnchainFundsRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue RedeemOnchainFundsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterRedeemOnchainFundsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) RedeemSwap(swapAddress string) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_redeem_swap(
			_pointer, FfiConverterStringINSTANCE.Lower(swapAddress), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) Refund(req RefundRequest) (RefundResponse, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_refund(
				_pointer, FfiConverterRefundRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue RefundResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterRefundResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) RegisterWebhook(webhookUrl string) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_register_webhook(
			_pointer, FfiConverterStringINSTANCE.Lower(webhookUrl), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) ReportIssue(req ReportIssueRequest) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_report_issue(
			_pointer, FfiConverterReportIssueRequestINSTANCE.Lower(req), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) RescanSwaps() *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_rescan_swaps(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) SendPayment(req SendPaymentRequest) (SendPaymentResponse, *SendPaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SendPaymentError](FfiConverterSendPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_send_payment(
				_pointer, FfiConverterSendPaymentRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SendPaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSendPaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) SendSpontaneousPayment(req SendSpontaneousPaymentRequest) (SendPaymentResponse, *SendPaymentError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SendPaymentError](FfiConverterSendPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_send_spontaneous_payment(
				_pointer, FfiConverterSendSpontaneousPaymentRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SendPaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSendPaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) SetPaymentMetadata(hash string, metadata string) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_set_payment_metadata(
			_pointer, FfiConverterStringINSTANCE.Lower(hash), FfiConverterStringINSTANCE.Lower(metadata), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) SignMessage(req SignMessageRequest) (SignMessageResponse, *SdkError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_sign_message(
				_pointer, FfiConverterSignMessageRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SignMessageResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSignMessageResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) Sync() *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_sync(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) UnregisterWebhook(webhookUrl string) *SdkError {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_unregister_webhook(
			_pointer, FfiConverterStringINSTANCE.Lower(webhookUrl), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) WithdrawLnurl(request LnUrlWithdrawRequest) (LnUrlWithdrawResult, *LnUrlWithdrawError) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError[LnUrlWithdrawError](FfiConverterLnUrlWithdrawError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_withdraw_lnurl(
				_pointer, FfiConverterLnUrlWithdrawRequestINSTANCE.Lower(request), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlWithdrawResult
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterLnUrlWithdrawResultINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}
func (object *BlockingBreezServices) Destroy() {
	runtime.SetFinalizer(object, nil)
	object.ffiObject.destroy()
}

type FfiConverterBlockingBreezServices struct{}

var FfiConverterBlockingBreezServicesINSTANCE = FfiConverterBlockingBreezServices{}

func (c FfiConverterBlockingBreezServices) Lift(pointer unsafe.Pointer) *BlockingBreezServices {
	result := &BlockingBreezServices{
		newFfiObject(
			pointer,
			func(pointer unsafe.Pointer, status *C.RustCallStatus) unsafe.Pointer {
				return C.uniffi_breez_sdk_bindings_fn_clone_blockingbreezservices(pointer, status)
			},
			func(pointer unsafe.Pointer, status *C.RustCallStatus) {
				C.uniffi_breez_sdk_bindings_fn_free_blockingbreezservices(pointer, status)
			},
		),
	}
	runtime.SetFinalizer(result, (*BlockingBreezServices).Destroy)
	return result
}

func (c FfiConverterBlockingBreezServices) Read(reader io.Reader) *BlockingBreezServices {
	return c.Lift(unsafe.Pointer(uintptr(readUint64(reader))))
}

func (c FfiConverterBlockingBreezServices) Lower(value *BlockingBreezServices) unsafe.Pointer {
	// TODO: this is bad - all synchronization from ObjectRuntime.go is discarded here,
	// because the pointer will be decremented immediately after this function returns,
	// and someone will be left holding onto a non-locked pointer.
	pointer := value.ffiObject.incrementPointer("*BlockingBreezServices")
	defer value.ffiObject.decrementPointer()
	return pointer

}

func (c FfiConverterBlockingBreezServices) Write(writer io.Writer, value *BlockingBreezServices) {
	writeUint64(writer, uint64(uintptr(c.Lower(value))))
}

type FfiDestroyerBlockingBreezServices struct{}

func (_ FfiDestroyerBlockingBreezServices) Destroy(value *BlockingBreezServices) {
	value.Destroy()
}

type AesSuccessActionDataDecrypted struct {
	Description string
	Plaintext   string
}

func (r *AesSuccessActionDataDecrypted) Destroy() {
	FfiDestroyerString{}.Destroy(r.Description)
	FfiDestroyerString{}.Destroy(r.Plaintext)
}

type FfiConverterAesSuccessActionDataDecrypted struct{}

var FfiConverterAesSuccessActionDataDecryptedINSTANCE = FfiConverterAesSuccessActionDataDecrypted{}

func (c FfiConverterAesSuccessActionDataDecrypted) Lift(rb RustBufferI) AesSuccessActionDataDecrypted {
	return LiftFromRustBuffer[AesSuccessActionDataDecrypted](c, rb)
}

func (c FfiConverterAesSuccessActionDataDecrypted) Read(reader io.Reader) AesSuccessActionDataDecrypted {
	return AesSuccessActionDataDecrypted{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterAesSuccessActionDataDecrypted) Lower(value AesSuccessActionDataDecrypted) C.RustBuffer {
	return LowerIntoRustBuffer[AesSuccessActionDataDecrypted](c, value)
}

func (c FfiConverterAesSuccessActionDataDecrypted) Write(writer io.Writer, value AesSuccessActionDataDecrypted) {
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.Plaintext)
}

type FfiDestroyerAesSuccessActionDataDecrypted struct{}

func (_ FfiDestroyerAesSuccessActionDataDecrypted) Destroy(value AesSuccessActionDataDecrypted) {
	value.Destroy()
}

type BackupFailedData struct {
	Error string
}

func (r *BackupFailedData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Error)
}

type FfiConverterBackupFailedData struct{}

var FfiConverterBackupFailedDataINSTANCE = FfiConverterBackupFailedData{}

func (c FfiConverterBackupFailedData) Lift(rb RustBufferI) BackupFailedData {
	return LiftFromRustBuffer[BackupFailedData](c, rb)
}

func (c FfiConverterBackupFailedData) Read(reader io.Reader) BackupFailedData {
	return BackupFailedData{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterBackupFailedData) Lower(value BackupFailedData) C.RustBuffer {
	return LowerIntoRustBuffer[BackupFailedData](c, value)
}

func (c FfiConverterBackupFailedData) Write(writer io.Writer, value BackupFailedData) {
	FfiConverterStringINSTANCE.Write(writer, value.Error)
}

type FfiDestroyerBackupFailedData struct{}

func (_ FfiDestroyerBackupFailedData) Destroy(value BackupFailedData) {
	value.Destroy()
}

type BackupStatus struct {
	BackedUp       bool
	LastBackupTime *uint64
}

func (r *BackupStatus) Destroy() {
	FfiDestroyerBool{}.Destroy(r.BackedUp)
	FfiDestroyerOptionalUint64{}.Destroy(r.LastBackupTime)
}

type FfiConverterBackupStatus struct{}

var FfiConverterBackupStatusINSTANCE = FfiConverterBackupStatus{}

func (c FfiConverterBackupStatus) Lift(rb RustBufferI) BackupStatus {
	return LiftFromRustBuffer[BackupStatus](c, rb)
}

func (c FfiConverterBackupStatus) Read(reader io.Reader) BackupStatus {
	return BackupStatus{
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterBackupStatus) Lower(value BackupStatus) C.RustBuffer {
	return LowerIntoRustBuffer[BackupStatus](c, value)
}

func (c FfiConverterBackupStatus) Write(writer io.Writer, value BackupStatus) {
	FfiConverterBoolINSTANCE.Write(writer, value.BackedUp)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.LastBackupTime)
}

type FfiDestroyerBackupStatus struct{}

func (_ FfiDestroyerBackupStatus) Destroy(value BackupStatus) {
	value.Destroy()
}

type BitcoinAddressData struct {
	Address   string
	Network   Network
	AmountSat *uint64
	Label     *string
	Message   *string
}

func (r *BitcoinAddressData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Address)
	FfiDestroyerNetwork{}.Destroy(r.Network)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountSat)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
	FfiDestroyerOptionalString{}.Destroy(r.Message)
}

type FfiConverterBitcoinAddressData struct{}

var FfiConverterBitcoinAddressDataINSTANCE = FfiConverterBitcoinAddressData{}

func (c FfiConverterBitcoinAddressData) Lift(rb RustBufferI) BitcoinAddressData {
	return LiftFromRustBuffer[BitcoinAddressData](c, rb)
}

func (c FfiConverterBitcoinAddressData) Read(reader io.Reader) BitcoinAddressData {
	return BitcoinAddressData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterNetworkINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterBitcoinAddressData) Lower(value BitcoinAddressData) C.RustBuffer {
	return LowerIntoRustBuffer[BitcoinAddressData](c, value)
}

func (c FfiConverterBitcoinAddressData) Write(writer io.Writer, value BitcoinAddressData) {
	FfiConverterStringINSTANCE.Write(writer, value.Address)
	FfiConverterNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerBitcoinAddressData struct{}

func (_ FfiDestroyerBitcoinAddressData) Destroy(value BitcoinAddressData) {
	value.Destroy()
}

type BuyBitcoinRequest struct {
	Provider         BuyBitcoinProvider
	OpeningFeeParams *OpeningFeeParams
	RedirectUrl      *string
}

func (r *BuyBitcoinRequest) Destroy() {
	FfiDestroyerBuyBitcoinProvider{}.Destroy(r.Provider)
	FfiDestroyerOptionalOpeningFeeParams{}.Destroy(r.OpeningFeeParams)
	FfiDestroyerOptionalString{}.Destroy(r.RedirectUrl)
}

type FfiConverterBuyBitcoinRequest struct{}

var FfiConverterBuyBitcoinRequestINSTANCE = FfiConverterBuyBitcoinRequest{}

func (c FfiConverterBuyBitcoinRequest) Lift(rb RustBufferI) BuyBitcoinRequest {
	return LiftFromRustBuffer[BuyBitcoinRequest](c, rb)
}

func (c FfiConverterBuyBitcoinRequest) Read(reader io.Reader) BuyBitcoinRequest {
	return BuyBitcoinRequest{
		FfiConverterBuyBitcoinProviderINSTANCE.Read(reader),
		FfiConverterOptionalOpeningFeeParamsINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterBuyBitcoinRequest) Lower(value BuyBitcoinRequest) C.RustBuffer {
	return LowerIntoRustBuffer[BuyBitcoinRequest](c, value)
}

func (c FfiConverterBuyBitcoinRequest) Write(writer io.Writer, value BuyBitcoinRequest) {
	FfiConverterBuyBitcoinProviderINSTANCE.Write(writer, value.Provider)
	FfiConverterOptionalOpeningFeeParamsINSTANCE.Write(writer, value.OpeningFeeParams)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.RedirectUrl)
}

type FfiDestroyerBuyBitcoinRequest struct{}

func (_ FfiDestroyerBuyBitcoinRequest) Destroy(value BuyBitcoinRequest) {
	value.Destroy()
}

type BuyBitcoinResponse struct {
	Url              string
	OpeningFeeParams *OpeningFeeParams
}

func (r *BuyBitcoinResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Url)
	FfiDestroyerOptionalOpeningFeeParams{}.Destroy(r.OpeningFeeParams)
}

type FfiConverterBuyBitcoinResponse struct{}

var FfiConverterBuyBitcoinResponseINSTANCE = FfiConverterBuyBitcoinResponse{}

func (c FfiConverterBuyBitcoinResponse) Lift(rb RustBufferI) BuyBitcoinResponse {
	return LiftFromRustBuffer[BuyBitcoinResponse](c, rb)
}

func (c FfiConverterBuyBitcoinResponse) Read(reader io.Reader) BuyBitcoinResponse {
	return BuyBitcoinResponse{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalOpeningFeeParamsINSTANCE.Read(reader),
	}
}

func (c FfiConverterBuyBitcoinResponse) Lower(value BuyBitcoinResponse) C.RustBuffer {
	return LowerIntoRustBuffer[BuyBitcoinResponse](c, value)
}

func (c FfiConverterBuyBitcoinResponse) Write(writer io.Writer, value BuyBitcoinResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Url)
	FfiConverterOptionalOpeningFeeParamsINSTANCE.Write(writer, value.OpeningFeeParams)
}

type FfiDestroyerBuyBitcoinResponse struct{}

func (_ FfiDestroyerBuyBitcoinResponse) Destroy(value BuyBitcoinResponse) {
	value.Destroy()
}

type CheckMessageRequest struct {
	Message   string
	Pubkey    string
	Signature string
}

func (r *CheckMessageRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Message)
	FfiDestroyerString{}.Destroy(r.Pubkey)
	FfiDestroyerString{}.Destroy(r.Signature)
}

type FfiConverterCheckMessageRequest struct{}

var FfiConverterCheckMessageRequestINSTANCE = FfiConverterCheckMessageRequest{}

func (c FfiConverterCheckMessageRequest) Lift(rb RustBufferI) CheckMessageRequest {
	return LiftFromRustBuffer[CheckMessageRequest](c, rb)
}

func (c FfiConverterCheckMessageRequest) Read(reader io.Reader) CheckMessageRequest {
	return CheckMessageRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterCheckMessageRequest) Lower(value CheckMessageRequest) C.RustBuffer {
	return LowerIntoRustBuffer[CheckMessageRequest](c, value)
}

func (c FfiConverterCheckMessageRequest) Write(writer io.Writer, value CheckMessageRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
	FfiConverterStringINSTANCE.Write(writer, value.Pubkey)
	FfiConverterStringINSTANCE.Write(writer, value.Signature)
}

type FfiDestroyerCheckMessageRequest struct{}

func (_ FfiDestroyerCheckMessageRequest) Destroy(value CheckMessageRequest) {
	value.Destroy()
}

type CheckMessageResponse struct {
	IsValid bool
}

func (r *CheckMessageResponse) Destroy() {
	FfiDestroyerBool{}.Destroy(r.IsValid)
}

type FfiConverterCheckMessageResponse struct{}

var FfiConverterCheckMessageResponseINSTANCE = FfiConverterCheckMessageResponse{}

func (c FfiConverterCheckMessageResponse) Lift(rb RustBufferI) CheckMessageResponse {
	return LiftFromRustBuffer[CheckMessageResponse](c, rb)
}

func (c FfiConverterCheckMessageResponse) Read(reader io.Reader) CheckMessageResponse {
	return CheckMessageResponse{
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterCheckMessageResponse) Lower(value CheckMessageResponse) C.RustBuffer {
	return LowerIntoRustBuffer[CheckMessageResponse](c, value)
}

func (c FfiConverterCheckMessageResponse) Write(writer io.Writer, value CheckMessageResponse) {
	FfiConverterBoolINSTANCE.Write(writer, value.IsValid)
}

type FfiDestroyerCheckMessageResponse struct{}

func (_ FfiDestroyerCheckMessageResponse) Destroy(value CheckMessageResponse) {
	value.Destroy()
}

type ClosedChannelPaymentDetails struct {
	State          ChannelState
	FundingTxid    string
	ShortChannelId *string
	ClosingTxid    *string
}

func (r *ClosedChannelPaymentDetails) Destroy() {
	FfiDestroyerChannelState{}.Destroy(r.State)
	FfiDestroyerString{}.Destroy(r.FundingTxid)
	FfiDestroyerOptionalString{}.Destroy(r.ShortChannelId)
	FfiDestroyerOptionalString{}.Destroy(r.ClosingTxid)
}

type FfiConverterClosedChannelPaymentDetails struct{}

var FfiConverterClosedChannelPaymentDetailsINSTANCE = FfiConverterClosedChannelPaymentDetails{}

func (c FfiConverterClosedChannelPaymentDetails) Lift(rb RustBufferI) ClosedChannelPaymentDetails {
	return LiftFromRustBuffer[ClosedChannelPaymentDetails](c, rb)
}

func (c FfiConverterClosedChannelPaymentDetails) Read(reader io.Reader) ClosedChannelPaymentDetails {
	return ClosedChannelPaymentDetails{
		FfiConverterChannelStateINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterClosedChannelPaymentDetails) Lower(value ClosedChannelPaymentDetails) C.RustBuffer {
	return LowerIntoRustBuffer[ClosedChannelPaymentDetails](c, value)
}

func (c FfiConverterClosedChannelPaymentDetails) Write(writer io.Writer, value ClosedChannelPaymentDetails) {
	FfiConverterChannelStateINSTANCE.Write(writer, value.State)
	FfiConverterStringINSTANCE.Write(writer, value.FundingTxid)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.ShortChannelId)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.ClosingTxid)
}

type FfiDestroyerClosedChannelPaymentDetails struct{}

func (_ FfiDestroyerClosedChannelPaymentDetails) Destroy(value ClosedChannelPaymentDetails) {
	value.Destroy()
}

type Config struct {
	Breezserver       string
	ChainnotifierUrl  string
	MempoolspaceUrl   *string
	WorkingDir        string
	Network           Network
	PaymentTimeoutSec uint32
	DefaultLspId      *string
	ApiKey            *string
	MaxfeePercent     float64
	ExemptfeeMsat     uint64
	NodeConfig        NodeConfig
}

func (r *Config) Destroy() {
	FfiDestroyerString{}.Destroy(r.Breezserver)
	FfiDestroyerString{}.Destroy(r.ChainnotifierUrl)
	FfiDestroyerOptionalString{}.Destroy(r.MempoolspaceUrl)
	FfiDestroyerString{}.Destroy(r.WorkingDir)
	FfiDestroyerNetwork{}.Destroy(r.Network)
	FfiDestroyerUint32{}.Destroy(r.PaymentTimeoutSec)
	FfiDestroyerOptionalString{}.Destroy(r.DefaultLspId)
	FfiDestroyerOptionalString{}.Destroy(r.ApiKey)
	FfiDestroyerFloat64{}.Destroy(r.MaxfeePercent)
	FfiDestroyerUint64{}.Destroy(r.ExemptfeeMsat)
	FfiDestroyerNodeConfig{}.Destroy(r.NodeConfig)
}

type FfiConverterConfig struct{}

var FfiConverterConfigINSTANCE = FfiConverterConfig{}

func (c FfiConverterConfig) Lift(rb RustBufferI) Config {
	return LiftFromRustBuffer[Config](c, rb)
}

func (c FfiConverterConfig) Read(reader io.Reader) Config {
	return Config{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterNetworkINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterFloat64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterNodeConfigINSTANCE.Read(reader),
	}
}

func (c FfiConverterConfig) Lower(value Config) C.RustBuffer {
	return LowerIntoRustBuffer[Config](c, value)
}

func (c FfiConverterConfig) Write(writer io.Writer, value Config) {
	FfiConverterStringINSTANCE.Write(writer, value.Breezserver)
	FfiConverterStringINSTANCE.Write(writer, value.ChainnotifierUrl)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.MempoolspaceUrl)
	FfiConverterStringINSTANCE.Write(writer, value.WorkingDir)
	FfiConverterNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterUint32INSTANCE.Write(writer, value.PaymentTimeoutSec)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.DefaultLspId)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.ApiKey)
	FfiConverterFloat64INSTANCE.Write(writer, value.MaxfeePercent)
	FfiConverterUint64INSTANCE.Write(writer, value.ExemptfeeMsat)
	FfiConverterNodeConfigINSTANCE.Write(writer, value.NodeConfig)
}

type FfiDestroyerConfig struct{}

func (_ FfiDestroyerConfig) Destroy(value Config) {
	value.Destroy()
}

type ConfigureNodeRequest struct {
	CloseToAddress *string
}

func (r *ConfigureNodeRequest) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.CloseToAddress)
}

type FfiConverterConfigureNodeRequest struct{}

var FfiConverterConfigureNodeRequestINSTANCE = FfiConverterConfigureNodeRequest{}

func (c FfiConverterConfigureNodeRequest) Lift(rb RustBufferI) ConfigureNodeRequest {
	return LiftFromRustBuffer[ConfigureNodeRequest](c, rb)
}

func (c FfiConverterConfigureNodeRequest) Read(reader io.Reader) ConfigureNodeRequest {
	return ConfigureNodeRequest{
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterConfigureNodeRequest) Lower(value ConfigureNodeRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ConfigureNodeRequest](c, value)
}

func (c FfiConverterConfigureNodeRequest) Write(writer io.Writer, value ConfigureNodeRequest) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.CloseToAddress)
}

type FfiDestroyerConfigureNodeRequest struct{}

func (_ FfiDestroyerConfigureNodeRequest) Destroy(value ConfigureNodeRequest) {
	value.Destroy()
}

type ConnectRequest struct {
	Config      Config
	Seed        []uint8
	RestoreOnly *bool
}

func (r *ConnectRequest) Destroy() {
	FfiDestroyerConfig{}.Destroy(r.Config)
	FfiDestroyerSequenceUint8{}.Destroy(r.Seed)
	FfiDestroyerOptionalBool{}.Destroy(r.RestoreOnly)
}

type FfiConverterConnectRequest struct{}

var FfiConverterConnectRequestINSTANCE = FfiConverterConnectRequest{}

func (c FfiConverterConnectRequest) Lift(rb RustBufferI) ConnectRequest {
	return LiftFromRustBuffer[ConnectRequest](c, rb)
}

func (c FfiConverterConnectRequest) Read(reader io.Reader) ConnectRequest {
	return ConnectRequest{
		FfiConverterConfigINSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterConnectRequest) Lower(value ConnectRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ConnectRequest](c, value)
}

func (c FfiConverterConnectRequest) Write(writer io.Writer, value ConnectRequest) {
	FfiConverterConfigINSTANCE.Write(writer, value.Config)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Seed)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.RestoreOnly)
}

type FfiDestroyerConnectRequest struct{}

func (_ FfiDestroyerConnectRequest) Destroy(value ConnectRequest) {
	value.Destroy()
}

type CurrencyInfo struct {
	Name            string
	FractionSize    uint32
	Spacing         *uint32
	Symbol          *Symbol
	UniqSymbol      *Symbol
	LocalizedName   []LocalizedName
	LocaleOverrides []LocaleOverrides
}

func (r *CurrencyInfo) Destroy() {
	FfiDestroyerString{}.Destroy(r.Name)
	FfiDestroyerUint32{}.Destroy(r.FractionSize)
	FfiDestroyerOptionalUint32{}.Destroy(r.Spacing)
	FfiDestroyerOptionalSymbol{}.Destroy(r.Symbol)
	FfiDestroyerOptionalSymbol{}.Destroy(r.UniqSymbol)
	FfiDestroyerSequenceLocalizedName{}.Destroy(r.LocalizedName)
	FfiDestroyerSequenceLocaleOverrides{}.Destroy(r.LocaleOverrides)
}

type FfiConverterCurrencyInfo struct{}

var FfiConverterCurrencyInfoINSTANCE = FfiConverterCurrencyInfo{}

func (c FfiConverterCurrencyInfo) Lift(rb RustBufferI) CurrencyInfo {
	return LiftFromRustBuffer[CurrencyInfo](c, rb)
}

func (c FfiConverterCurrencyInfo) Read(reader io.Reader) CurrencyInfo {
	return CurrencyInfo{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalSymbolINSTANCE.Read(reader),
		FfiConverterOptionalSymbolINSTANCE.Read(reader),
		FfiConverterSequenceLocalizedNameINSTANCE.Read(reader),
		FfiConverterSequenceLocaleOverridesINSTANCE.Read(reader),
	}
}

func (c FfiConverterCurrencyInfo) Lower(value CurrencyInfo) C.RustBuffer {
	return LowerIntoRustBuffer[CurrencyInfo](c, value)
}

func (c FfiConverterCurrencyInfo) Write(writer io.Writer, value CurrencyInfo) {
	FfiConverterStringINSTANCE.Write(writer, value.Name)
	FfiConverterUint32INSTANCE.Write(writer, value.FractionSize)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Spacing)
	FfiConverterOptionalSymbolINSTANCE.Write(writer, value.Symbol)
	FfiConverterOptionalSymbolINSTANCE.Write(writer, value.UniqSymbol)
	FfiConverterSequenceLocalizedNameINSTANCE.Write(writer, value.LocalizedName)
	FfiConverterSequenceLocaleOverridesINSTANCE.Write(writer, value.LocaleOverrides)
}

type FfiDestroyerCurrencyInfo struct{}

func (_ FfiDestroyerCurrencyInfo) Destroy(value CurrencyInfo) {
	value.Destroy()
}

type FiatCurrency struct {
	Id   string
	Info CurrencyInfo
}

func (r *FiatCurrency) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerCurrencyInfo{}.Destroy(r.Info)
}

type FfiConverterFiatCurrency struct{}

var FfiConverterFiatCurrencyINSTANCE = FfiConverterFiatCurrency{}

func (c FfiConverterFiatCurrency) Lift(rb RustBufferI) FiatCurrency {
	return LiftFromRustBuffer[FiatCurrency](c, rb)
}

func (c FfiConverterFiatCurrency) Read(reader io.Reader) FiatCurrency {
	return FiatCurrency{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterCurrencyInfoINSTANCE.Read(reader),
	}
}

func (c FfiConverterFiatCurrency) Lower(value FiatCurrency) C.RustBuffer {
	return LowerIntoRustBuffer[FiatCurrency](c, value)
}

func (c FfiConverterFiatCurrency) Write(writer io.Writer, value FiatCurrency) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterCurrencyInfoINSTANCE.Write(writer, value.Info)
}

type FfiDestroyerFiatCurrency struct{}

func (_ FfiDestroyerFiatCurrency) Destroy(value FiatCurrency) {
	value.Destroy()
}

type GreenlightCredentials struct {
	DeveloperKey  []uint8
	DeveloperCert []uint8
}

func (r *GreenlightCredentials) Destroy() {
	FfiDestroyerSequenceUint8{}.Destroy(r.DeveloperKey)
	FfiDestroyerSequenceUint8{}.Destroy(r.DeveloperCert)
}

type FfiConverterGreenlightCredentials struct{}

var FfiConverterGreenlightCredentialsINSTANCE = FfiConverterGreenlightCredentials{}

func (c FfiConverterGreenlightCredentials) Lift(rb RustBufferI) GreenlightCredentials {
	return LiftFromRustBuffer[GreenlightCredentials](c, rb)
}

func (c FfiConverterGreenlightCredentials) Read(reader io.Reader) GreenlightCredentials {
	return GreenlightCredentials{
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
	}
}

func (c FfiConverterGreenlightCredentials) Lower(value GreenlightCredentials) C.RustBuffer {
	return LowerIntoRustBuffer[GreenlightCredentials](c, value)
}

func (c FfiConverterGreenlightCredentials) Write(writer io.Writer, value GreenlightCredentials) {
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.DeveloperKey)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.DeveloperCert)
}

type FfiDestroyerGreenlightCredentials struct{}

func (_ FfiDestroyerGreenlightCredentials) Destroy(value GreenlightCredentials) {
	value.Destroy()
}

type GreenlightDeviceCredentials struct {
	Device []uint8
}

func (r *GreenlightDeviceCredentials) Destroy() {
	FfiDestroyerSequenceUint8{}.Destroy(r.Device)
}

type FfiConverterGreenlightDeviceCredentials struct{}

var FfiConverterGreenlightDeviceCredentialsINSTANCE = FfiConverterGreenlightDeviceCredentials{}

func (c FfiConverterGreenlightDeviceCredentials) Lift(rb RustBufferI) GreenlightDeviceCredentials {
	return LiftFromRustBuffer[GreenlightDeviceCredentials](c, rb)
}

func (c FfiConverterGreenlightDeviceCredentials) Read(reader io.Reader) GreenlightDeviceCredentials {
	return GreenlightDeviceCredentials{
		FfiConverterSequenceUint8INSTANCE.Read(reader),
	}
}

func (c FfiConverterGreenlightDeviceCredentials) Lower(value GreenlightDeviceCredentials) C.RustBuffer {
	return LowerIntoRustBuffer[GreenlightDeviceCredentials](c, value)
}

func (c FfiConverterGreenlightDeviceCredentials) Write(writer io.Writer, value GreenlightDeviceCredentials) {
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Device)
}

type FfiDestroyerGreenlightDeviceCredentials struct{}

func (_ FfiDestroyerGreenlightDeviceCredentials) Destroy(value GreenlightDeviceCredentials) {
	value.Destroy()
}

type GreenlightNodeConfig struct {
	PartnerCredentials *GreenlightCredentials
	InviteCode         *string
}

func (r *GreenlightNodeConfig) Destroy() {
	FfiDestroyerOptionalGreenlightCredentials{}.Destroy(r.PartnerCredentials)
	FfiDestroyerOptionalString{}.Destroy(r.InviteCode)
}

type FfiConverterGreenlightNodeConfig struct{}

var FfiConverterGreenlightNodeConfigINSTANCE = FfiConverterGreenlightNodeConfig{}

func (c FfiConverterGreenlightNodeConfig) Lift(rb RustBufferI) GreenlightNodeConfig {
	return LiftFromRustBuffer[GreenlightNodeConfig](c, rb)
}

func (c FfiConverterGreenlightNodeConfig) Read(reader io.Reader) GreenlightNodeConfig {
	return GreenlightNodeConfig{
		FfiConverterOptionalGreenlightCredentialsINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterGreenlightNodeConfig) Lower(value GreenlightNodeConfig) C.RustBuffer {
	return LowerIntoRustBuffer[GreenlightNodeConfig](c, value)
}

func (c FfiConverterGreenlightNodeConfig) Write(writer io.Writer, value GreenlightNodeConfig) {
	FfiConverterOptionalGreenlightCredentialsINSTANCE.Write(writer, value.PartnerCredentials)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.InviteCode)
}

type FfiDestroyerGreenlightNodeConfig struct{}

func (_ FfiDestroyerGreenlightNodeConfig) Destroy(value GreenlightNodeConfig) {
	value.Destroy()
}

type InvoicePaidDetails struct {
	PaymentHash string
	Bolt11      string
	Payment     *Payment
}

func (r *InvoicePaidDetails) Destroy() {
	FfiDestroyerString{}.Destroy(r.PaymentHash)
	FfiDestroyerString{}.Destroy(r.Bolt11)
	FfiDestroyerOptionalPayment{}.Destroy(r.Payment)
}

type FfiConverterInvoicePaidDetails struct{}

var FfiConverterInvoicePaidDetailsINSTANCE = FfiConverterInvoicePaidDetails{}

func (c FfiConverterInvoicePaidDetails) Lift(rb RustBufferI) InvoicePaidDetails {
	return LiftFromRustBuffer[InvoicePaidDetails](c, rb)
}

func (c FfiConverterInvoicePaidDetails) Read(reader io.Reader) InvoicePaidDetails {
	return InvoicePaidDetails{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalPaymentINSTANCE.Read(reader),
	}
}

func (c FfiConverterInvoicePaidDetails) Lower(value InvoicePaidDetails) C.RustBuffer {
	return LowerIntoRustBuffer[InvoicePaidDetails](c, value)
}

func (c FfiConverterInvoicePaidDetails) Write(writer io.Writer, value InvoicePaidDetails) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterOptionalPaymentINSTANCE.Write(writer, value.Payment)
}

type FfiDestroyerInvoicePaidDetails struct{}

func (_ FfiDestroyerInvoicePaidDetails) Destroy(value InvoicePaidDetails) {
	value.Destroy()
}

type LnInvoice struct {
	Bolt11                  string
	Network                 Network
	PayeePubkey             string
	PaymentHash             string
	Description             *string
	DescriptionHash         *string
	AmountMsat              *uint64
	Timestamp               uint64
	Expiry                  uint64
	RoutingHints            []RouteHint
	PaymentSecret           []uint8
	MinFinalCltvExpiryDelta uint64
}

func (r *LnInvoice) Destroy() {
	FfiDestroyerString{}.Destroy(r.Bolt11)
	FfiDestroyerNetwork{}.Destroy(r.Network)
	FfiDestroyerString{}.Destroy(r.PayeePubkey)
	FfiDestroyerString{}.Destroy(r.PaymentHash)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerOptionalString{}.Destroy(r.DescriptionHash)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerUint64{}.Destroy(r.Timestamp)
	FfiDestroyerUint64{}.Destroy(r.Expiry)
	FfiDestroyerSequenceRouteHint{}.Destroy(r.RoutingHints)
	FfiDestroyerSequenceUint8{}.Destroy(r.PaymentSecret)
	FfiDestroyerUint64{}.Destroy(r.MinFinalCltvExpiryDelta)
}

type FfiConverterLnInvoice struct{}

var FfiConverterLnInvoiceINSTANCE = FfiConverterLnInvoice{}

func (c FfiConverterLnInvoice) Lift(rb RustBufferI) LnInvoice {
	return LiftFromRustBuffer[LnInvoice](c, rb)
}

func (c FfiConverterLnInvoice) Read(reader io.Reader) LnInvoice {
	return LnInvoice{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterNetworkINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSequenceRouteHintINSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterLnInvoice) Lower(value LnInvoice) C.RustBuffer {
	return LowerIntoRustBuffer[LnInvoice](c, value)
}

func (c FfiConverterLnInvoice) Write(writer io.Writer, value LnInvoice) {
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterStringINSTANCE.Write(writer, value.PayeePubkey)
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.DescriptionHash)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.Timestamp)
	FfiConverterUint64INSTANCE.Write(writer, value.Expiry)
	FfiConverterSequenceRouteHintINSTANCE.Write(writer, value.RoutingHints)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.PaymentSecret)
	FfiConverterUint64INSTANCE.Write(writer, value.MinFinalCltvExpiryDelta)
}

type FfiDestroyerLnInvoice struct{}

func (_ FfiDestroyerLnInvoice) Destroy(value LnInvoice) {
	value.Destroy()
}

type ListPaymentsRequest struct {
	Filters         *[]PaymentTypeFilter
	MetadataFilters *[]MetadataFilter
	FromTimestamp   *int64
	ToTimestamp     *int64
	IncludeFailures *bool
	Offset          *uint32
	Limit           *uint32
}

func (r *ListPaymentsRequest) Destroy() {
	FfiDestroyerOptionalSequencePaymentTypeFilter{}.Destroy(r.Filters)
	FfiDestroyerOptionalSequenceMetadataFilter{}.Destroy(r.MetadataFilters)
	FfiDestroyerOptionalInt64{}.Destroy(r.FromTimestamp)
	FfiDestroyerOptionalInt64{}.Destroy(r.ToTimestamp)
	FfiDestroyerOptionalBool{}.Destroy(r.IncludeFailures)
	FfiDestroyerOptionalUint32{}.Destroy(r.Offset)
	FfiDestroyerOptionalUint32{}.Destroy(r.Limit)
}

type FfiConverterListPaymentsRequest struct{}

var FfiConverterListPaymentsRequestINSTANCE = FfiConverterListPaymentsRequest{}

func (c FfiConverterListPaymentsRequest) Lift(rb RustBufferI) ListPaymentsRequest {
	return LiftFromRustBuffer[ListPaymentsRequest](c, rb)
}

func (c FfiConverterListPaymentsRequest) Read(reader io.Reader) ListPaymentsRequest {
	return ListPaymentsRequest{
		FfiConverterOptionalSequencePaymentTypeFilterINSTANCE.Read(reader),
		FfiConverterOptionalSequenceMetadataFilterINSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterListPaymentsRequest) Lower(value ListPaymentsRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ListPaymentsRequest](c, value)
}

func (c FfiConverterListPaymentsRequest) Write(writer io.Writer, value ListPaymentsRequest) {
	FfiConverterOptionalSequencePaymentTypeFilterINSTANCE.Write(writer, value.Filters)
	FfiConverterOptionalSequenceMetadataFilterINSTANCE.Write(writer, value.MetadataFilters)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.FromTimestamp)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.ToTimestamp)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.IncludeFailures)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Offset)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Limit)
}

type FfiDestroyerListPaymentsRequest struct{}

func (_ FfiDestroyerListPaymentsRequest) Destroy(value ListPaymentsRequest) {
	value.Destroy()
}

type ListSwapsRequest struct {
	Status        *[]SwapStatus
	FromTimestamp *int64
	ToTimestamp   *int64
	Offset        *uint32
	Limit         *uint32
}

func (r *ListSwapsRequest) Destroy() {
	FfiDestroyerOptionalSequenceSwapStatus{}.Destroy(r.Status)
	FfiDestroyerOptionalInt64{}.Destroy(r.FromTimestamp)
	FfiDestroyerOptionalInt64{}.Destroy(r.ToTimestamp)
	FfiDestroyerOptionalUint32{}.Destroy(r.Offset)
	FfiDestroyerOptionalUint32{}.Destroy(r.Limit)
}

type FfiConverterListSwapsRequest struct{}

var FfiConverterListSwapsRequestINSTANCE = FfiConverterListSwapsRequest{}

func (c FfiConverterListSwapsRequest) Lift(rb RustBufferI) ListSwapsRequest {
	return LiftFromRustBuffer[ListSwapsRequest](c, rb)
}

func (c FfiConverterListSwapsRequest) Read(reader io.Reader) ListSwapsRequest {
	return ListSwapsRequest{
		FfiConverterOptionalSequenceSwapStatusINSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterListSwapsRequest) Lower(value ListSwapsRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ListSwapsRequest](c, value)
}

func (c FfiConverterListSwapsRequest) Write(writer io.Writer, value ListSwapsRequest) {
	FfiConverterOptionalSequenceSwapStatusINSTANCE.Write(writer, value.Status)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.FromTimestamp)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.ToTimestamp)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Offset)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Limit)
}

type FfiDestroyerListSwapsRequest struct{}

func (_ FfiDestroyerListSwapsRequest) Destroy(value ListSwapsRequest) {
	value.Destroy()
}

type LnPaymentDetails struct {
	PaymentHash            string
	Label                  string
	DestinationPubkey      string
	PaymentPreimage        string
	Keysend                bool
	Bolt11                 string
	OpenChannelBolt11      *string
	LnurlSuccessAction     *SuccessActionProcessed
	LnurlPayDomain         *string
	LnurlPayComment        *string
	LnurlMetadata          *string
	LnAddress              *string
	LnurlWithdrawEndpoint  *string
	SwapInfo               *SwapInfo
	ReverseSwapInfo        *ReverseSwapInfo
	PendingExpirationBlock *uint32
}

func (r *LnPaymentDetails) Destroy() {
	FfiDestroyerString{}.Destroy(r.PaymentHash)
	FfiDestroyerString{}.Destroy(r.Label)
	FfiDestroyerString{}.Destroy(r.DestinationPubkey)
	FfiDestroyerString{}.Destroy(r.PaymentPreimage)
	FfiDestroyerBool{}.Destroy(r.Keysend)
	FfiDestroyerString{}.Destroy(r.Bolt11)
	FfiDestroyerOptionalString{}.Destroy(r.OpenChannelBolt11)
	FfiDestroyerOptionalSuccessActionProcessed{}.Destroy(r.LnurlSuccessAction)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlPayDomain)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlPayComment)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlMetadata)
	FfiDestroyerOptionalString{}.Destroy(r.LnAddress)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlWithdrawEndpoint)
	FfiDestroyerOptionalSwapInfo{}.Destroy(r.SwapInfo)
	FfiDestroyerOptionalReverseSwapInfo{}.Destroy(r.ReverseSwapInfo)
	FfiDestroyerOptionalUint32{}.Destroy(r.PendingExpirationBlock)
}

type FfiConverterLnPaymentDetails struct{}

var FfiConverterLnPaymentDetailsINSTANCE = FfiConverterLnPaymentDetails{}

func (c FfiConverterLnPaymentDetails) Lift(rb RustBufferI) LnPaymentDetails {
	return LiftFromRustBuffer[LnPaymentDetails](c, rb)
}

func (c FfiConverterLnPaymentDetails) Read(reader io.Reader) LnPaymentDetails {
	return LnPaymentDetails{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalSuccessActionProcessedINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalSwapInfoINSTANCE.Read(reader),
		FfiConverterOptionalReverseSwapInfoINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterLnPaymentDetails) Lower(value LnPaymentDetails) C.RustBuffer {
	return LowerIntoRustBuffer[LnPaymentDetails](c, value)
}

func (c FfiConverterLnPaymentDetails) Write(writer io.Writer, value LnPaymentDetails) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterStringINSTANCE.Write(writer, value.Label)
	FfiConverterStringINSTANCE.Write(writer, value.DestinationPubkey)
	FfiConverterStringINSTANCE.Write(writer, value.PaymentPreimage)
	FfiConverterBoolINSTANCE.Write(writer, value.Keysend)
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.OpenChannelBolt11)
	FfiConverterOptionalSuccessActionProcessedINSTANCE.Write(writer, value.LnurlSuccessAction)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlPayDomain)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlPayComment)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlMetadata)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnAddress)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlWithdrawEndpoint)
	FfiConverterOptionalSwapInfoINSTANCE.Write(writer, value.SwapInfo)
	FfiConverterOptionalReverseSwapInfoINSTANCE.Write(writer, value.ReverseSwapInfo)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.PendingExpirationBlock)
}

type FfiDestroyerLnPaymentDetails struct{}

func (_ FfiDestroyerLnPaymentDetails) Destroy(value LnPaymentDetails) {
	value.Destroy()
}

type LnUrlAuthRequestData struct {
	K1     string
	Domain string
	Url    string
	Action *string
}

func (r *LnUrlAuthRequestData) Destroy() {
	FfiDestroyerString{}.Destroy(r.K1)
	FfiDestroyerString{}.Destroy(r.Domain)
	FfiDestroyerString{}.Destroy(r.Url)
	FfiDestroyerOptionalString{}.Destroy(r.Action)
}

type FfiConverterLnUrlAuthRequestData struct{}

var FfiConverterLnUrlAuthRequestDataINSTANCE = FfiConverterLnUrlAuthRequestData{}

func (c FfiConverterLnUrlAuthRequestData) Lift(rb RustBufferI) LnUrlAuthRequestData {
	return LiftFromRustBuffer[LnUrlAuthRequestData](c, rb)
}

func (c FfiConverterLnUrlAuthRequestData) Read(reader io.Reader) LnUrlAuthRequestData {
	return LnUrlAuthRequestData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlAuthRequestData) Lower(value LnUrlAuthRequestData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlAuthRequestData](c, value)
}

func (c FfiConverterLnUrlAuthRequestData) Write(writer io.Writer, value LnUrlAuthRequestData) {
	FfiConverterStringINSTANCE.Write(writer, value.K1)
	FfiConverterStringINSTANCE.Write(writer, value.Domain)
	FfiConverterStringINSTANCE.Write(writer, value.Url)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Action)
}

type FfiDestroyerLnUrlAuthRequestData struct{}

func (_ FfiDestroyerLnUrlAuthRequestData) Destroy(value LnUrlAuthRequestData) {
	value.Destroy()
}

type LnUrlErrorData struct {
	Reason string
}

func (r *LnUrlErrorData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Reason)
}

type FfiConverterLnUrlErrorData struct{}

var FfiConverterLnUrlErrorDataINSTANCE = FfiConverterLnUrlErrorData{}

func (c FfiConverterLnUrlErrorData) Lift(rb RustBufferI) LnUrlErrorData {
	return LiftFromRustBuffer[LnUrlErrorData](c, rb)
}

func (c FfiConverterLnUrlErrorData) Read(reader io.Reader) LnUrlErrorData {
	return LnUrlErrorData{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlErrorData) Lower(value LnUrlErrorData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlErrorData](c, value)
}

func (c FfiConverterLnUrlErrorData) Write(writer io.Writer, value LnUrlErrorData) {
	FfiConverterStringINSTANCE.Write(writer, value.Reason)
}

type FfiDestroyerLnUrlErrorData struct{}

func (_ FfiDestroyerLnUrlErrorData) Destroy(value LnUrlErrorData) {
	value.Destroy()
}

type LnUrlPayErrorData struct {
	PaymentHash string
	Reason      string
}

func (r *LnUrlPayErrorData) Destroy() {
	FfiDestroyerString{}.Destroy(r.PaymentHash)
	FfiDestroyerString{}.Destroy(r.Reason)
}

type FfiConverterLnUrlPayErrorData struct{}

var FfiConverterLnUrlPayErrorDataINSTANCE = FfiConverterLnUrlPayErrorData{}

func (c FfiConverterLnUrlPayErrorData) Lift(rb RustBufferI) LnUrlPayErrorData {
	return LiftFromRustBuffer[LnUrlPayErrorData](c, rb)
}

func (c FfiConverterLnUrlPayErrorData) Read(reader io.Reader) LnUrlPayErrorData {
	return LnUrlPayErrorData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlPayErrorData) Lower(value LnUrlPayErrorData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayErrorData](c, value)
}

func (c FfiConverterLnUrlPayErrorData) Write(writer io.Writer, value LnUrlPayErrorData) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterStringINSTANCE.Write(writer, value.Reason)
}

type FfiDestroyerLnUrlPayErrorData struct{}

func (_ FfiDestroyerLnUrlPayErrorData) Destroy(value LnUrlPayErrorData) {
	value.Destroy()
}

type LnUrlPayRequest struct {
	Data                     LnUrlPayRequestData
	AmountMsat               uint64
	UseTrampoline            bool
	Comment                  *string
	PaymentLabel             *string
	ValidateSuccessActionUrl *bool
}

func (r *LnUrlPayRequest) Destroy() {
	FfiDestroyerLnUrlPayRequestData{}.Destroy(r.Data)
	FfiDestroyerUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerBool{}.Destroy(r.UseTrampoline)
	FfiDestroyerOptionalString{}.Destroy(r.Comment)
	FfiDestroyerOptionalString{}.Destroy(r.PaymentLabel)
	FfiDestroyerOptionalBool{}.Destroy(r.ValidateSuccessActionUrl)
}

type FfiConverterLnUrlPayRequest struct{}

var FfiConverterLnUrlPayRequestINSTANCE = FfiConverterLnUrlPayRequest{}

func (c FfiConverterLnUrlPayRequest) Lift(rb RustBufferI) LnUrlPayRequest {
	return LiftFromRustBuffer[LnUrlPayRequest](c, rb)
}

func (c FfiConverterLnUrlPayRequest) Read(reader io.Reader) LnUrlPayRequest {
	return LnUrlPayRequest{
		FfiConverterLnUrlPayRequestDataINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlPayRequest) Lower(value LnUrlPayRequest) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayRequest](c, value)
}

func (c FfiConverterLnUrlPayRequest) Write(writer io.Writer, value LnUrlPayRequest) {
	FfiConverterLnUrlPayRequestDataINSTANCE.Write(writer, value.Data)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterBoolINSTANCE.Write(writer, value.UseTrampoline)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Comment)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.PaymentLabel)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.ValidateSuccessActionUrl)
}

type FfiDestroyerLnUrlPayRequest struct{}

func (_ FfiDestroyerLnUrlPayRequest) Destroy(value LnUrlPayRequest) {
	value.Destroy()
}

type LnUrlPayRequestData struct {
	Callback       string
	MinSendable    uint64
	MaxSendable    uint64
	MetadataStr    string
	CommentAllowed uint16
	Domain         string
	AllowsNostr    bool
	NostrPubkey    *string
	LnAddress      *string
}

func (r *LnUrlPayRequestData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Callback)
	FfiDestroyerUint64{}.Destroy(r.MinSendable)
	FfiDestroyerUint64{}.Destroy(r.MaxSendable)
	FfiDestroyerString{}.Destroy(r.MetadataStr)
	FfiDestroyerUint16{}.Destroy(r.CommentAllowed)
	FfiDestroyerString{}.Destroy(r.Domain)
	FfiDestroyerBool{}.Destroy(r.AllowsNostr)
	FfiDestroyerOptionalString{}.Destroy(r.NostrPubkey)
	FfiDestroyerOptionalString{}.Destroy(r.LnAddress)
}

type FfiConverterLnUrlPayRequestData struct{}

var FfiConverterLnUrlPayRequestDataINSTANCE = FfiConverterLnUrlPayRequestData{}

func (c FfiConverterLnUrlPayRequestData) Lift(rb RustBufferI) LnUrlPayRequestData {
	return LiftFromRustBuffer[LnUrlPayRequestData](c, rb)
}

func (c FfiConverterLnUrlPayRequestData) Read(reader io.Reader) LnUrlPayRequestData {
	return LnUrlPayRequestData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint16INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlPayRequestData) Lower(value LnUrlPayRequestData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayRequestData](c, value)
}

func (c FfiConverterLnUrlPayRequestData) Write(writer io.Writer, value LnUrlPayRequestData) {
	FfiConverterStringINSTANCE.Write(writer, value.Callback)
	FfiConverterUint64INSTANCE.Write(writer, value.MinSendable)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxSendable)
	FfiConverterStringINSTANCE.Write(writer, value.MetadataStr)
	FfiConverterUint16INSTANCE.Write(writer, value.CommentAllowed)
	FfiConverterStringINSTANCE.Write(writer, value.Domain)
	FfiConverterBoolINSTANCE.Write(writer, value.AllowsNostr)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.NostrPubkey)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnAddress)
}

type FfiDestroyerLnUrlPayRequestData struct{}

func (_ FfiDestroyerLnUrlPayRequestData) Destroy(value LnUrlPayRequestData) {
	value.Destroy()
}

type LnUrlPaySuccessData struct {
	SuccessAction *SuccessActionProcessed
	Payment       Payment
}

func (r *LnUrlPaySuccessData) Destroy() {
	FfiDestroyerOptionalSuccessActionProcessed{}.Destroy(r.SuccessAction)
	FfiDestroyerPayment{}.Destroy(r.Payment)
}

type FfiConverterLnUrlPaySuccessData struct{}

var FfiConverterLnUrlPaySuccessDataINSTANCE = FfiConverterLnUrlPaySuccessData{}

func (c FfiConverterLnUrlPaySuccessData) Lift(rb RustBufferI) LnUrlPaySuccessData {
	return LiftFromRustBuffer[LnUrlPaySuccessData](c, rb)
}

func (c FfiConverterLnUrlPaySuccessData) Read(reader io.Reader) LnUrlPaySuccessData {
	return LnUrlPaySuccessData{
		FfiConverterOptionalSuccessActionProcessedINSTANCE.Read(reader),
		FfiConverterPaymentINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlPaySuccessData) Lower(value LnUrlPaySuccessData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlPaySuccessData](c, value)
}

func (c FfiConverterLnUrlPaySuccessData) Write(writer io.Writer, value LnUrlPaySuccessData) {
	FfiConverterOptionalSuccessActionProcessedINSTANCE.Write(writer, value.SuccessAction)
	FfiConverterPaymentINSTANCE.Write(writer, value.Payment)
}

type FfiDestroyerLnUrlPaySuccessData struct{}

func (_ FfiDestroyerLnUrlPaySuccessData) Destroy(value LnUrlPaySuccessData) {
	value.Destroy()
}

type LnUrlWithdrawRequest struct {
	Data        LnUrlWithdrawRequestData
	AmountMsat  uint64
	Description *string
}

func (r *LnUrlWithdrawRequest) Destroy() {
	FfiDestroyerLnUrlWithdrawRequestData{}.Destroy(r.Data)
	FfiDestroyerUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
}

type FfiConverterLnUrlWithdrawRequest struct{}

var FfiConverterLnUrlWithdrawRequestINSTANCE = FfiConverterLnUrlWithdrawRequest{}

func (c FfiConverterLnUrlWithdrawRequest) Lift(rb RustBufferI) LnUrlWithdrawRequest {
	return LiftFromRustBuffer[LnUrlWithdrawRequest](c, rb)
}

func (c FfiConverterLnUrlWithdrawRequest) Read(reader io.Reader) LnUrlWithdrawRequest {
	return LnUrlWithdrawRequest{
		FfiConverterLnUrlWithdrawRequestDataINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlWithdrawRequest) Lower(value LnUrlWithdrawRequest) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawRequest](c, value)
}

func (c FfiConverterLnUrlWithdrawRequest) Write(writer io.Writer, value LnUrlWithdrawRequest) {
	FfiConverterLnUrlWithdrawRequestDataINSTANCE.Write(writer, value.Data)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
}

type FfiDestroyerLnUrlWithdrawRequest struct{}

func (_ FfiDestroyerLnUrlWithdrawRequest) Destroy(value LnUrlWithdrawRequest) {
	value.Destroy()
}

type LnUrlWithdrawRequestData struct {
	Callback           string
	K1                 string
	DefaultDescription string
	MinWithdrawable    uint64
	MaxWithdrawable    uint64
}

func (r *LnUrlWithdrawRequestData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Callback)
	FfiDestroyerString{}.Destroy(r.K1)
	FfiDestroyerString{}.Destroy(r.DefaultDescription)
	FfiDestroyerUint64{}.Destroy(r.MinWithdrawable)
	FfiDestroyerUint64{}.Destroy(r.MaxWithdrawable)
}

type FfiConverterLnUrlWithdrawRequestData struct{}

var FfiConverterLnUrlWithdrawRequestDataINSTANCE = FfiConverterLnUrlWithdrawRequestData{}

func (c FfiConverterLnUrlWithdrawRequestData) Lift(rb RustBufferI) LnUrlWithdrawRequestData {
	return LiftFromRustBuffer[LnUrlWithdrawRequestData](c, rb)
}

func (c FfiConverterLnUrlWithdrawRequestData) Read(reader io.Reader) LnUrlWithdrawRequestData {
	return LnUrlWithdrawRequestData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlWithdrawRequestData) Lower(value LnUrlWithdrawRequestData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawRequestData](c, value)
}

func (c FfiConverterLnUrlWithdrawRequestData) Write(writer io.Writer, value LnUrlWithdrawRequestData) {
	FfiConverterStringINSTANCE.Write(writer, value.Callback)
	FfiConverterStringINSTANCE.Write(writer, value.K1)
	FfiConverterStringINSTANCE.Write(writer, value.DefaultDescription)
	FfiConverterUint64INSTANCE.Write(writer, value.MinWithdrawable)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxWithdrawable)
}

type FfiDestroyerLnUrlWithdrawRequestData struct{}

func (_ FfiDestroyerLnUrlWithdrawRequestData) Destroy(value LnUrlWithdrawRequestData) {
	value.Destroy()
}

type LnUrlWithdrawSuccessData struct {
	Invoice LnInvoice
}

func (r *LnUrlWithdrawSuccessData) Destroy() {
	FfiDestroyerLnInvoice{}.Destroy(r.Invoice)
}

type FfiConverterLnUrlWithdrawSuccessData struct{}

var FfiConverterLnUrlWithdrawSuccessDataINSTANCE = FfiConverterLnUrlWithdrawSuccessData{}

func (c FfiConverterLnUrlWithdrawSuccessData) Lift(rb RustBufferI) LnUrlWithdrawSuccessData {
	return LiftFromRustBuffer[LnUrlWithdrawSuccessData](c, rb)
}

func (c FfiConverterLnUrlWithdrawSuccessData) Read(reader io.Reader) LnUrlWithdrawSuccessData {
	return LnUrlWithdrawSuccessData{
		FfiConverterLnInvoiceINSTANCE.Read(reader),
	}
}

func (c FfiConverterLnUrlWithdrawSuccessData) Lower(value LnUrlWithdrawSuccessData) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawSuccessData](c, value)
}

func (c FfiConverterLnUrlWithdrawSuccessData) Write(writer io.Writer, value LnUrlWithdrawSuccessData) {
	FfiConverterLnInvoiceINSTANCE.Write(writer, value.Invoice)
}

type FfiDestroyerLnUrlWithdrawSuccessData struct{}

func (_ FfiDestroyerLnUrlWithdrawSuccessData) Destroy(value LnUrlWithdrawSuccessData) {
	value.Destroy()
}

type LocaleOverrides struct {
	Locale  string
	Spacing *uint32
	Symbol  Symbol
}

func (r *LocaleOverrides) Destroy() {
	FfiDestroyerString{}.Destroy(r.Locale)
	FfiDestroyerOptionalUint32{}.Destroy(r.Spacing)
	FfiDestroyerSymbol{}.Destroy(r.Symbol)
}

type FfiConverterLocaleOverrides struct{}

var FfiConverterLocaleOverridesINSTANCE = FfiConverterLocaleOverrides{}

func (c FfiConverterLocaleOverrides) Lift(rb RustBufferI) LocaleOverrides {
	return LiftFromRustBuffer[LocaleOverrides](c, rb)
}

func (c FfiConverterLocaleOverrides) Read(reader io.Reader) LocaleOverrides {
	return LocaleOverrides{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterSymbolINSTANCE.Read(reader),
	}
}

func (c FfiConverterLocaleOverrides) Lower(value LocaleOverrides) C.RustBuffer {
	return LowerIntoRustBuffer[LocaleOverrides](c, value)
}

func (c FfiConverterLocaleOverrides) Write(writer io.Writer, value LocaleOverrides) {
	FfiConverterStringINSTANCE.Write(writer, value.Locale)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Spacing)
	FfiConverterSymbolINSTANCE.Write(writer, value.Symbol)
}

type FfiDestroyerLocaleOverrides struct{}

func (_ FfiDestroyerLocaleOverrides) Destroy(value LocaleOverrides) {
	value.Destroy()
}

type LocalizedName struct {
	Locale string
	Name   string
}

func (r *LocalizedName) Destroy() {
	FfiDestroyerString{}.Destroy(r.Locale)
	FfiDestroyerString{}.Destroy(r.Name)
}

type FfiConverterLocalizedName struct{}

var FfiConverterLocalizedNameINSTANCE = FfiConverterLocalizedName{}

func (c FfiConverterLocalizedName) Lift(rb RustBufferI) LocalizedName {
	return LiftFromRustBuffer[LocalizedName](c, rb)
}

func (c FfiConverterLocalizedName) Read(reader io.Reader) LocalizedName {
	return LocalizedName{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLocalizedName) Lower(value LocalizedName) C.RustBuffer {
	return LowerIntoRustBuffer[LocalizedName](c, value)
}

func (c FfiConverterLocalizedName) Write(writer io.Writer, value LocalizedName) {
	FfiConverterStringINSTANCE.Write(writer, value.Locale)
	FfiConverterStringINSTANCE.Write(writer, value.Name)
}

type FfiDestroyerLocalizedName struct{}

func (_ FfiDestroyerLocalizedName) Destroy(value LocalizedName) {
	value.Destroy()
}

type LogEntry struct {
	Line  string
	Level string
}

func (r *LogEntry) Destroy() {
	FfiDestroyerString{}.Destroy(r.Line)
	FfiDestroyerString{}.Destroy(r.Level)
}

type FfiConverterLogEntry struct{}

var FfiConverterLogEntryINSTANCE = FfiConverterLogEntry{}

func (c FfiConverterLogEntry) Lift(rb RustBufferI) LogEntry {
	return LiftFromRustBuffer[LogEntry](c, rb)
}

func (c FfiConverterLogEntry) Read(reader io.Reader) LogEntry {
	return LogEntry{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterLogEntry) Lower(value LogEntry) C.RustBuffer {
	return LowerIntoRustBuffer[LogEntry](c, value)
}

func (c FfiConverterLogEntry) Write(writer io.Writer, value LogEntry) {
	FfiConverterStringINSTANCE.Write(writer, value.Line)
	FfiConverterStringINSTANCE.Write(writer, value.Level)
}

type FfiDestroyerLogEntry struct{}

func (_ FfiDestroyerLogEntry) Destroy(value LogEntry) {
	value.Destroy()
}

type LspInformation struct {
	Id                   string
	Name                 string
	WidgetUrl            string
	Pubkey               string
	Host                 string
	BaseFeeMsat          int64
	FeeRate              float64
	TimeLockDelta        uint32
	MinHtlcMsat          int64
	LspPubkey            []uint8
	OpeningFeeParamsList OpeningFeeParamsMenu
}

func (r *LspInformation) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerString{}.Destroy(r.Name)
	FfiDestroyerString{}.Destroy(r.WidgetUrl)
	FfiDestroyerString{}.Destroy(r.Pubkey)
	FfiDestroyerString{}.Destroy(r.Host)
	FfiDestroyerInt64{}.Destroy(r.BaseFeeMsat)
	FfiDestroyerFloat64{}.Destroy(r.FeeRate)
	FfiDestroyerUint32{}.Destroy(r.TimeLockDelta)
	FfiDestroyerInt64{}.Destroy(r.MinHtlcMsat)
	FfiDestroyerSequenceUint8{}.Destroy(r.LspPubkey)
	FfiDestroyerOpeningFeeParamsMenu{}.Destroy(r.OpeningFeeParamsList)
}

type FfiConverterLspInformation struct{}

var FfiConverterLspInformationINSTANCE = FfiConverterLspInformation{}

func (c FfiConverterLspInformation) Lift(rb RustBufferI) LspInformation {
	return LiftFromRustBuffer[LspInformation](c, rb)
}

func (c FfiConverterLspInformation) Read(reader io.Reader) LspInformation {
	return LspInformation{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterFloat64INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterOpeningFeeParamsMenuINSTANCE.Read(reader),
	}
}

func (c FfiConverterLspInformation) Lower(value LspInformation) C.RustBuffer {
	return LowerIntoRustBuffer[LspInformation](c, value)
}

func (c FfiConverterLspInformation) Write(writer io.Writer, value LspInformation) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterStringINSTANCE.Write(writer, value.Name)
	FfiConverterStringINSTANCE.Write(writer, value.WidgetUrl)
	FfiConverterStringINSTANCE.Write(writer, value.Pubkey)
	FfiConverterStringINSTANCE.Write(writer, value.Host)
	FfiConverterInt64INSTANCE.Write(writer, value.BaseFeeMsat)
	FfiConverterFloat64INSTANCE.Write(writer, value.FeeRate)
	FfiConverterUint32INSTANCE.Write(writer, value.TimeLockDelta)
	FfiConverterInt64INSTANCE.Write(writer, value.MinHtlcMsat)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.LspPubkey)
	FfiConverterOpeningFeeParamsMenuINSTANCE.Write(writer, value.OpeningFeeParamsList)
}

type FfiDestroyerLspInformation struct{}

func (_ FfiDestroyerLspInformation) Destroy(value LspInformation) {
	value.Destroy()
}

type MessageSuccessActionData struct {
	Message string
}

func (r *MessageSuccessActionData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Message)
}

type FfiConverterMessageSuccessActionData struct{}

var FfiConverterMessageSuccessActionDataINSTANCE = FfiConverterMessageSuccessActionData{}

func (c FfiConverterMessageSuccessActionData) Lift(rb RustBufferI) MessageSuccessActionData {
	return LiftFromRustBuffer[MessageSuccessActionData](c, rb)
}

func (c FfiConverterMessageSuccessActionData) Read(reader io.Reader) MessageSuccessActionData {
	return MessageSuccessActionData{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterMessageSuccessActionData) Lower(value MessageSuccessActionData) C.RustBuffer {
	return LowerIntoRustBuffer[MessageSuccessActionData](c, value)
}

func (c FfiConverterMessageSuccessActionData) Write(writer io.Writer, value MessageSuccessActionData) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerMessageSuccessActionData struct{}

func (_ FfiDestroyerMessageSuccessActionData) Destroy(value MessageSuccessActionData) {
	value.Destroy()
}

type MetadataFilter struct {
	JsonPath  string
	JsonValue string
}

func (r *MetadataFilter) Destroy() {
	FfiDestroyerString{}.Destroy(r.JsonPath)
	FfiDestroyerString{}.Destroy(r.JsonValue)
}

type FfiConverterMetadataFilter struct{}

var FfiConverterMetadataFilterINSTANCE = FfiConverterMetadataFilter{}

func (c FfiConverterMetadataFilter) Lift(rb RustBufferI) MetadataFilter {
	return LiftFromRustBuffer[MetadataFilter](c, rb)
}

func (c FfiConverterMetadataFilter) Read(reader io.Reader) MetadataFilter {
	return MetadataFilter{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterMetadataFilter) Lower(value MetadataFilter) C.RustBuffer {
	return LowerIntoRustBuffer[MetadataFilter](c, value)
}

func (c FfiConverterMetadataFilter) Write(writer io.Writer, value MetadataFilter) {
	FfiConverterStringINSTANCE.Write(writer, value.JsonPath)
	FfiConverterStringINSTANCE.Write(writer, value.JsonValue)
}

type FfiDestroyerMetadataFilter struct{}

func (_ FfiDestroyerMetadataFilter) Destroy(value MetadataFilter) {
	value.Destroy()
}

type MetadataItem struct {
	Key   string
	Value string
}

func (r *MetadataItem) Destroy() {
	FfiDestroyerString{}.Destroy(r.Key)
	FfiDestroyerString{}.Destroy(r.Value)
}

type FfiConverterMetadataItem struct{}

var FfiConverterMetadataItemINSTANCE = FfiConverterMetadataItem{}

func (c FfiConverterMetadataItem) Lift(rb RustBufferI) MetadataItem {
	return LiftFromRustBuffer[MetadataItem](c, rb)
}

func (c FfiConverterMetadataItem) Read(reader io.Reader) MetadataItem {
	return MetadataItem{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterMetadataItem) Lower(value MetadataItem) C.RustBuffer {
	return LowerIntoRustBuffer[MetadataItem](c, value)
}

func (c FfiConverterMetadataItem) Write(writer io.Writer, value MetadataItem) {
	FfiConverterStringINSTANCE.Write(writer, value.Key)
	FfiConverterStringINSTANCE.Write(writer, value.Value)
}

type FfiDestroyerMetadataItem struct{}

func (_ FfiDestroyerMetadataItem) Destroy(value MetadataItem) {
	value.Destroy()
}

type NodeState struct {
	Id                                   string
	BlockHeight                          uint32
	ChannelsBalanceMsat                  uint64
	OnchainBalanceMsat                   uint64
	PendingOnchainBalanceMsat            uint64
	Utxos                                []UnspentTransactionOutput
	MaxPayableMsat                       uint64
	MaxReceivableMsat                    uint64
	MaxSinglePaymentAmountMsat           uint64
	MaxChanReserveMsats                  uint64
	ConnectedPeers                       []string
	MaxReceivableSinglePaymentAmountMsat uint64
	TotalInboundLiquidityMsats           uint64
}

func (r *NodeState) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerUint32{}.Destroy(r.BlockHeight)
	FfiDestroyerUint64{}.Destroy(r.ChannelsBalanceMsat)
	FfiDestroyerUint64{}.Destroy(r.OnchainBalanceMsat)
	FfiDestroyerUint64{}.Destroy(r.PendingOnchainBalanceMsat)
	FfiDestroyerSequenceUnspentTransactionOutput{}.Destroy(r.Utxos)
	FfiDestroyerUint64{}.Destroy(r.MaxPayableMsat)
	FfiDestroyerUint64{}.Destroy(r.MaxReceivableMsat)
	FfiDestroyerUint64{}.Destroy(r.MaxSinglePaymentAmountMsat)
	FfiDestroyerUint64{}.Destroy(r.MaxChanReserveMsats)
	FfiDestroyerSequenceString{}.Destroy(r.ConnectedPeers)
	FfiDestroyerUint64{}.Destroy(r.MaxReceivableSinglePaymentAmountMsat)
	FfiDestroyerUint64{}.Destroy(r.TotalInboundLiquidityMsats)
}

type FfiConverterNodeState struct{}

var FfiConverterNodeStateINSTANCE = FfiConverterNodeState{}

func (c FfiConverterNodeState) Lift(rb RustBufferI) NodeState {
	return LiftFromRustBuffer[NodeState](c, rb)
}

func (c FfiConverterNodeState) Read(reader io.Reader) NodeState {
	return NodeState{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSequenceUnspentTransactionOutputINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSequenceStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterNodeState) Lower(value NodeState) C.RustBuffer {
	return LowerIntoRustBuffer[NodeState](c, value)
}

func (c FfiConverterNodeState) Write(writer io.Writer, value NodeState) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterUint32INSTANCE.Write(writer, value.BlockHeight)
	FfiConverterUint64INSTANCE.Write(writer, value.ChannelsBalanceMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.OnchainBalanceMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.PendingOnchainBalanceMsat)
	FfiConverterSequenceUnspentTransactionOutputINSTANCE.Write(writer, value.Utxos)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxPayableMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxReceivableMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxSinglePaymentAmountMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxChanReserveMsats)
	FfiConverterSequenceStringINSTANCE.Write(writer, value.ConnectedPeers)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxReceivableSinglePaymentAmountMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.TotalInboundLiquidityMsats)
}

type FfiDestroyerNodeState struct{}

func (_ FfiDestroyerNodeState) Destroy(value NodeState) {
	value.Destroy()
}

type OnchainPaymentLimitsResponse struct {
	MinSat        uint64
	MaxSat        uint64
	MaxPayableSat uint64
}

func (r *OnchainPaymentLimitsResponse) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.MinSat)
	FfiDestroyerUint64{}.Destroy(r.MaxSat)
	FfiDestroyerUint64{}.Destroy(r.MaxPayableSat)
}

type FfiConverterOnchainPaymentLimitsResponse struct{}

var FfiConverterOnchainPaymentLimitsResponseINSTANCE = FfiConverterOnchainPaymentLimitsResponse{}

func (c FfiConverterOnchainPaymentLimitsResponse) Lift(rb RustBufferI) OnchainPaymentLimitsResponse {
	return LiftFromRustBuffer[OnchainPaymentLimitsResponse](c, rb)
}

func (c FfiConverterOnchainPaymentLimitsResponse) Read(reader io.Reader) OnchainPaymentLimitsResponse {
	return OnchainPaymentLimitsResponse{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterOnchainPaymentLimitsResponse) Lower(value OnchainPaymentLimitsResponse) C.RustBuffer {
	return LowerIntoRustBuffer[OnchainPaymentLimitsResponse](c, value)
}

func (c FfiConverterOnchainPaymentLimitsResponse) Write(writer io.Writer, value OnchainPaymentLimitsResponse) {
	FfiConverterUint64INSTANCE.Write(writer, value.MinSat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxSat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxPayableSat)
}

type FfiDestroyerOnchainPaymentLimitsResponse struct{}

func (_ FfiDestroyerOnchainPaymentLimitsResponse) Destroy(value OnchainPaymentLimitsResponse) {
	value.Destroy()
}

type OpenChannelFeeRequest struct {
	AmountMsat *uint64
	Expiry     *uint32
}

func (r *OpenChannelFeeRequest) Destroy() {
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerOptionalUint32{}.Destroy(r.Expiry)
}

type FfiConverterOpenChannelFeeRequest struct{}

var FfiConverterOpenChannelFeeRequestINSTANCE = FfiConverterOpenChannelFeeRequest{}

func (c FfiConverterOpenChannelFeeRequest) Lift(rb RustBufferI) OpenChannelFeeRequest {
	return LiftFromRustBuffer[OpenChannelFeeRequest](c, rb)
}

func (c FfiConverterOpenChannelFeeRequest) Read(reader io.Reader) OpenChannelFeeRequest {
	return OpenChannelFeeRequest{
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterOpenChannelFeeRequest) Lower(value OpenChannelFeeRequest) C.RustBuffer {
	return LowerIntoRustBuffer[OpenChannelFeeRequest](c, value)
}

func (c FfiConverterOpenChannelFeeRequest) Write(writer io.Writer, value OpenChannelFeeRequest) {
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Expiry)
}

type FfiDestroyerOpenChannelFeeRequest struct{}

func (_ FfiDestroyerOpenChannelFeeRequest) Destroy(value OpenChannelFeeRequest) {
	value.Destroy()
}

type OpenChannelFeeResponse struct {
	FeeMsat   *uint64
	FeeParams OpeningFeeParams
}

func (r *OpenChannelFeeResponse) Destroy() {
	FfiDestroyerOptionalUint64{}.Destroy(r.FeeMsat)
	FfiDestroyerOpeningFeeParams{}.Destroy(r.FeeParams)
}

type FfiConverterOpenChannelFeeResponse struct{}

var FfiConverterOpenChannelFeeResponseINSTANCE = FfiConverterOpenChannelFeeResponse{}

func (c FfiConverterOpenChannelFeeResponse) Lift(rb RustBufferI) OpenChannelFeeResponse {
	return LiftFromRustBuffer[OpenChannelFeeResponse](c, rb)
}

func (c FfiConverterOpenChannelFeeResponse) Read(reader io.Reader) OpenChannelFeeResponse {
	return OpenChannelFeeResponse{
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOpeningFeeParamsINSTANCE.Read(reader),
	}
}

func (c FfiConverterOpenChannelFeeResponse) Lower(value OpenChannelFeeResponse) C.RustBuffer {
	return LowerIntoRustBuffer[OpenChannelFeeResponse](c, value)
}

func (c FfiConverterOpenChannelFeeResponse) Write(writer io.Writer, value OpenChannelFeeResponse) {
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.FeeMsat)
	FfiConverterOpeningFeeParamsINSTANCE.Write(writer, value.FeeParams)
}

type FfiDestroyerOpenChannelFeeResponse struct{}

func (_ FfiDestroyerOpenChannelFeeResponse) Destroy(value OpenChannelFeeResponse) {
	value.Destroy()
}

type OpeningFeeParams struct {
	MinMsat              uint64
	Proportional         uint32
	ValidUntil           string
	MaxIdleTime          uint32
	MaxClientToSelfDelay uint32
	Promise              string
}

func (r *OpeningFeeParams) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.MinMsat)
	FfiDestroyerUint32{}.Destroy(r.Proportional)
	FfiDestroyerString{}.Destroy(r.ValidUntil)
	FfiDestroyerUint32{}.Destroy(r.MaxIdleTime)
	FfiDestroyerUint32{}.Destroy(r.MaxClientToSelfDelay)
	FfiDestroyerString{}.Destroy(r.Promise)
}

type FfiConverterOpeningFeeParams struct{}

var FfiConverterOpeningFeeParamsINSTANCE = FfiConverterOpeningFeeParams{}

func (c FfiConverterOpeningFeeParams) Lift(rb RustBufferI) OpeningFeeParams {
	return LiftFromRustBuffer[OpeningFeeParams](c, rb)
}

func (c FfiConverterOpeningFeeParams) Read(reader io.Reader) OpeningFeeParams {
	return OpeningFeeParams{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterOpeningFeeParams) Lower(value OpeningFeeParams) C.RustBuffer {
	return LowerIntoRustBuffer[OpeningFeeParams](c, value)
}

func (c FfiConverterOpeningFeeParams) Write(writer io.Writer, value OpeningFeeParams) {
	FfiConverterUint64INSTANCE.Write(writer, value.MinMsat)
	FfiConverterUint32INSTANCE.Write(writer, value.Proportional)
	FfiConverterStringINSTANCE.Write(writer, value.ValidUntil)
	FfiConverterUint32INSTANCE.Write(writer, value.MaxIdleTime)
	FfiConverterUint32INSTANCE.Write(writer, value.MaxClientToSelfDelay)
	FfiConverterStringINSTANCE.Write(writer, value.Promise)
}

type FfiDestroyerOpeningFeeParams struct{}

func (_ FfiDestroyerOpeningFeeParams) Destroy(value OpeningFeeParams) {
	value.Destroy()
}

type OpeningFeeParamsMenu struct {
	Values []OpeningFeeParams
}

func (r *OpeningFeeParamsMenu) Destroy() {
	FfiDestroyerSequenceOpeningFeeParams{}.Destroy(r.Values)
}

type FfiConverterOpeningFeeParamsMenu struct{}

var FfiConverterOpeningFeeParamsMenuINSTANCE = FfiConverterOpeningFeeParamsMenu{}

func (c FfiConverterOpeningFeeParamsMenu) Lift(rb RustBufferI) OpeningFeeParamsMenu {
	return LiftFromRustBuffer[OpeningFeeParamsMenu](c, rb)
}

func (c FfiConverterOpeningFeeParamsMenu) Read(reader io.Reader) OpeningFeeParamsMenu {
	return OpeningFeeParamsMenu{
		FfiConverterSequenceOpeningFeeParamsINSTANCE.Read(reader),
	}
}

func (c FfiConverterOpeningFeeParamsMenu) Lower(value OpeningFeeParamsMenu) C.RustBuffer {
	return LowerIntoRustBuffer[OpeningFeeParamsMenu](c, value)
}

func (c FfiConverterOpeningFeeParamsMenu) Write(writer io.Writer, value OpeningFeeParamsMenu) {
	FfiConverterSequenceOpeningFeeParamsINSTANCE.Write(writer, value.Values)
}

type FfiDestroyerOpeningFeeParamsMenu struct{}

func (_ FfiDestroyerOpeningFeeParamsMenu) Destroy(value OpeningFeeParamsMenu) {
	value.Destroy()
}

type PayOnchainRequest struct {
	RecipientAddress string
	PrepareRes       PrepareOnchainPaymentResponse
}

func (r *PayOnchainRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.RecipientAddress)
	FfiDestroyerPrepareOnchainPaymentResponse{}.Destroy(r.PrepareRes)
}

type FfiConverterPayOnchainRequest struct{}

var FfiConverterPayOnchainRequestINSTANCE = FfiConverterPayOnchainRequest{}

func (c FfiConverterPayOnchainRequest) Lift(rb RustBufferI) PayOnchainRequest {
	return LiftFromRustBuffer[PayOnchainRequest](c, rb)
}

func (c FfiConverterPayOnchainRequest) Read(reader io.Reader) PayOnchainRequest {
	return PayOnchainRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterPrepareOnchainPaymentResponseINSTANCE.Read(reader),
	}
}

func (c FfiConverterPayOnchainRequest) Lower(value PayOnchainRequest) C.RustBuffer {
	return LowerIntoRustBuffer[PayOnchainRequest](c, value)
}

func (c FfiConverterPayOnchainRequest) Write(writer io.Writer, value PayOnchainRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.RecipientAddress)
	FfiConverterPrepareOnchainPaymentResponseINSTANCE.Write(writer, value.PrepareRes)
}

type FfiDestroyerPayOnchainRequest struct{}

func (_ FfiDestroyerPayOnchainRequest) Destroy(value PayOnchainRequest) {
	value.Destroy()
}

type PayOnchainResponse struct {
	ReverseSwapInfo ReverseSwapInfo
}

func (r *PayOnchainResponse) Destroy() {
	FfiDestroyerReverseSwapInfo{}.Destroy(r.ReverseSwapInfo)
}

type FfiConverterPayOnchainResponse struct{}

var FfiConverterPayOnchainResponseINSTANCE = FfiConverterPayOnchainResponse{}

func (c FfiConverterPayOnchainResponse) Lift(rb RustBufferI) PayOnchainResponse {
	return LiftFromRustBuffer[PayOnchainResponse](c, rb)
}

func (c FfiConverterPayOnchainResponse) Read(reader io.Reader) PayOnchainResponse {
	return PayOnchainResponse{
		FfiConverterReverseSwapInfoINSTANCE.Read(reader),
	}
}

func (c FfiConverterPayOnchainResponse) Lower(value PayOnchainResponse) C.RustBuffer {
	return LowerIntoRustBuffer[PayOnchainResponse](c, value)
}

func (c FfiConverterPayOnchainResponse) Write(writer io.Writer, value PayOnchainResponse) {
	FfiConverterReverseSwapInfoINSTANCE.Write(writer, value.ReverseSwapInfo)
}

type FfiDestroyerPayOnchainResponse struct{}

func (_ FfiDestroyerPayOnchainResponse) Destroy(value PayOnchainResponse) {
	value.Destroy()
}

type Payment struct {
	Id          string
	PaymentType PaymentType
	PaymentTime int64
	AmountMsat  uint64
	FeeMsat     uint64
	Status      PaymentStatus
	Error       *string
	Description *string
	Details     PaymentDetails
	Metadata    *string
}

func (r *Payment) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerPaymentType{}.Destroy(r.PaymentType)
	FfiDestroyerInt64{}.Destroy(r.PaymentTime)
	FfiDestroyerUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerUint64{}.Destroy(r.FeeMsat)
	FfiDestroyerPaymentStatus{}.Destroy(r.Status)
	FfiDestroyerOptionalString{}.Destroy(r.Error)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerPaymentDetails{}.Destroy(r.Details)
	FfiDestroyerOptionalString{}.Destroy(r.Metadata)
}

type FfiConverterPayment struct{}

var FfiConverterPaymentINSTANCE = FfiConverterPayment{}

func (c FfiConverterPayment) Lift(rb RustBufferI) Payment {
	return LiftFromRustBuffer[Payment](c, rb)
}

func (c FfiConverterPayment) Read(reader io.Reader) Payment {
	return Payment{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterPaymentTypeINSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterPaymentStatusINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterPaymentDetailsINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterPayment) Lower(value Payment) C.RustBuffer {
	return LowerIntoRustBuffer[Payment](c, value)
}

func (c FfiConverterPayment) Write(writer io.Writer, value Payment) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterPaymentTypeINSTANCE.Write(writer, value.PaymentType)
	FfiConverterInt64INSTANCE.Write(writer, value.PaymentTime)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.FeeMsat)
	FfiConverterPaymentStatusINSTANCE.Write(writer, value.Status)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Error)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterPaymentDetailsINSTANCE.Write(writer, value.Details)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Metadata)
}

type FfiDestroyerPayment struct{}

func (_ FfiDestroyerPayment) Destroy(value Payment) {
	value.Destroy()
}

type PaymentFailedData struct {
	Error   string
	NodeId  string
	Invoice *LnInvoice
	Label   *string
}

func (r *PaymentFailedData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Error)
	FfiDestroyerString{}.Destroy(r.NodeId)
	FfiDestroyerOptionalLnInvoice{}.Destroy(r.Invoice)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
}

type FfiConverterPaymentFailedData struct{}

var FfiConverterPaymentFailedDataINSTANCE = FfiConverterPaymentFailedData{}

func (c FfiConverterPaymentFailedData) Lift(rb RustBufferI) PaymentFailedData {
	return LiftFromRustBuffer[PaymentFailedData](c, rb)
}

func (c FfiConverterPaymentFailedData) Read(reader io.Reader) PaymentFailedData {
	return PaymentFailedData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalLnInvoiceINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterPaymentFailedData) Lower(value PaymentFailedData) C.RustBuffer {
	return LowerIntoRustBuffer[PaymentFailedData](c, value)
}

func (c FfiConverterPaymentFailedData) Write(writer io.Writer, value PaymentFailedData) {
	FfiConverterStringINSTANCE.Write(writer, value.Error)
	FfiConverterStringINSTANCE.Write(writer, value.NodeId)
	FfiConverterOptionalLnInvoiceINSTANCE.Write(writer, value.Invoice)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
}

type FfiDestroyerPaymentFailedData struct{}

func (_ FfiDestroyerPaymentFailedData) Destroy(value PaymentFailedData) {
	value.Destroy()
}

type PrepareOnchainPaymentRequest struct {
	AmountSat      uint64
	AmountType     SwapAmountType
	ClaimTxFeerate uint32
}

func (r *PrepareOnchainPaymentRequest) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.AmountSat)
	FfiDestroyerSwapAmountType{}.Destroy(r.AmountType)
	FfiDestroyerUint32{}.Destroy(r.ClaimTxFeerate)
}

type FfiConverterPrepareOnchainPaymentRequest struct{}

var FfiConverterPrepareOnchainPaymentRequestINSTANCE = FfiConverterPrepareOnchainPaymentRequest{}

func (c FfiConverterPrepareOnchainPaymentRequest) Lift(rb RustBufferI) PrepareOnchainPaymentRequest {
	return LiftFromRustBuffer[PrepareOnchainPaymentRequest](c, rb)
}

func (c FfiConverterPrepareOnchainPaymentRequest) Read(reader io.Reader) PrepareOnchainPaymentRequest {
	return PrepareOnchainPaymentRequest{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSwapAmountTypeINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareOnchainPaymentRequest) Lower(value PrepareOnchainPaymentRequest) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareOnchainPaymentRequest](c, value)
}

func (c FfiConverterPrepareOnchainPaymentRequest) Write(writer io.Writer, value PrepareOnchainPaymentRequest) {
	FfiConverterUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterSwapAmountTypeINSTANCE.Write(writer, value.AmountType)
	FfiConverterUint32INSTANCE.Write(writer, value.ClaimTxFeerate)
}

type FfiDestroyerPrepareOnchainPaymentRequest struct{}

func (_ FfiDestroyerPrepareOnchainPaymentRequest) Destroy(value PrepareOnchainPaymentRequest) {
	value.Destroy()
}

type PrepareOnchainPaymentResponse struct {
	FeesHash           string
	FeesPercentage     float64
	FeesLockup         uint64
	FeesClaim          uint64
	SenderAmountSat    uint64
	RecipientAmountSat uint64
	TotalFees          uint64
}

func (r *PrepareOnchainPaymentResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.FeesHash)
	FfiDestroyerFloat64{}.Destroy(r.FeesPercentage)
	FfiDestroyerUint64{}.Destroy(r.FeesLockup)
	FfiDestroyerUint64{}.Destroy(r.FeesClaim)
	FfiDestroyerUint64{}.Destroy(r.SenderAmountSat)
	FfiDestroyerUint64{}.Destroy(r.RecipientAmountSat)
	FfiDestroyerUint64{}.Destroy(r.TotalFees)
}

type FfiConverterPrepareOnchainPaymentResponse struct{}

var FfiConverterPrepareOnchainPaymentResponseINSTANCE = FfiConverterPrepareOnchainPaymentResponse{}

func (c FfiConverterPrepareOnchainPaymentResponse) Lift(rb RustBufferI) PrepareOnchainPaymentResponse {
	return LiftFromRustBuffer[PrepareOnchainPaymentResponse](c, rb)
}

func (c FfiConverterPrepareOnchainPaymentResponse) Read(reader io.Reader) PrepareOnchainPaymentResponse {
	return PrepareOnchainPaymentResponse{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFloat64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareOnchainPaymentResponse) Lower(value PrepareOnchainPaymentResponse) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareOnchainPaymentResponse](c, value)
}

func (c FfiConverterPrepareOnchainPaymentResponse) Write(writer io.Writer, value PrepareOnchainPaymentResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.FeesHash)
	FfiConverterFloat64INSTANCE.Write(writer, value.FeesPercentage)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesLockup)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesClaim)
	FfiConverterUint64INSTANCE.Write(writer, value.SenderAmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.RecipientAmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.TotalFees)
}

type FfiDestroyerPrepareOnchainPaymentResponse struct{}

func (_ FfiDestroyerPrepareOnchainPaymentResponse) Destroy(value PrepareOnchainPaymentResponse) {
	value.Destroy()
}

type PrepareRedeemOnchainFundsRequest struct {
	ToAddress   string
	SatPerVbyte uint32
}

func (r *PrepareRedeemOnchainFundsRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.ToAddress)
	FfiDestroyerUint32{}.Destroy(r.SatPerVbyte)
}

type FfiConverterPrepareRedeemOnchainFundsRequest struct{}

var FfiConverterPrepareRedeemOnchainFundsRequestINSTANCE = FfiConverterPrepareRedeemOnchainFundsRequest{}

func (c FfiConverterPrepareRedeemOnchainFundsRequest) Lift(rb RustBufferI) PrepareRedeemOnchainFundsRequest {
	return LiftFromRustBuffer[PrepareRedeemOnchainFundsRequest](c, rb)
}

func (c FfiConverterPrepareRedeemOnchainFundsRequest) Read(reader io.Reader) PrepareRedeemOnchainFundsRequest {
	return PrepareRedeemOnchainFundsRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareRedeemOnchainFundsRequest) Lower(value PrepareRedeemOnchainFundsRequest) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareRedeemOnchainFundsRequest](c, value)
}

func (c FfiConverterPrepareRedeemOnchainFundsRequest) Write(writer io.Writer, value PrepareRedeemOnchainFundsRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.ToAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.SatPerVbyte)
}

type FfiDestroyerPrepareRedeemOnchainFundsRequest struct{}

func (_ FfiDestroyerPrepareRedeemOnchainFundsRequest) Destroy(value PrepareRedeemOnchainFundsRequest) {
	value.Destroy()
}

type PrepareRedeemOnchainFundsResponse struct {
	TxWeight uint64
	TxFeeSat uint64
}

func (r *PrepareRedeemOnchainFundsResponse) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.TxWeight)
	FfiDestroyerUint64{}.Destroy(r.TxFeeSat)
}

type FfiConverterPrepareRedeemOnchainFundsResponse struct{}

var FfiConverterPrepareRedeemOnchainFundsResponseINSTANCE = FfiConverterPrepareRedeemOnchainFundsResponse{}

func (c FfiConverterPrepareRedeemOnchainFundsResponse) Lift(rb RustBufferI) PrepareRedeemOnchainFundsResponse {
	return LiftFromRustBuffer[PrepareRedeemOnchainFundsResponse](c, rb)
}

func (c FfiConverterPrepareRedeemOnchainFundsResponse) Read(reader io.Reader) PrepareRedeemOnchainFundsResponse {
	return PrepareRedeemOnchainFundsResponse{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareRedeemOnchainFundsResponse) Lower(value PrepareRedeemOnchainFundsResponse) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareRedeemOnchainFundsResponse](c, value)
}

func (c FfiConverterPrepareRedeemOnchainFundsResponse) Write(writer io.Writer, value PrepareRedeemOnchainFundsResponse) {
	FfiConverterUint64INSTANCE.Write(writer, value.TxWeight)
	FfiConverterUint64INSTANCE.Write(writer, value.TxFeeSat)
}

type FfiDestroyerPrepareRedeemOnchainFundsResponse struct{}

func (_ FfiDestroyerPrepareRedeemOnchainFundsResponse) Destroy(value PrepareRedeemOnchainFundsResponse) {
	value.Destroy()
}

type PrepareRefundRequest struct {
	SwapAddress string
	ToAddress   string
	SatPerVbyte uint32
	Unilateral  *bool
}

func (r *PrepareRefundRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.SwapAddress)
	FfiDestroyerString{}.Destroy(r.ToAddress)
	FfiDestroyerUint32{}.Destroy(r.SatPerVbyte)
	FfiDestroyerOptionalBool{}.Destroy(r.Unilateral)
}

type FfiConverterPrepareRefundRequest struct{}

var FfiConverterPrepareRefundRequestINSTANCE = FfiConverterPrepareRefundRequest{}

func (c FfiConverterPrepareRefundRequest) Lift(rb RustBufferI) PrepareRefundRequest {
	return LiftFromRustBuffer[PrepareRefundRequest](c, rb)
}

func (c FfiConverterPrepareRefundRequest) Read(reader io.Reader) PrepareRefundRequest {
	return PrepareRefundRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareRefundRequest) Lower(value PrepareRefundRequest) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareRefundRequest](c, value)
}

func (c FfiConverterPrepareRefundRequest) Write(writer io.Writer, value PrepareRefundRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapAddress)
	FfiConverterStringINSTANCE.Write(writer, value.ToAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.SatPerVbyte)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.Unilateral)
}

type FfiDestroyerPrepareRefundRequest struct{}

func (_ FfiDestroyerPrepareRefundRequest) Destroy(value PrepareRefundRequest) {
	value.Destroy()
}

type PrepareRefundResponse struct {
	RefundTxWeight uint32
	RefundTxFeeSat uint64
}

func (r *PrepareRefundResponse) Destroy() {
	FfiDestroyerUint32{}.Destroy(r.RefundTxWeight)
	FfiDestroyerUint64{}.Destroy(r.RefundTxFeeSat)
}

type FfiConverterPrepareRefundResponse struct{}

var FfiConverterPrepareRefundResponseINSTANCE = FfiConverterPrepareRefundResponse{}

func (c FfiConverterPrepareRefundResponse) Lift(rb RustBufferI) PrepareRefundResponse {
	return LiftFromRustBuffer[PrepareRefundResponse](c, rb)
}

func (c FfiConverterPrepareRefundResponse) Read(reader io.Reader) PrepareRefundResponse {
	return PrepareRefundResponse{
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterPrepareRefundResponse) Lower(value PrepareRefundResponse) C.RustBuffer {
	return LowerIntoRustBuffer[PrepareRefundResponse](c, value)
}

func (c FfiConverterPrepareRefundResponse) Write(writer io.Writer, value PrepareRefundResponse) {
	FfiConverterUint32INSTANCE.Write(writer, value.RefundTxWeight)
	FfiConverterUint64INSTANCE.Write(writer, value.RefundTxFeeSat)
}

type FfiDestroyerPrepareRefundResponse struct{}

func (_ FfiDestroyerPrepareRefundResponse) Destroy(value PrepareRefundResponse) {
	value.Destroy()
}

type Rate struct {
	Coin  string
	Value float64
}

func (r *Rate) Destroy() {
	FfiDestroyerString{}.Destroy(r.Coin)
	FfiDestroyerFloat64{}.Destroy(r.Value)
}

type FfiConverterRate struct{}

var FfiConverterRateINSTANCE = FfiConverterRate{}

func (c FfiConverterRate) Lift(rb RustBufferI) Rate {
	return LiftFromRustBuffer[Rate](c, rb)
}

func (c FfiConverterRate) Read(reader io.Reader) Rate {
	return Rate{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFloat64INSTANCE.Read(reader),
	}
}

func (c FfiConverterRate) Lower(value Rate) C.RustBuffer {
	return LowerIntoRustBuffer[Rate](c, value)
}

func (c FfiConverterRate) Write(writer io.Writer, value Rate) {
	FfiConverterStringINSTANCE.Write(writer, value.Coin)
	FfiConverterFloat64INSTANCE.Write(writer, value.Value)
}

type FfiDestroyerRate struct{}

func (_ FfiDestroyerRate) Destroy(value Rate) {
	value.Destroy()
}

type ReceiveOnchainRequest struct {
	OpeningFeeParams *OpeningFeeParams
}

func (r *ReceiveOnchainRequest) Destroy() {
	FfiDestroyerOptionalOpeningFeeParams{}.Destroy(r.OpeningFeeParams)
}

type FfiConverterReceiveOnchainRequest struct{}

var FfiConverterReceiveOnchainRequestINSTANCE = FfiConverterReceiveOnchainRequest{}

func (c FfiConverterReceiveOnchainRequest) Lift(rb RustBufferI) ReceiveOnchainRequest {
	return LiftFromRustBuffer[ReceiveOnchainRequest](c, rb)
}

func (c FfiConverterReceiveOnchainRequest) Read(reader io.Reader) ReceiveOnchainRequest {
	return ReceiveOnchainRequest{
		FfiConverterOptionalOpeningFeeParamsINSTANCE.Read(reader),
	}
}

func (c FfiConverterReceiveOnchainRequest) Lower(value ReceiveOnchainRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ReceiveOnchainRequest](c, value)
}

func (c FfiConverterReceiveOnchainRequest) Write(writer io.Writer, value ReceiveOnchainRequest) {
	FfiConverterOptionalOpeningFeeParamsINSTANCE.Write(writer, value.OpeningFeeParams)
}

type FfiDestroyerReceiveOnchainRequest struct{}

func (_ FfiDestroyerReceiveOnchainRequest) Destroy(value ReceiveOnchainRequest) {
	value.Destroy()
}

type ReceivePaymentRequest struct {
	AmountMsat         uint64
	Description        string
	Preimage           *[]uint8
	OpeningFeeParams   *OpeningFeeParams
	UseDescriptionHash *bool
	Expiry             *uint32
	Cltv               *uint32
}

func (r *ReceivePaymentRequest) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerString{}.Destroy(r.Description)
	FfiDestroyerOptionalSequenceUint8{}.Destroy(r.Preimage)
	FfiDestroyerOptionalOpeningFeeParams{}.Destroy(r.OpeningFeeParams)
	FfiDestroyerOptionalBool{}.Destroy(r.UseDescriptionHash)
	FfiDestroyerOptionalUint32{}.Destroy(r.Expiry)
	FfiDestroyerOptionalUint32{}.Destroy(r.Cltv)
}

type FfiConverterReceivePaymentRequest struct{}

var FfiConverterReceivePaymentRequestINSTANCE = FfiConverterReceivePaymentRequest{}

func (c FfiConverterReceivePaymentRequest) Lift(rb RustBufferI) ReceivePaymentRequest {
	return LiftFromRustBuffer[ReceivePaymentRequest](c, rb)
}

func (c FfiConverterReceivePaymentRequest) Read(reader io.Reader) ReceivePaymentRequest {
	return ReceivePaymentRequest{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalSequenceUint8INSTANCE.Read(reader),
		FfiConverterOptionalOpeningFeeParamsINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterReceivePaymentRequest) Lower(value ReceivePaymentRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ReceivePaymentRequest](c, value)
}

func (c FfiConverterReceivePaymentRequest) Write(writer io.Writer, value ReceivePaymentRequest) {
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterOptionalSequenceUint8INSTANCE.Write(writer, value.Preimage)
	FfiConverterOptionalOpeningFeeParamsINSTANCE.Write(writer, value.OpeningFeeParams)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.UseDescriptionHash)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Expiry)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Cltv)
}

type FfiDestroyerReceivePaymentRequest struct{}

func (_ FfiDestroyerReceivePaymentRequest) Destroy(value ReceivePaymentRequest) {
	value.Destroy()
}

type ReceivePaymentResponse struct {
	LnInvoice        LnInvoice
	OpeningFeeParams *OpeningFeeParams
	OpeningFeeMsat   *uint64
}

func (r *ReceivePaymentResponse) Destroy() {
	FfiDestroyerLnInvoice{}.Destroy(r.LnInvoice)
	FfiDestroyerOptionalOpeningFeeParams{}.Destroy(r.OpeningFeeParams)
	FfiDestroyerOptionalUint64{}.Destroy(r.OpeningFeeMsat)
}

type FfiConverterReceivePaymentResponse struct{}

var FfiConverterReceivePaymentResponseINSTANCE = FfiConverterReceivePaymentResponse{}

func (c FfiConverterReceivePaymentResponse) Lift(rb RustBufferI) ReceivePaymentResponse {
	return LiftFromRustBuffer[ReceivePaymentResponse](c, rb)
}

func (c FfiConverterReceivePaymentResponse) Read(reader io.Reader) ReceivePaymentResponse {
	return ReceivePaymentResponse{
		FfiConverterLnInvoiceINSTANCE.Read(reader),
		FfiConverterOptionalOpeningFeeParamsINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterReceivePaymentResponse) Lower(value ReceivePaymentResponse) C.RustBuffer {
	return LowerIntoRustBuffer[ReceivePaymentResponse](c, value)
}

func (c FfiConverterReceivePaymentResponse) Write(writer io.Writer, value ReceivePaymentResponse) {
	FfiConverterLnInvoiceINSTANCE.Write(writer, value.LnInvoice)
	FfiConverterOptionalOpeningFeeParamsINSTANCE.Write(writer, value.OpeningFeeParams)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.OpeningFeeMsat)
}

type FfiDestroyerReceivePaymentResponse struct{}

func (_ FfiDestroyerReceivePaymentResponse) Destroy(value ReceivePaymentResponse) {
	value.Destroy()
}

type RecommendedFees struct {
	FastestFee  uint64
	HalfHourFee uint64
	HourFee     uint64
	EconomyFee  uint64
	MinimumFee  uint64
}

func (r *RecommendedFees) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.FastestFee)
	FfiDestroyerUint64{}.Destroy(r.HalfHourFee)
	FfiDestroyerUint64{}.Destroy(r.HourFee)
	FfiDestroyerUint64{}.Destroy(r.EconomyFee)
	FfiDestroyerUint64{}.Destroy(r.MinimumFee)
}

type FfiConverterRecommendedFees struct{}

var FfiConverterRecommendedFeesINSTANCE = FfiConverterRecommendedFees{}

func (c FfiConverterRecommendedFees) Lift(rb RustBufferI) RecommendedFees {
	return LiftFromRustBuffer[RecommendedFees](c, rb)
}

func (c FfiConverterRecommendedFees) Read(reader io.Reader) RecommendedFees {
	return RecommendedFees{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterRecommendedFees) Lower(value RecommendedFees) C.RustBuffer {
	return LowerIntoRustBuffer[RecommendedFees](c, value)
}

func (c FfiConverterRecommendedFees) Write(writer io.Writer, value RecommendedFees) {
	FfiConverterUint64INSTANCE.Write(writer, value.FastestFee)
	FfiConverterUint64INSTANCE.Write(writer, value.HalfHourFee)
	FfiConverterUint64INSTANCE.Write(writer, value.HourFee)
	FfiConverterUint64INSTANCE.Write(writer, value.EconomyFee)
	FfiConverterUint64INSTANCE.Write(writer, value.MinimumFee)
}

type FfiDestroyerRecommendedFees struct{}

func (_ FfiDestroyerRecommendedFees) Destroy(value RecommendedFees) {
	value.Destroy()
}

type RedeemOnchainFundsRequest struct {
	ToAddress   string
	SatPerVbyte uint32
}

func (r *RedeemOnchainFundsRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.ToAddress)
	FfiDestroyerUint32{}.Destroy(r.SatPerVbyte)
}

type FfiConverterRedeemOnchainFundsRequest struct{}

var FfiConverterRedeemOnchainFundsRequestINSTANCE = FfiConverterRedeemOnchainFundsRequest{}

func (c FfiConverterRedeemOnchainFundsRequest) Lift(rb RustBufferI) RedeemOnchainFundsRequest {
	return LiftFromRustBuffer[RedeemOnchainFundsRequest](c, rb)
}

func (c FfiConverterRedeemOnchainFundsRequest) Read(reader io.Reader) RedeemOnchainFundsRequest {
	return RedeemOnchainFundsRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterRedeemOnchainFundsRequest) Lower(value RedeemOnchainFundsRequest) C.RustBuffer {
	return LowerIntoRustBuffer[RedeemOnchainFundsRequest](c, value)
}

func (c FfiConverterRedeemOnchainFundsRequest) Write(writer io.Writer, value RedeemOnchainFundsRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.ToAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.SatPerVbyte)
}

type FfiDestroyerRedeemOnchainFundsRequest struct{}

func (_ FfiDestroyerRedeemOnchainFundsRequest) Destroy(value RedeemOnchainFundsRequest) {
	value.Destroy()
}

type RedeemOnchainFundsResponse struct {
	Txid []uint8
}

func (r *RedeemOnchainFundsResponse) Destroy() {
	FfiDestroyerSequenceUint8{}.Destroy(r.Txid)
}

type FfiConverterRedeemOnchainFundsResponse struct{}

var FfiConverterRedeemOnchainFundsResponseINSTANCE = FfiConverterRedeemOnchainFundsResponse{}

func (c FfiConverterRedeemOnchainFundsResponse) Lift(rb RustBufferI) RedeemOnchainFundsResponse {
	return LiftFromRustBuffer[RedeemOnchainFundsResponse](c, rb)
}

func (c FfiConverterRedeemOnchainFundsResponse) Read(reader io.Reader) RedeemOnchainFundsResponse {
	return RedeemOnchainFundsResponse{
		FfiConverterSequenceUint8INSTANCE.Read(reader),
	}
}

func (c FfiConverterRedeemOnchainFundsResponse) Lower(value RedeemOnchainFundsResponse) C.RustBuffer {
	return LowerIntoRustBuffer[RedeemOnchainFundsResponse](c, value)
}

func (c FfiConverterRedeemOnchainFundsResponse) Write(writer io.Writer, value RedeemOnchainFundsResponse) {
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Txid)
}

type FfiDestroyerRedeemOnchainFundsResponse struct{}

func (_ FfiDestroyerRedeemOnchainFundsResponse) Destroy(value RedeemOnchainFundsResponse) {
	value.Destroy()
}

type RefundRequest struct {
	SwapAddress string
	ToAddress   string
	SatPerVbyte uint32
	Unilateral  *bool
}

func (r *RefundRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.SwapAddress)
	FfiDestroyerString{}.Destroy(r.ToAddress)
	FfiDestroyerUint32{}.Destroy(r.SatPerVbyte)
	FfiDestroyerOptionalBool{}.Destroy(r.Unilateral)
}

type FfiConverterRefundRequest struct{}

var FfiConverterRefundRequestINSTANCE = FfiConverterRefundRequest{}

func (c FfiConverterRefundRequest) Lift(rb RustBufferI) RefundRequest {
	return LiftFromRustBuffer[RefundRequest](c, rb)
}

func (c FfiConverterRefundRequest) Read(reader io.Reader) RefundRequest {
	return RefundRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterRefundRequest) Lower(value RefundRequest) C.RustBuffer {
	return LowerIntoRustBuffer[RefundRequest](c, value)
}

func (c FfiConverterRefundRequest) Write(writer io.Writer, value RefundRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapAddress)
	FfiConverterStringINSTANCE.Write(writer, value.ToAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.SatPerVbyte)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.Unilateral)
}

type FfiDestroyerRefundRequest struct{}

func (_ FfiDestroyerRefundRequest) Destroy(value RefundRequest) {
	value.Destroy()
}

type RefundResponse struct {
	RefundTxId string
}

func (r *RefundResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.RefundTxId)
}

type FfiConverterRefundResponse struct{}

var FfiConverterRefundResponseINSTANCE = FfiConverterRefundResponse{}

func (c FfiConverterRefundResponse) Lift(rb RustBufferI) RefundResponse {
	return LiftFromRustBuffer[RefundResponse](c, rb)
}

func (c FfiConverterRefundResponse) Read(reader io.Reader) RefundResponse {
	return RefundResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterRefundResponse) Lower(value RefundResponse) C.RustBuffer {
	return LowerIntoRustBuffer[RefundResponse](c, value)
}

func (c FfiConverterRefundResponse) Write(writer io.Writer, value RefundResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.RefundTxId)
}

type FfiDestroyerRefundResponse struct{}

func (_ FfiDestroyerRefundResponse) Destroy(value RefundResponse) {
	value.Destroy()
}

type ReportPaymentFailureDetails struct {
	PaymentHash string
	Comment     *string
}

func (r *ReportPaymentFailureDetails) Destroy() {
	FfiDestroyerString{}.Destroy(r.PaymentHash)
	FfiDestroyerOptionalString{}.Destroy(r.Comment)
}

type FfiConverterReportPaymentFailureDetails struct{}

var FfiConverterReportPaymentFailureDetailsINSTANCE = FfiConverterReportPaymentFailureDetails{}

func (c FfiConverterReportPaymentFailureDetails) Lift(rb RustBufferI) ReportPaymentFailureDetails {
	return LiftFromRustBuffer[ReportPaymentFailureDetails](c, rb)
}

func (c FfiConverterReportPaymentFailureDetails) Read(reader io.Reader) ReportPaymentFailureDetails {
	return ReportPaymentFailureDetails{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterReportPaymentFailureDetails) Lower(value ReportPaymentFailureDetails) C.RustBuffer {
	return LowerIntoRustBuffer[ReportPaymentFailureDetails](c, value)
}

func (c FfiConverterReportPaymentFailureDetails) Write(writer io.Writer, value ReportPaymentFailureDetails) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Comment)
}

type FfiDestroyerReportPaymentFailureDetails struct{}

func (_ FfiDestroyerReportPaymentFailureDetails) Destroy(value ReportPaymentFailureDetails) {
	value.Destroy()
}

type ReverseSwapFeesRequest struct {
	SendAmountSat  *uint64
	ClaimTxFeerate *uint32
}

func (r *ReverseSwapFeesRequest) Destroy() {
	FfiDestroyerOptionalUint64{}.Destroy(r.SendAmountSat)
	FfiDestroyerOptionalUint32{}.Destroy(r.ClaimTxFeerate)
}

type FfiConverterReverseSwapFeesRequest struct{}

var FfiConverterReverseSwapFeesRequestINSTANCE = FfiConverterReverseSwapFeesRequest{}

func (c FfiConverterReverseSwapFeesRequest) Lift(rb RustBufferI) ReverseSwapFeesRequest {
	return LiftFromRustBuffer[ReverseSwapFeesRequest](c, rb)
}

func (c FfiConverterReverseSwapFeesRequest) Read(reader io.Reader) ReverseSwapFeesRequest {
	return ReverseSwapFeesRequest{
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterReverseSwapFeesRequest) Lower(value ReverseSwapFeesRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ReverseSwapFeesRequest](c, value)
}

func (c FfiConverterReverseSwapFeesRequest) Write(writer io.Writer, value ReverseSwapFeesRequest) {
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.SendAmountSat)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.ClaimTxFeerate)
}

type FfiDestroyerReverseSwapFeesRequest struct{}

func (_ FfiDestroyerReverseSwapFeesRequest) Destroy(value ReverseSwapFeesRequest) {
	value.Destroy()
}

type ReverseSwapInfo struct {
	Id               string
	ClaimPubkey      string
	LockupTxid       *string
	ClaimTxid        *string
	OnchainAmountSat uint64
	Status           ReverseSwapStatus
}

func (r *ReverseSwapInfo) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerString{}.Destroy(r.ClaimPubkey)
	FfiDestroyerOptionalString{}.Destroy(r.LockupTxid)
	FfiDestroyerOptionalString{}.Destroy(r.ClaimTxid)
	FfiDestroyerUint64{}.Destroy(r.OnchainAmountSat)
	FfiDestroyerReverseSwapStatus{}.Destroy(r.Status)
}

type FfiConverterReverseSwapInfo struct{}

var FfiConverterReverseSwapInfoINSTANCE = FfiConverterReverseSwapInfo{}

func (c FfiConverterReverseSwapInfo) Lift(rb RustBufferI) ReverseSwapInfo {
	return LiftFromRustBuffer[ReverseSwapInfo](c, rb)
}

func (c FfiConverterReverseSwapInfo) Read(reader io.Reader) ReverseSwapInfo {
	return ReverseSwapInfo{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterReverseSwapStatusINSTANCE.Read(reader),
	}
}

func (c FfiConverterReverseSwapInfo) Lower(value ReverseSwapInfo) C.RustBuffer {
	return LowerIntoRustBuffer[ReverseSwapInfo](c, value)
}

func (c FfiConverterReverseSwapInfo) Write(writer io.Writer, value ReverseSwapInfo) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterStringINSTANCE.Write(writer, value.ClaimPubkey)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LockupTxid)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.ClaimTxid)
	FfiConverterUint64INSTANCE.Write(writer, value.OnchainAmountSat)
	FfiConverterReverseSwapStatusINSTANCE.Write(writer, value.Status)
}

type FfiDestroyerReverseSwapInfo struct{}

func (_ FfiDestroyerReverseSwapInfo) Destroy(value ReverseSwapInfo) {
	value.Destroy()
}

type ReverseSwapPairInfo struct {
	Min            uint64
	Max            uint64
	FeesHash       string
	FeesPercentage float64
	FeesLockup     uint64
	FeesClaim      uint64
	TotalFees      *uint64
}

func (r *ReverseSwapPairInfo) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.Min)
	FfiDestroyerUint64{}.Destroy(r.Max)
	FfiDestroyerString{}.Destroy(r.FeesHash)
	FfiDestroyerFloat64{}.Destroy(r.FeesPercentage)
	FfiDestroyerUint64{}.Destroy(r.FeesLockup)
	FfiDestroyerUint64{}.Destroy(r.FeesClaim)
	FfiDestroyerOptionalUint64{}.Destroy(r.TotalFees)
}

type FfiConverterReverseSwapPairInfo struct{}

var FfiConverterReverseSwapPairInfoINSTANCE = FfiConverterReverseSwapPairInfo{}

func (c FfiConverterReverseSwapPairInfo) Lift(rb RustBufferI) ReverseSwapPairInfo {
	return LiftFromRustBuffer[ReverseSwapPairInfo](c, rb)
}

func (c FfiConverterReverseSwapPairInfo) Read(reader io.Reader) ReverseSwapPairInfo {
	return ReverseSwapPairInfo{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFloat64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterReverseSwapPairInfo) Lower(value ReverseSwapPairInfo) C.RustBuffer {
	return LowerIntoRustBuffer[ReverseSwapPairInfo](c, value)
}

func (c FfiConverterReverseSwapPairInfo) Write(writer io.Writer, value ReverseSwapPairInfo) {
	FfiConverterUint64INSTANCE.Write(writer, value.Min)
	FfiConverterUint64INSTANCE.Write(writer, value.Max)
	FfiConverterStringINSTANCE.Write(writer, value.FeesHash)
	FfiConverterFloat64INSTANCE.Write(writer, value.FeesPercentage)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesLockup)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesClaim)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.TotalFees)
}

type FfiDestroyerReverseSwapPairInfo struct{}

func (_ FfiDestroyerReverseSwapPairInfo) Destroy(value ReverseSwapPairInfo) {
	value.Destroy()
}

type RouteHint struct {
	Hops []RouteHintHop
}

func (r *RouteHint) Destroy() {
	FfiDestroyerSequenceRouteHintHop{}.Destroy(r.Hops)
}

type FfiConverterRouteHint struct{}

var FfiConverterRouteHintINSTANCE = FfiConverterRouteHint{}

func (c FfiConverterRouteHint) Lift(rb RustBufferI) RouteHint {
	return LiftFromRustBuffer[RouteHint](c, rb)
}

func (c FfiConverterRouteHint) Read(reader io.Reader) RouteHint {
	return RouteHint{
		FfiConverterSequenceRouteHintHopINSTANCE.Read(reader),
	}
}

func (c FfiConverterRouteHint) Lower(value RouteHint) C.RustBuffer {
	return LowerIntoRustBuffer[RouteHint](c, value)
}

func (c FfiConverterRouteHint) Write(writer io.Writer, value RouteHint) {
	FfiConverterSequenceRouteHintHopINSTANCE.Write(writer, value.Hops)
}

type FfiDestroyerRouteHint struct{}

func (_ FfiDestroyerRouteHint) Destroy(value RouteHint) {
	value.Destroy()
}

type RouteHintHop struct {
	SrcNodeId                  string
	ShortChannelId             string
	FeesBaseMsat               uint32
	FeesProportionalMillionths uint32
	CltvExpiryDelta            uint64
	HtlcMinimumMsat            *uint64
	HtlcMaximumMsat            *uint64
}

func (r *RouteHintHop) Destroy() {
	FfiDestroyerString{}.Destroy(r.SrcNodeId)
	FfiDestroyerString{}.Destroy(r.ShortChannelId)
	FfiDestroyerUint32{}.Destroy(r.FeesBaseMsat)
	FfiDestroyerUint32{}.Destroy(r.FeesProportionalMillionths)
	FfiDestroyerUint64{}.Destroy(r.CltvExpiryDelta)
	FfiDestroyerOptionalUint64{}.Destroy(r.HtlcMinimumMsat)
	FfiDestroyerOptionalUint64{}.Destroy(r.HtlcMaximumMsat)
}

type FfiConverterRouteHintHop struct{}

var FfiConverterRouteHintHopINSTANCE = FfiConverterRouteHintHop{}

func (c FfiConverterRouteHintHop) Lift(rb RustBufferI) RouteHintHop {
	return LiftFromRustBuffer[RouteHintHop](c, rb)
}

func (c FfiConverterRouteHintHop) Read(reader io.Reader) RouteHintHop {
	return RouteHintHop{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterRouteHintHop) Lower(value RouteHintHop) C.RustBuffer {
	return LowerIntoRustBuffer[RouteHintHop](c, value)
}

func (c FfiConverterRouteHintHop) Write(writer io.Writer, value RouteHintHop) {
	FfiConverterStringINSTANCE.Write(writer, value.SrcNodeId)
	FfiConverterStringINSTANCE.Write(writer, value.ShortChannelId)
	FfiConverterUint32INSTANCE.Write(writer, value.FeesBaseMsat)
	FfiConverterUint32INSTANCE.Write(writer, value.FeesProportionalMillionths)
	FfiConverterUint64INSTANCE.Write(writer, value.CltvExpiryDelta)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.HtlcMinimumMsat)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.HtlcMaximumMsat)
}

type FfiDestroyerRouteHintHop struct{}

func (_ FfiDestroyerRouteHintHop) Destroy(value RouteHintHop) {
	value.Destroy()
}

type SendPaymentRequest struct {
	Bolt11        string
	UseTrampoline bool
	AmountMsat    *uint64
	Label         *string
}

func (r *SendPaymentRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Bolt11)
	FfiDestroyerBool{}.Destroy(r.UseTrampoline)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
}

type FfiConverterSendPaymentRequest struct{}

var FfiConverterSendPaymentRequestINSTANCE = FfiConverterSendPaymentRequest{}

func (c FfiConverterSendPaymentRequest) Lift(rb RustBufferI) SendPaymentRequest {
	return LiftFromRustBuffer[SendPaymentRequest](c, rb)
}

func (c FfiConverterSendPaymentRequest) Read(reader io.Reader) SendPaymentRequest {
	return SendPaymentRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterSendPaymentRequest) Lower(value SendPaymentRequest) C.RustBuffer {
	return LowerIntoRustBuffer[SendPaymentRequest](c, value)
}

func (c FfiConverterSendPaymentRequest) Write(writer io.Writer, value SendPaymentRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterBoolINSTANCE.Write(writer, value.UseTrampoline)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
}

type FfiDestroyerSendPaymentRequest struct{}

func (_ FfiDestroyerSendPaymentRequest) Destroy(value SendPaymentRequest) {
	value.Destroy()
}

type SendPaymentResponse struct {
	Payment Payment
}

func (r *SendPaymentResponse) Destroy() {
	FfiDestroyerPayment{}.Destroy(r.Payment)
}

type FfiConverterSendPaymentResponse struct{}

var FfiConverterSendPaymentResponseINSTANCE = FfiConverterSendPaymentResponse{}

func (c FfiConverterSendPaymentResponse) Lift(rb RustBufferI) SendPaymentResponse {
	return LiftFromRustBuffer[SendPaymentResponse](c, rb)
}

func (c FfiConverterSendPaymentResponse) Read(reader io.Reader) SendPaymentResponse {
	return SendPaymentResponse{
		FfiConverterPaymentINSTANCE.Read(reader),
	}
}

func (c FfiConverterSendPaymentResponse) Lower(value SendPaymentResponse) C.RustBuffer {
	return LowerIntoRustBuffer[SendPaymentResponse](c, value)
}

func (c FfiConverterSendPaymentResponse) Write(writer io.Writer, value SendPaymentResponse) {
	FfiConverterPaymentINSTANCE.Write(writer, value.Payment)
}

type FfiDestroyerSendPaymentResponse struct{}

func (_ FfiDestroyerSendPaymentResponse) Destroy(value SendPaymentResponse) {
	value.Destroy()
}

type SendSpontaneousPaymentRequest struct {
	NodeId     string
	AmountMsat uint64
	ExtraTlvs  *[]TlvEntry
	Label      *string
}

func (r *SendSpontaneousPaymentRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.NodeId)
	FfiDestroyerUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerOptionalSequenceTlvEntry{}.Destroy(r.ExtraTlvs)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
}

type FfiConverterSendSpontaneousPaymentRequest struct{}

var FfiConverterSendSpontaneousPaymentRequestINSTANCE = FfiConverterSendSpontaneousPaymentRequest{}

func (c FfiConverterSendSpontaneousPaymentRequest) Lift(rb RustBufferI) SendSpontaneousPaymentRequest {
	return LiftFromRustBuffer[SendSpontaneousPaymentRequest](c, rb)
}

func (c FfiConverterSendSpontaneousPaymentRequest) Read(reader io.Reader) SendSpontaneousPaymentRequest {
	return SendSpontaneousPaymentRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalSequenceTlvEntryINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterSendSpontaneousPaymentRequest) Lower(value SendSpontaneousPaymentRequest) C.RustBuffer {
	return LowerIntoRustBuffer[SendSpontaneousPaymentRequest](c, value)
}

func (c FfiConverterSendSpontaneousPaymentRequest) Write(writer io.Writer, value SendSpontaneousPaymentRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.NodeId)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalSequenceTlvEntryINSTANCE.Write(writer, value.ExtraTlvs)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
}

type FfiDestroyerSendSpontaneousPaymentRequest struct{}

func (_ FfiDestroyerSendSpontaneousPaymentRequest) Destroy(value SendSpontaneousPaymentRequest) {
	value.Destroy()
}

type ServiceHealthCheckResponse struct {
	Status HealthCheckStatus
}

func (r *ServiceHealthCheckResponse) Destroy() {
	FfiDestroyerHealthCheckStatus{}.Destroy(r.Status)
}

type FfiConverterServiceHealthCheckResponse struct{}

var FfiConverterServiceHealthCheckResponseINSTANCE = FfiConverterServiceHealthCheckResponse{}

func (c FfiConverterServiceHealthCheckResponse) Lift(rb RustBufferI) ServiceHealthCheckResponse {
	return LiftFromRustBuffer[ServiceHealthCheckResponse](c, rb)
}

func (c FfiConverterServiceHealthCheckResponse) Read(reader io.Reader) ServiceHealthCheckResponse {
	return ServiceHealthCheckResponse{
		FfiConverterHealthCheckStatusINSTANCE.Read(reader),
	}
}

func (c FfiConverterServiceHealthCheckResponse) Lower(value ServiceHealthCheckResponse) C.RustBuffer {
	return LowerIntoRustBuffer[ServiceHealthCheckResponse](c, value)
}

func (c FfiConverterServiceHealthCheckResponse) Write(writer io.Writer, value ServiceHealthCheckResponse) {
	FfiConverterHealthCheckStatusINSTANCE.Write(writer, value.Status)
}

type FfiDestroyerServiceHealthCheckResponse struct{}

func (_ FfiDestroyerServiceHealthCheckResponse) Destroy(value ServiceHealthCheckResponse) {
	value.Destroy()
}

type SignMessageRequest struct {
	Message string
}

func (r *SignMessageRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Message)
}

type FfiConverterSignMessageRequest struct{}

var FfiConverterSignMessageRequestINSTANCE = FfiConverterSignMessageRequest{}

func (c FfiConverterSignMessageRequest) Lift(rb RustBufferI) SignMessageRequest {
	return LiftFromRustBuffer[SignMessageRequest](c, rb)
}

func (c FfiConverterSignMessageRequest) Read(reader io.Reader) SignMessageRequest {
	return SignMessageRequest{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterSignMessageRequest) Lower(value SignMessageRequest) C.RustBuffer {
	return LowerIntoRustBuffer[SignMessageRequest](c, value)
}

func (c FfiConverterSignMessageRequest) Write(writer io.Writer, value SignMessageRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerSignMessageRequest struct{}

func (_ FfiDestroyerSignMessageRequest) Destroy(value SignMessageRequest) {
	value.Destroy()
}

type SignMessageResponse struct {
	Signature string
}

func (r *SignMessageResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Signature)
}

type FfiConverterSignMessageResponse struct{}

var FfiConverterSignMessageResponseINSTANCE = FfiConverterSignMessageResponse{}

func (c FfiConverterSignMessageResponse) Lift(rb RustBufferI) SignMessageResponse {
	return LiftFromRustBuffer[SignMessageResponse](c, rb)
}

func (c FfiConverterSignMessageResponse) Read(reader io.Reader) SignMessageResponse {
	return SignMessageResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterSignMessageResponse) Lower(value SignMessageResponse) C.RustBuffer {
	return LowerIntoRustBuffer[SignMessageResponse](c, value)
}

func (c FfiConverterSignMessageResponse) Write(writer io.Writer, value SignMessageResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Signature)
}

type FfiDestroyerSignMessageResponse struct{}

func (_ FfiDestroyerSignMessageResponse) Destroy(value SignMessageResponse) {
	value.Destroy()
}

type StaticBackupRequest struct {
	WorkingDir string
}

func (r *StaticBackupRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.WorkingDir)
}

type FfiConverterStaticBackupRequest struct{}

var FfiConverterStaticBackupRequestINSTANCE = FfiConverterStaticBackupRequest{}

func (c FfiConverterStaticBackupRequest) Lift(rb RustBufferI) StaticBackupRequest {
	return LiftFromRustBuffer[StaticBackupRequest](c, rb)
}

func (c FfiConverterStaticBackupRequest) Read(reader io.Reader) StaticBackupRequest {
	return StaticBackupRequest{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterStaticBackupRequest) Lower(value StaticBackupRequest) C.RustBuffer {
	return LowerIntoRustBuffer[StaticBackupRequest](c, value)
}

func (c FfiConverterStaticBackupRequest) Write(writer io.Writer, value StaticBackupRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.WorkingDir)
}

type FfiDestroyerStaticBackupRequest struct{}

func (_ FfiDestroyerStaticBackupRequest) Destroy(value StaticBackupRequest) {
	value.Destroy()
}

type StaticBackupResponse struct {
	Backup *[]string
}

func (r *StaticBackupResponse) Destroy() {
	FfiDestroyerOptionalSequenceString{}.Destroy(r.Backup)
}

type FfiConverterStaticBackupResponse struct{}

var FfiConverterStaticBackupResponseINSTANCE = FfiConverterStaticBackupResponse{}

func (c FfiConverterStaticBackupResponse) Lift(rb RustBufferI) StaticBackupResponse {
	return LiftFromRustBuffer[StaticBackupResponse](c, rb)
}

func (c FfiConverterStaticBackupResponse) Read(reader io.Reader) StaticBackupResponse {
	return StaticBackupResponse{
		FfiConverterOptionalSequenceStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterStaticBackupResponse) Lower(value StaticBackupResponse) C.RustBuffer {
	return LowerIntoRustBuffer[StaticBackupResponse](c, value)
}

func (c FfiConverterStaticBackupResponse) Write(writer io.Writer, value StaticBackupResponse) {
	FfiConverterOptionalSequenceStringINSTANCE.Write(writer, value.Backup)
}

type FfiDestroyerStaticBackupResponse struct{}

func (_ FfiDestroyerStaticBackupResponse) Destroy(value StaticBackupResponse) {
	value.Destroy()
}

type SwapInfo struct {
	BitcoinAddress     string
	CreatedAt          int64
	LockHeight         int64
	PaymentHash        []uint8
	Preimage           []uint8
	PrivateKey         []uint8
	PublicKey          []uint8
	SwapperPublicKey   []uint8
	Script             []uint8
	Bolt11             *string
	PaidMsat           uint64
	UnconfirmedSats    uint64
	ConfirmedSats      uint64
	TotalIncomingTxs   uint64
	Status             SwapStatus
	RefundTxIds        []string
	UnconfirmedTxIds   []string
	ConfirmedTxIds     []string
	MinAllowedDeposit  int64
	MaxAllowedDeposit  int64
	MaxSwapperPayable  int64
	LastRedeemError    *string
	ChannelOpeningFees *OpeningFeeParams
	ConfirmedAt        *uint32
}

func (r *SwapInfo) Destroy() {
	FfiDestroyerString{}.Destroy(r.BitcoinAddress)
	FfiDestroyerInt64{}.Destroy(r.CreatedAt)
	FfiDestroyerInt64{}.Destroy(r.LockHeight)
	FfiDestroyerSequenceUint8{}.Destroy(r.PaymentHash)
	FfiDestroyerSequenceUint8{}.Destroy(r.Preimage)
	FfiDestroyerSequenceUint8{}.Destroy(r.PrivateKey)
	FfiDestroyerSequenceUint8{}.Destroy(r.PublicKey)
	FfiDestroyerSequenceUint8{}.Destroy(r.SwapperPublicKey)
	FfiDestroyerSequenceUint8{}.Destroy(r.Script)
	FfiDestroyerOptionalString{}.Destroy(r.Bolt11)
	FfiDestroyerUint64{}.Destroy(r.PaidMsat)
	FfiDestroyerUint64{}.Destroy(r.UnconfirmedSats)
	FfiDestroyerUint64{}.Destroy(r.ConfirmedSats)
	FfiDestroyerUint64{}.Destroy(r.TotalIncomingTxs)
	FfiDestroyerSwapStatus{}.Destroy(r.Status)
	FfiDestroyerSequenceString{}.Destroy(r.RefundTxIds)
	FfiDestroyerSequenceString{}.Destroy(r.UnconfirmedTxIds)
	FfiDestroyerSequenceString{}.Destroy(r.ConfirmedTxIds)
	FfiDestroyerInt64{}.Destroy(r.MinAllowedDeposit)
	FfiDestroyerInt64{}.Destroy(r.MaxAllowedDeposit)
	FfiDestroyerInt64{}.Destroy(r.MaxSwapperPayable)
	FfiDestroyerOptionalString{}.Destroy(r.LastRedeemError)
	FfiDestroyerOptionalOpeningFeeParams{}.Destroy(r.ChannelOpeningFees)
	FfiDestroyerOptionalUint32{}.Destroy(r.ConfirmedAt)
}

type FfiConverterSwapInfo struct{}

var FfiConverterSwapInfoINSTANCE = FfiConverterSwapInfo{}

func (c FfiConverterSwapInfo) Lift(rb RustBufferI) SwapInfo {
	return LiftFromRustBuffer[SwapInfo](c, rb)
}

func (c FfiConverterSwapInfo) Read(reader io.Reader) SwapInfo {
	return SwapInfo{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSwapStatusINSTANCE.Read(reader),
		FfiConverterSequenceStringINSTANCE.Read(reader),
		FfiConverterSequenceStringINSTANCE.Read(reader),
		FfiConverterSequenceStringINSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalOpeningFeeParamsINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterSwapInfo) Lower(value SwapInfo) C.RustBuffer {
	return LowerIntoRustBuffer[SwapInfo](c, value)
}

func (c FfiConverterSwapInfo) Write(writer io.Writer, value SwapInfo) {
	FfiConverterStringINSTANCE.Write(writer, value.BitcoinAddress)
	FfiConverterInt64INSTANCE.Write(writer, value.CreatedAt)
	FfiConverterInt64INSTANCE.Write(writer, value.LockHeight)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.PaymentHash)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Preimage)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.PrivateKey)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.PublicKey)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.SwapperPublicKey)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Script)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterUint64INSTANCE.Write(writer, value.PaidMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.UnconfirmedSats)
	FfiConverterUint64INSTANCE.Write(writer, value.ConfirmedSats)
	FfiConverterUint64INSTANCE.Write(writer, value.TotalIncomingTxs)
	FfiConverterSwapStatusINSTANCE.Write(writer, value.Status)
	FfiConverterSequenceStringINSTANCE.Write(writer, value.RefundTxIds)
	FfiConverterSequenceStringINSTANCE.Write(writer, value.UnconfirmedTxIds)
	FfiConverterSequenceStringINSTANCE.Write(writer, value.ConfirmedTxIds)
	FfiConverterInt64INSTANCE.Write(writer, value.MinAllowedDeposit)
	FfiConverterInt64INSTANCE.Write(writer, value.MaxAllowedDeposit)
	FfiConverterInt64INSTANCE.Write(writer, value.MaxSwapperPayable)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LastRedeemError)
	FfiConverterOptionalOpeningFeeParamsINSTANCE.Write(writer, value.ChannelOpeningFees)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.ConfirmedAt)
}

type FfiDestroyerSwapInfo struct{}

func (_ FfiDestroyerSwapInfo) Destroy(value SwapInfo) {
	value.Destroy()
}

type Symbol struct {
	Grapheme *string
	Template *string
	Rtl      *bool
	Position *uint32
}

func (r *Symbol) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.Grapheme)
	FfiDestroyerOptionalString{}.Destroy(r.Template)
	FfiDestroyerOptionalBool{}.Destroy(r.Rtl)
	FfiDestroyerOptionalUint32{}.Destroy(r.Position)
}

type FfiConverterSymbol struct{}

var FfiConverterSymbolINSTANCE = FfiConverterSymbol{}

func (c FfiConverterSymbol) Lift(rb RustBufferI) Symbol {
	return LiftFromRustBuffer[Symbol](c, rb)
}

func (c FfiConverterSymbol) Read(reader io.Reader) Symbol {
	return Symbol{
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterSymbol) Lower(value Symbol) C.RustBuffer {
	return LowerIntoRustBuffer[Symbol](c, value)
}

func (c FfiConverterSymbol) Write(writer io.Writer, value Symbol) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Grapheme)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Template)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.Rtl)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Position)
}

type FfiDestroyerSymbol struct{}

func (_ FfiDestroyerSymbol) Destroy(value Symbol) {
	value.Destroy()
}

type TlvEntry struct {
	FieldNumber uint64
	Value       []uint8
}

func (r *TlvEntry) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.FieldNumber)
	FfiDestroyerSequenceUint8{}.Destroy(r.Value)
}

type FfiConverterTlvEntry struct{}

var FfiConverterTlvEntryINSTANCE = FfiConverterTlvEntry{}

func (c FfiConverterTlvEntry) Lift(rb RustBufferI) TlvEntry {
	return LiftFromRustBuffer[TlvEntry](c, rb)
}

func (c FfiConverterTlvEntry) Read(reader io.Reader) TlvEntry {
	return TlvEntry{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
	}
}

func (c FfiConverterTlvEntry) Lower(value TlvEntry) C.RustBuffer {
	return LowerIntoRustBuffer[TlvEntry](c, value)
}

func (c FfiConverterTlvEntry) Write(writer io.Writer, value TlvEntry) {
	FfiConverterUint64INSTANCE.Write(writer, value.FieldNumber)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Value)
}

type FfiDestroyerTlvEntry struct{}

func (_ FfiDestroyerTlvEntry) Destroy(value TlvEntry) {
	value.Destroy()
}

type UnspentTransactionOutput struct {
	Txid               []uint8
	Outnum             uint32
	AmountMillisatoshi uint64
	Address            string
	Reserved           bool
}

func (r *UnspentTransactionOutput) Destroy() {
	FfiDestroyerSequenceUint8{}.Destroy(r.Txid)
	FfiDestroyerUint32{}.Destroy(r.Outnum)
	FfiDestroyerUint64{}.Destroy(r.AmountMillisatoshi)
	FfiDestroyerString{}.Destroy(r.Address)
	FfiDestroyerBool{}.Destroy(r.Reserved)
}

type FfiConverterUnspentTransactionOutput struct{}

var FfiConverterUnspentTransactionOutputINSTANCE = FfiConverterUnspentTransactionOutput{}

func (c FfiConverterUnspentTransactionOutput) Lift(rb RustBufferI) UnspentTransactionOutput {
	return LiftFromRustBuffer[UnspentTransactionOutput](c, rb)
}

func (c FfiConverterUnspentTransactionOutput) Read(reader io.Reader) UnspentTransactionOutput {
	return UnspentTransactionOutput{
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterUnspentTransactionOutput) Lower(value UnspentTransactionOutput) C.RustBuffer {
	return LowerIntoRustBuffer[UnspentTransactionOutput](c, value)
}

func (c FfiConverterUnspentTransactionOutput) Write(writer io.Writer, value UnspentTransactionOutput) {
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Txid)
	FfiConverterUint32INSTANCE.Write(writer, value.Outnum)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMillisatoshi)
	FfiConverterStringINSTANCE.Write(writer, value.Address)
	FfiConverterBoolINSTANCE.Write(writer, value.Reserved)
}

type FfiDestroyerUnspentTransactionOutput struct{}

func (_ FfiDestroyerUnspentTransactionOutput) Destroy(value UnspentTransactionOutput) {
	value.Destroy()
}

type UrlSuccessActionData struct {
	Description           string
	Url                   string
	MatchesCallbackDomain bool
}

func (r *UrlSuccessActionData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Description)
	FfiDestroyerString{}.Destroy(r.Url)
	FfiDestroyerBool{}.Destroy(r.MatchesCallbackDomain)
}

type FfiConverterUrlSuccessActionData struct{}

var FfiConverterUrlSuccessActionDataINSTANCE = FfiConverterUrlSuccessActionData{}

func (c FfiConverterUrlSuccessActionData) Lift(rb RustBufferI) UrlSuccessActionData {
	return LiftFromRustBuffer[UrlSuccessActionData](c, rb)
}

func (c FfiConverterUrlSuccessActionData) Read(reader io.Reader) UrlSuccessActionData {
	return UrlSuccessActionData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterUrlSuccessActionData) Lower(value UrlSuccessActionData) C.RustBuffer {
	return LowerIntoRustBuffer[UrlSuccessActionData](c, value)
}

func (c FfiConverterUrlSuccessActionData) Write(writer io.Writer, value UrlSuccessActionData) {
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.Url)
	FfiConverterBoolINSTANCE.Write(writer, value.MatchesCallbackDomain)
}

type FfiDestroyerUrlSuccessActionData struct{}

func (_ FfiDestroyerUrlSuccessActionData) Destroy(value UrlSuccessActionData) {
	value.Destroy()
}

type AesSuccessActionDataResult interface {
	Destroy()
}
type AesSuccessActionDataResultDecrypted struct {
	Data AesSuccessActionDataDecrypted
}

func (e AesSuccessActionDataResultDecrypted) Destroy() {
	FfiDestroyerAesSuccessActionDataDecrypted{}.Destroy(e.Data)
}

type AesSuccessActionDataResultErrorStatus struct {
	Reason string
}

func (e AesSuccessActionDataResultErrorStatus) Destroy() {
	FfiDestroyerString{}.Destroy(e.Reason)
}

type FfiConverterAesSuccessActionDataResult struct{}

var FfiConverterAesSuccessActionDataResultINSTANCE = FfiConverterAesSuccessActionDataResult{}

func (c FfiConverterAesSuccessActionDataResult) Lift(rb RustBufferI) AesSuccessActionDataResult {
	return LiftFromRustBuffer[AesSuccessActionDataResult](c, rb)
}

func (c FfiConverterAesSuccessActionDataResult) Lower(value AesSuccessActionDataResult) C.RustBuffer {
	return LowerIntoRustBuffer[AesSuccessActionDataResult](c, value)
}
func (FfiConverterAesSuccessActionDataResult) Read(reader io.Reader) AesSuccessActionDataResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return AesSuccessActionDataResultDecrypted{
			FfiConverterAesSuccessActionDataDecryptedINSTANCE.Read(reader),
		}
	case 2:
		return AesSuccessActionDataResultErrorStatus{
			FfiConverterStringINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterAesSuccessActionDataResult.Read()", id))
	}
}

func (FfiConverterAesSuccessActionDataResult) Write(writer io.Writer, value AesSuccessActionDataResult) {
	switch variant_value := value.(type) {
	case AesSuccessActionDataResultDecrypted:
		writeInt32(writer, 1)
		FfiConverterAesSuccessActionDataDecryptedINSTANCE.Write(writer, variant_value.Data)
	case AesSuccessActionDataResultErrorStatus:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Reason)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterAesSuccessActionDataResult.Write", value))
	}
}

type FfiDestroyerAesSuccessActionDataResult struct{}

func (_ FfiDestroyerAesSuccessActionDataResult) Destroy(value AesSuccessActionDataResult) {
	value.Destroy()
}

type BreezEvent interface {
	Destroy()
}
type BreezEventNewBlock struct {
	Block uint32
}

func (e BreezEventNewBlock) Destroy() {
	FfiDestroyerUint32{}.Destroy(e.Block)
}

type BreezEventInvoicePaid struct {
	Details InvoicePaidDetails
}

func (e BreezEventInvoicePaid) Destroy() {
	FfiDestroyerInvoicePaidDetails{}.Destroy(e.Details)
}

type BreezEventSynced struct {
}

func (e BreezEventSynced) Destroy() {
}

type BreezEventPaymentSucceed struct {
	Details Payment
}

func (e BreezEventPaymentSucceed) Destroy() {
	FfiDestroyerPayment{}.Destroy(e.Details)
}

type BreezEventPaymentFailed struct {
	Details PaymentFailedData
}

func (e BreezEventPaymentFailed) Destroy() {
	FfiDestroyerPaymentFailedData{}.Destroy(e.Details)
}

type BreezEventBackupStarted struct {
}

func (e BreezEventBackupStarted) Destroy() {
}

type BreezEventBackupSucceeded struct {
}

func (e BreezEventBackupSucceeded) Destroy() {
}

type BreezEventBackupFailed struct {
	Details BackupFailedData
}

func (e BreezEventBackupFailed) Destroy() {
	FfiDestroyerBackupFailedData{}.Destroy(e.Details)
}

type BreezEventReverseSwapUpdated struct {
	Details ReverseSwapInfo
}

func (e BreezEventReverseSwapUpdated) Destroy() {
	FfiDestroyerReverseSwapInfo{}.Destroy(e.Details)
}

type BreezEventSwapUpdated struct {
	Details SwapInfo
}

func (e BreezEventSwapUpdated) Destroy() {
	FfiDestroyerSwapInfo{}.Destroy(e.Details)
}

type FfiConverterBreezEvent struct{}

var FfiConverterBreezEventINSTANCE = FfiConverterBreezEvent{}

func (c FfiConverterBreezEvent) Lift(rb RustBufferI) BreezEvent {
	return LiftFromRustBuffer[BreezEvent](c, rb)
}

func (c FfiConverterBreezEvent) Lower(value BreezEvent) C.RustBuffer {
	return LowerIntoRustBuffer[BreezEvent](c, value)
}
func (FfiConverterBreezEvent) Read(reader io.Reader) BreezEvent {
	id := readInt32(reader)
	switch id {
	case 1:
		return BreezEventNewBlock{
			FfiConverterUint32INSTANCE.Read(reader),
		}
	case 2:
		return BreezEventInvoicePaid{
			FfiConverterInvoicePaidDetailsINSTANCE.Read(reader),
		}
	case 3:
		return BreezEventSynced{}
	case 4:
		return BreezEventPaymentSucceed{
			FfiConverterPaymentINSTANCE.Read(reader),
		}
	case 5:
		return BreezEventPaymentFailed{
			FfiConverterPaymentFailedDataINSTANCE.Read(reader),
		}
	case 6:
		return BreezEventBackupStarted{}
	case 7:
		return BreezEventBackupSucceeded{}
	case 8:
		return BreezEventBackupFailed{
			FfiConverterBackupFailedDataINSTANCE.Read(reader),
		}
	case 9:
		return BreezEventReverseSwapUpdated{
			FfiConverterReverseSwapInfoINSTANCE.Read(reader),
		}
	case 10:
		return BreezEventSwapUpdated{
			FfiConverterSwapInfoINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterBreezEvent.Read()", id))
	}
}

func (FfiConverterBreezEvent) Write(writer io.Writer, value BreezEvent) {
	switch variant_value := value.(type) {
	case BreezEventNewBlock:
		writeInt32(writer, 1)
		FfiConverterUint32INSTANCE.Write(writer, variant_value.Block)
	case BreezEventInvoicePaid:
		writeInt32(writer, 2)
		FfiConverterInvoicePaidDetailsINSTANCE.Write(writer, variant_value.Details)
	case BreezEventSynced:
		writeInt32(writer, 3)
	case BreezEventPaymentSucceed:
		writeInt32(writer, 4)
		FfiConverterPaymentINSTANCE.Write(writer, variant_value.Details)
	case BreezEventPaymentFailed:
		writeInt32(writer, 5)
		FfiConverterPaymentFailedDataINSTANCE.Write(writer, variant_value.Details)
	case BreezEventBackupStarted:
		writeInt32(writer, 6)
	case BreezEventBackupSucceeded:
		writeInt32(writer, 7)
	case BreezEventBackupFailed:
		writeInt32(writer, 8)
		FfiConverterBackupFailedDataINSTANCE.Write(writer, variant_value.Details)
	case BreezEventReverseSwapUpdated:
		writeInt32(writer, 9)
		FfiConverterReverseSwapInfoINSTANCE.Write(writer, variant_value.Details)
	case BreezEventSwapUpdated:
		writeInt32(writer, 10)
		FfiConverterSwapInfoINSTANCE.Write(writer, variant_value.Details)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterBreezEvent.Write", value))
	}
}

type FfiDestroyerBreezEvent struct{}

func (_ FfiDestroyerBreezEvent) Destroy(value BreezEvent) {
	value.Destroy()
}

type BuyBitcoinProvider uint

const (
	BuyBitcoinProviderMoonpay BuyBitcoinProvider = 1
)

type FfiConverterBuyBitcoinProvider struct{}

var FfiConverterBuyBitcoinProviderINSTANCE = FfiConverterBuyBitcoinProvider{}

func (c FfiConverterBuyBitcoinProvider) Lift(rb RustBufferI) BuyBitcoinProvider {
	return LiftFromRustBuffer[BuyBitcoinProvider](c, rb)
}

func (c FfiConverterBuyBitcoinProvider) Lower(value BuyBitcoinProvider) C.RustBuffer {
	return LowerIntoRustBuffer[BuyBitcoinProvider](c, value)
}
func (FfiConverterBuyBitcoinProvider) Read(reader io.Reader) BuyBitcoinProvider {
	id := readInt32(reader)
	return BuyBitcoinProvider(id)
}

func (FfiConverterBuyBitcoinProvider) Write(writer io.Writer, value BuyBitcoinProvider) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerBuyBitcoinProvider struct{}

func (_ FfiDestroyerBuyBitcoinProvider) Destroy(value BuyBitcoinProvider) {
}

type ChannelState uint

const (
	ChannelStatePendingOpen  ChannelState = 1
	ChannelStateOpened       ChannelState = 2
	ChannelStatePendingClose ChannelState = 3
	ChannelStateClosed       ChannelState = 4
)

type FfiConverterChannelState struct{}

var FfiConverterChannelStateINSTANCE = FfiConverterChannelState{}

func (c FfiConverterChannelState) Lift(rb RustBufferI) ChannelState {
	return LiftFromRustBuffer[ChannelState](c, rb)
}

func (c FfiConverterChannelState) Lower(value ChannelState) C.RustBuffer {
	return LowerIntoRustBuffer[ChannelState](c, value)
}
func (FfiConverterChannelState) Read(reader io.Reader) ChannelState {
	id := readInt32(reader)
	return ChannelState(id)
}

func (FfiConverterChannelState) Write(writer io.Writer, value ChannelState) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerChannelState struct{}

func (_ FfiDestroyerChannelState) Destroy(value ChannelState) {
}

type ConnectError struct {
	err error
}

// Convience method to turn *ConnectError into error
// Avoiding treating nil pointer as non nil error interface
func (err *ConnectError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err ConnectError) Error() string {
	return fmt.Sprintf("ConnectError: %s", err.err.Error())
}

func (err ConnectError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrConnectErrorGeneric = fmt.Errorf("ConnectErrorGeneric")
var ErrConnectErrorRestoreOnly = fmt.Errorf("ConnectErrorRestoreOnly")
var ErrConnectErrorServiceConnectivity = fmt.Errorf("ConnectErrorServiceConnectivity")

// Variant structs
type ConnectErrorGeneric struct {
	message string
}

func NewConnectErrorGeneric() *ConnectError {
	return &ConnectError{err: &ConnectErrorGeneric{}}
}

func (e ConnectErrorGeneric) destroy() {
}

func (err ConnectErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self ConnectErrorGeneric) Is(target error) bool {
	return target == ErrConnectErrorGeneric
}

type ConnectErrorRestoreOnly struct {
	message string
}

func NewConnectErrorRestoreOnly() *ConnectError {
	return &ConnectError{err: &ConnectErrorRestoreOnly{}}
}

func (e ConnectErrorRestoreOnly) destroy() {
}

func (err ConnectErrorRestoreOnly) Error() string {
	return fmt.Sprintf("RestoreOnly: %s", err.message)
}

func (self ConnectErrorRestoreOnly) Is(target error) bool {
	return target == ErrConnectErrorRestoreOnly
}

type ConnectErrorServiceConnectivity struct {
	message string
}

func NewConnectErrorServiceConnectivity() *ConnectError {
	return &ConnectError{err: &ConnectErrorServiceConnectivity{}}
}

func (e ConnectErrorServiceConnectivity) destroy() {
}

func (err ConnectErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self ConnectErrorServiceConnectivity) Is(target error) bool {
	return target == ErrConnectErrorServiceConnectivity
}

type FfiConverterConnectError struct{}

var FfiConverterConnectErrorINSTANCE = FfiConverterConnectError{}

func (c FfiConverterConnectError) Lift(eb RustBufferI) *ConnectError {
	return LiftFromRustBuffer[*ConnectError](c, eb)
}

func (c FfiConverterConnectError) Lower(value *ConnectError) C.RustBuffer {
	return LowerIntoRustBuffer[*ConnectError](c, value)
}

func (c FfiConverterConnectError) Read(reader io.Reader) *ConnectError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &ConnectError{&ConnectErrorGeneric{message}}
	case 2:
		return &ConnectError{&ConnectErrorRestoreOnly{message}}
	case 3:
		return &ConnectError{&ConnectErrorServiceConnectivity{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterConnectError.Read()", errorID))
	}

}

func (c FfiConverterConnectError) Write(writer io.Writer, value *ConnectError) {
	switch variantValue := value.err.(type) {
	case *ConnectErrorGeneric:
		writeInt32(writer, 1)
	case *ConnectErrorRestoreOnly:
		writeInt32(writer, 2)
	case *ConnectErrorServiceConnectivity:
		writeInt32(writer, 3)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterConnectError.Write", value))
	}
}

type FfiDestroyerConnectError struct{}

func (_ FfiDestroyerConnectError) Destroy(value *ConnectError) {
	switch variantValue := value.err.(type) {
	case ConnectErrorGeneric:
		variantValue.destroy()
	case ConnectErrorRestoreOnly:
		variantValue.destroy()
	case ConnectErrorServiceConnectivity:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerConnectError.Destroy", value))
	}
}

type EnvironmentType uint

const (
	EnvironmentTypeProduction EnvironmentType = 1
	EnvironmentTypeStaging    EnvironmentType = 2
	EnvironmentTypeRegtest    EnvironmentType = 3
)

type FfiConverterEnvironmentType struct{}

var FfiConverterEnvironmentTypeINSTANCE = FfiConverterEnvironmentType{}

func (c FfiConverterEnvironmentType) Lift(rb RustBufferI) EnvironmentType {
	return LiftFromRustBuffer[EnvironmentType](c, rb)
}

func (c FfiConverterEnvironmentType) Lower(value EnvironmentType) C.RustBuffer {
	return LowerIntoRustBuffer[EnvironmentType](c, value)
}
func (FfiConverterEnvironmentType) Read(reader io.Reader) EnvironmentType {
	id := readInt32(reader)
	return EnvironmentType(id)
}

func (FfiConverterEnvironmentType) Write(writer io.Writer, value EnvironmentType) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerEnvironmentType struct{}

func (_ FfiDestroyerEnvironmentType) Destroy(value EnvironmentType) {
}

type FeeratePreset uint

const (
	FeeratePresetRegular  FeeratePreset = 1
	FeeratePresetEconomy  FeeratePreset = 2
	FeeratePresetPriority FeeratePreset = 3
)

type FfiConverterFeeratePreset struct{}

var FfiConverterFeeratePresetINSTANCE = FfiConverterFeeratePreset{}

func (c FfiConverterFeeratePreset) Lift(rb RustBufferI) FeeratePreset {
	return LiftFromRustBuffer[FeeratePreset](c, rb)
}

func (c FfiConverterFeeratePreset) Lower(value FeeratePreset) C.RustBuffer {
	return LowerIntoRustBuffer[FeeratePreset](c, value)
}
func (FfiConverterFeeratePreset) Read(reader io.Reader) FeeratePreset {
	id := readInt32(reader)
	return FeeratePreset(id)
}

func (FfiConverterFeeratePreset) Write(writer io.Writer, value FeeratePreset) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerFeeratePreset struct{}

func (_ FfiDestroyerFeeratePreset) Destroy(value FeeratePreset) {
}

type HealthCheckStatus uint

const (
	HealthCheckStatusOperational       HealthCheckStatus = 1
	HealthCheckStatusMaintenance       HealthCheckStatus = 2
	HealthCheckStatusServiceDisruption HealthCheckStatus = 3
)

type FfiConverterHealthCheckStatus struct{}

var FfiConverterHealthCheckStatusINSTANCE = FfiConverterHealthCheckStatus{}

func (c FfiConverterHealthCheckStatus) Lift(rb RustBufferI) HealthCheckStatus {
	return LiftFromRustBuffer[HealthCheckStatus](c, rb)
}

func (c FfiConverterHealthCheckStatus) Lower(value HealthCheckStatus) C.RustBuffer {
	return LowerIntoRustBuffer[HealthCheckStatus](c, value)
}
func (FfiConverterHealthCheckStatus) Read(reader io.Reader) HealthCheckStatus {
	id := readInt32(reader)
	return HealthCheckStatus(id)
}

func (FfiConverterHealthCheckStatus) Write(writer io.Writer, value HealthCheckStatus) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerHealthCheckStatus struct{}

func (_ FfiDestroyerHealthCheckStatus) Destroy(value HealthCheckStatus) {
}

type InputType interface {
	Destroy()
}
type InputTypeBitcoinAddress struct {
	Address BitcoinAddressData
}

func (e InputTypeBitcoinAddress) Destroy() {
	FfiDestroyerBitcoinAddressData{}.Destroy(e.Address)
}

type InputTypeBolt11 struct {
	Invoice LnInvoice
}

func (e InputTypeBolt11) Destroy() {
	FfiDestroyerLnInvoice{}.Destroy(e.Invoice)
}

type InputTypeNodeId struct {
	NodeId string
}

func (e InputTypeNodeId) Destroy() {
	FfiDestroyerString{}.Destroy(e.NodeId)
}

type InputTypeUrl struct {
	Url string
}

func (e InputTypeUrl) Destroy() {
	FfiDestroyerString{}.Destroy(e.Url)
}

type InputTypeLnUrlPay struct {
	Data          LnUrlPayRequestData
	Bip353Address *string
}

func (e InputTypeLnUrlPay) Destroy() {
	FfiDestroyerLnUrlPayRequestData{}.Destroy(e.Data)
	FfiDestroyerOptionalString{}.Destroy(e.Bip353Address)
}

type InputTypeLnUrlWithdraw struct {
	Data LnUrlWithdrawRequestData
}

func (e InputTypeLnUrlWithdraw) Destroy() {
	FfiDestroyerLnUrlWithdrawRequestData{}.Destroy(e.Data)
}

type InputTypeLnUrlAuth struct {
	Data LnUrlAuthRequestData
}

func (e InputTypeLnUrlAuth) Destroy() {
	FfiDestroyerLnUrlAuthRequestData{}.Destroy(e.Data)
}

type InputTypeLnUrlError struct {
	Data LnUrlErrorData
}

func (e InputTypeLnUrlError) Destroy() {
	FfiDestroyerLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterInputType struct{}

var FfiConverterInputTypeINSTANCE = FfiConverterInputType{}

func (c FfiConverterInputType) Lift(rb RustBufferI) InputType {
	return LiftFromRustBuffer[InputType](c, rb)
}

func (c FfiConverterInputType) Lower(value InputType) C.RustBuffer {
	return LowerIntoRustBuffer[InputType](c, value)
}
func (FfiConverterInputType) Read(reader io.Reader) InputType {
	id := readInt32(reader)
	switch id {
	case 1:
		return InputTypeBitcoinAddress{
			FfiConverterBitcoinAddressDataINSTANCE.Read(reader),
		}
	case 2:
		return InputTypeBolt11{
			FfiConverterLnInvoiceINSTANCE.Read(reader),
		}
	case 3:
		return InputTypeNodeId{
			FfiConverterStringINSTANCE.Read(reader),
		}
	case 4:
		return InputTypeUrl{
			FfiConverterStringINSTANCE.Read(reader),
		}
	case 5:
		return InputTypeLnUrlPay{
			FfiConverterLnUrlPayRequestDataINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
		}
	case 6:
		return InputTypeLnUrlWithdraw{
			FfiConverterLnUrlWithdrawRequestDataINSTANCE.Read(reader),
		}
	case 7:
		return InputTypeLnUrlAuth{
			FfiConverterLnUrlAuthRequestDataINSTANCE.Read(reader),
		}
	case 8:
		return InputTypeLnUrlError{
			FfiConverterLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterInputType.Read()", id))
	}
}

func (FfiConverterInputType) Write(writer io.Writer, value InputType) {
	switch variant_value := value.(type) {
	case InputTypeBitcoinAddress:
		writeInt32(writer, 1)
		FfiConverterBitcoinAddressDataINSTANCE.Write(writer, variant_value.Address)
	case InputTypeBolt11:
		writeInt32(writer, 2)
		FfiConverterLnInvoiceINSTANCE.Write(writer, variant_value.Invoice)
	case InputTypeNodeId:
		writeInt32(writer, 3)
		FfiConverterStringINSTANCE.Write(writer, variant_value.NodeId)
	case InputTypeUrl:
		writeInt32(writer, 4)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Url)
	case InputTypeLnUrlPay:
		writeInt32(writer, 5)
		FfiConverterLnUrlPayRequestDataINSTANCE.Write(writer, variant_value.Data)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Bip353Address)
	case InputTypeLnUrlWithdraw:
		writeInt32(writer, 6)
		FfiConverterLnUrlWithdrawRequestDataINSTANCE.Write(writer, variant_value.Data)
	case InputTypeLnUrlAuth:
		writeInt32(writer, 7)
		FfiConverterLnUrlAuthRequestDataINSTANCE.Write(writer, variant_value.Data)
	case InputTypeLnUrlError:
		writeInt32(writer, 8)
		FfiConverterLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterInputType.Write", value))
	}
}

type FfiDestroyerInputType struct{}

func (_ FfiDestroyerInputType) Destroy(value InputType) {
	value.Destroy()
}

type LevelFilter uint

const (
	LevelFilterOff   LevelFilter = 1
	LevelFilterError LevelFilter = 2
	LevelFilterWarn  LevelFilter = 3
	LevelFilterInfo  LevelFilter = 4
	LevelFilterDebug LevelFilter = 5
	LevelFilterTrace LevelFilter = 6
)

type FfiConverterLevelFilter struct{}

var FfiConverterLevelFilterINSTANCE = FfiConverterLevelFilter{}

func (c FfiConverterLevelFilter) Lift(rb RustBufferI) LevelFilter {
	return LiftFromRustBuffer[LevelFilter](c, rb)
}

func (c FfiConverterLevelFilter) Lower(value LevelFilter) C.RustBuffer {
	return LowerIntoRustBuffer[LevelFilter](c, value)
}
func (FfiConverterLevelFilter) Read(reader io.Reader) LevelFilter {
	id := readInt32(reader)
	return LevelFilter(id)
}

func (FfiConverterLevelFilter) Write(writer io.Writer, value LevelFilter) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerLevelFilter struct{}

func (_ FfiDestroyerLevelFilter) Destroy(value LevelFilter) {
}

type LnUrlAuthError struct {
	err error
}

// Convience method to turn *LnUrlAuthError into error
// Avoiding treating nil pointer as non nil error interface
func (err *LnUrlAuthError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err LnUrlAuthError) Error() string {
	return fmt.Sprintf("LnUrlAuthError: %s", err.err.Error())
}

func (err LnUrlAuthError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrLnUrlAuthErrorGeneric = fmt.Errorf("LnUrlAuthErrorGeneric")
var ErrLnUrlAuthErrorInvalidUri = fmt.Errorf("LnUrlAuthErrorInvalidUri")
var ErrLnUrlAuthErrorServiceConnectivity = fmt.Errorf("LnUrlAuthErrorServiceConnectivity")

// Variant structs
type LnUrlAuthErrorGeneric struct {
	message string
}

func NewLnUrlAuthErrorGeneric() *LnUrlAuthError {
	return &LnUrlAuthError{err: &LnUrlAuthErrorGeneric{}}
}

func (e LnUrlAuthErrorGeneric) destroy() {
}

func (err LnUrlAuthErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self LnUrlAuthErrorGeneric) Is(target error) bool {
	return target == ErrLnUrlAuthErrorGeneric
}

type LnUrlAuthErrorInvalidUri struct {
	message string
}

func NewLnUrlAuthErrorInvalidUri() *LnUrlAuthError {
	return &LnUrlAuthError{err: &LnUrlAuthErrorInvalidUri{}}
}

func (e LnUrlAuthErrorInvalidUri) destroy() {
}

func (err LnUrlAuthErrorInvalidUri) Error() string {
	return fmt.Sprintf("InvalidUri: %s", err.message)
}

func (self LnUrlAuthErrorInvalidUri) Is(target error) bool {
	return target == ErrLnUrlAuthErrorInvalidUri
}

type LnUrlAuthErrorServiceConnectivity struct {
	message string
}

func NewLnUrlAuthErrorServiceConnectivity() *LnUrlAuthError {
	return &LnUrlAuthError{err: &LnUrlAuthErrorServiceConnectivity{}}
}

func (e LnUrlAuthErrorServiceConnectivity) destroy() {
}

func (err LnUrlAuthErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self LnUrlAuthErrorServiceConnectivity) Is(target error) bool {
	return target == ErrLnUrlAuthErrorServiceConnectivity
}

type FfiConverterLnUrlAuthError struct{}

var FfiConverterLnUrlAuthErrorINSTANCE = FfiConverterLnUrlAuthError{}

func (c FfiConverterLnUrlAuthError) Lift(eb RustBufferI) *LnUrlAuthError {
	return LiftFromRustBuffer[*LnUrlAuthError](c, eb)
}

func (c FfiConverterLnUrlAuthError) Lower(value *LnUrlAuthError) C.RustBuffer {
	return LowerIntoRustBuffer[*LnUrlAuthError](c, value)
}

func (c FfiConverterLnUrlAuthError) Read(reader io.Reader) *LnUrlAuthError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &LnUrlAuthError{&LnUrlAuthErrorGeneric{message}}
	case 2:
		return &LnUrlAuthError{&LnUrlAuthErrorInvalidUri{message}}
	case 3:
		return &LnUrlAuthError{&LnUrlAuthErrorServiceConnectivity{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterLnUrlAuthError.Read()", errorID))
	}

}

func (c FfiConverterLnUrlAuthError) Write(writer io.Writer, value *LnUrlAuthError) {
	switch variantValue := value.err.(type) {
	case *LnUrlAuthErrorGeneric:
		writeInt32(writer, 1)
	case *LnUrlAuthErrorInvalidUri:
		writeInt32(writer, 2)
	case *LnUrlAuthErrorServiceConnectivity:
		writeInt32(writer, 3)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterLnUrlAuthError.Write", value))
	}
}

type FfiDestroyerLnUrlAuthError struct{}

func (_ FfiDestroyerLnUrlAuthError) Destroy(value *LnUrlAuthError) {
	switch variantValue := value.err.(type) {
	case LnUrlAuthErrorGeneric:
		variantValue.destroy()
	case LnUrlAuthErrorInvalidUri:
		variantValue.destroy()
	case LnUrlAuthErrorServiceConnectivity:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerLnUrlAuthError.Destroy", value))
	}
}

type LnUrlCallbackStatus interface {
	Destroy()
}
type LnUrlCallbackStatusOk struct {
}

func (e LnUrlCallbackStatusOk) Destroy() {
}

type LnUrlCallbackStatusErrorStatus struct {
	Data LnUrlErrorData
}

func (e LnUrlCallbackStatusErrorStatus) Destroy() {
	FfiDestroyerLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterLnUrlCallbackStatus struct{}

var FfiConverterLnUrlCallbackStatusINSTANCE = FfiConverterLnUrlCallbackStatus{}

func (c FfiConverterLnUrlCallbackStatus) Lift(rb RustBufferI) LnUrlCallbackStatus {
	return LiftFromRustBuffer[LnUrlCallbackStatus](c, rb)
}

func (c FfiConverterLnUrlCallbackStatus) Lower(value LnUrlCallbackStatus) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlCallbackStatus](c, value)
}
func (FfiConverterLnUrlCallbackStatus) Read(reader io.Reader) LnUrlCallbackStatus {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlCallbackStatusOk{}
	case 2:
		return LnUrlCallbackStatusErrorStatus{
			FfiConverterLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterLnUrlCallbackStatus.Read()", id))
	}
}

func (FfiConverterLnUrlCallbackStatus) Write(writer io.Writer, value LnUrlCallbackStatus) {
	switch variant_value := value.(type) {
	case LnUrlCallbackStatusOk:
		writeInt32(writer, 1)
	case LnUrlCallbackStatusErrorStatus:
		writeInt32(writer, 2)
		FfiConverterLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterLnUrlCallbackStatus.Write", value))
	}
}

type FfiDestroyerLnUrlCallbackStatus struct{}

func (_ FfiDestroyerLnUrlCallbackStatus) Destroy(value LnUrlCallbackStatus) {
	value.Destroy()
}

type LnUrlPayError struct {
	err error
}

// Convience method to turn *LnUrlPayError into error
// Avoiding treating nil pointer as non nil error interface
func (err *LnUrlPayError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err LnUrlPayError) Error() string {
	return fmt.Sprintf("LnUrlPayError: %s", err.err.Error())
}

func (err LnUrlPayError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrLnUrlPayErrorAlreadyPaid = fmt.Errorf("LnUrlPayErrorAlreadyPaid")
var ErrLnUrlPayErrorGeneric = fmt.Errorf("LnUrlPayErrorGeneric")
var ErrLnUrlPayErrorInvalidAmount = fmt.Errorf("LnUrlPayErrorInvalidAmount")
var ErrLnUrlPayErrorInvalidInvoice = fmt.Errorf("LnUrlPayErrorInvalidInvoice")
var ErrLnUrlPayErrorInvalidNetwork = fmt.Errorf("LnUrlPayErrorInvalidNetwork")
var ErrLnUrlPayErrorInvalidUri = fmt.Errorf("LnUrlPayErrorInvalidUri")
var ErrLnUrlPayErrorInvoiceExpired = fmt.Errorf("LnUrlPayErrorInvoiceExpired")
var ErrLnUrlPayErrorPaymentFailed = fmt.Errorf("LnUrlPayErrorPaymentFailed")
var ErrLnUrlPayErrorPaymentTimeout = fmt.Errorf("LnUrlPayErrorPaymentTimeout")
var ErrLnUrlPayErrorRouteNotFound = fmt.Errorf("LnUrlPayErrorRouteNotFound")
var ErrLnUrlPayErrorRouteTooExpensive = fmt.Errorf("LnUrlPayErrorRouteTooExpensive")
var ErrLnUrlPayErrorServiceConnectivity = fmt.Errorf("LnUrlPayErrorServiceConnectivity")
var ErrLnUrlPayErrorInsufficientBalance = fmt.Errorf("LnUrlPayErrorInsufficientBalance")

// Variant structs
type LnUrlPayErrorAlreadyPaid struct {
	message string
}

func NewLnUrlPayErrorAlreadyPaid() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorAlreadyPaid{}}
}

func (e LnUrlPayErrorAlreadyPaid) destroy() {
}

func (err LnUrlPayErrorAlreadyPaid) Error() string {
	return fmt.Sprintf("AlreadyPaid: %s", err.message)
}

func (self LnUrlPayErrorAlreadyPaid) Is(target error) bool {
	return target == ErrLnUrlPayErrorAlreadyPaid
}

type LnUrlPayErrorGeneric struct {
	message string
}

func NewLnUrlPayErrorGeneric() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorGeneric{}}
}

func (e LnUrlPayErrorGeneric) destroy() {
}

func (err LnUrlPayErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self LnUrlPayErrorGeneric) Is(target error) bool {
	return target == ErrLnUrlPayErrorGeneric
}

type LnUrlPayErrorInvalidAmount struct {
	message string
}

func NewLnUrlPayErrorInvalidAmount() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInvalidAmount{}}
}

func (e LnUrlPayErrorInvalidAmount) destroy() {
}

func (err LnUrlPayErrorInvalidAmount) Error() string {
	return fmt.Sprintf("InvalidAmount: %s", err.message)
}

func (self LnUrlPayErrorInvalidAmount) Is(target error) bool {
	return target == ErrLnUrlPayErrorInvalidAmount
}

type LnUrlPayErrorInvalidInvoice struct {
	message string
}

func NewLnUrlPayErrorInvalidInvoice() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInvalidInvoice{}}
}

func (e LnUrlPayErrorInvalidInvoice) destroy() {
}

func (err LnUrlPayErrorInvalidInvoice) Error() string {
	return fmt.Sprintf("InvalidInvoice: %s", err.message)
}

func (self LnUrlPayErrorInvalidInvoice) Is(target error) bool {
	return target == ErrLnUrlPayErrorInvalidInvoice
}

type LnUrlPayErrorInvalidNetwork struct {
	message string
}

func NewLnUrlPayErrorInvalidNetwork() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInvalidNetwork{}}
}

func (e LnUrlPayErrorInvalidNetwork) destroy() {
}

func (err LnUrlPayErrorInvalidNetwork) Error() string {
	return fmt.Sprintf("InvalidNetwork: %s", err.message)
}

func (self LnUrlPayErrorInvalidNetwork) Is(target error) bool {
	return target == ErrLnUrlPayErrorInvalidNetwork
}

type LnUrlPayErrorInvalidUri struct {
	message string
}

func NewLnUrlPayErrorInvalidUri() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInvalidUri{}}
}

func (e LnUrlPayErrorInvalidUri) destroy() {
}

func (err LnUrlPayErrorInvalidUri) Error() string {
	return fmt.Sprintf("InvalidUri: %s", err.message)
}

func (self LnUrlPayErrorInvalidUri) Is(target error) bool {
	return target == ErrLnUrlPayErrorInvalidUri
}

type LnUrlPayErrorInvoiceExpired struct {
	message string
}

func NewLnUrlPayErrorInvoiceExpired() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInvoiceExpired{}}
}

func (e LnUrlPayErrorInvoiceExpired) destroy() {
}

func (err LnUrlPayErrorInvoiceExpired) Error() string {
	return fmt.Sprintf("InvoiceExpired: %s", err.message)
}

func (self LnUrlPayErrorInvoiceExpired) Is(target error) bool {
	return target == ErrLnUrlPayErrorInvoiceExpired
}

type LnUrlPayErrorPaymentFailed struct {
	message string
}

func NewLnUrlPayErrorPaymentFailed() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorPaymentFailed{}}
}

func (e LnUrlPayErrorPaymentFailed) destroy() {
}

func (err LnUrlPayErrorPaymentFailed) Error() string {
	return fmt.Sprintf("PaymentFailed: %s", err.message)
}

func (self LnUrlPayErrorPaymentFailed) Is(target error) bool {
	return target == ErrLnUrlPayErrorPaymentFailed
}

type LnUrlPayErrorPaymentTimeout struct {
	message string
}

func NewLnUrlPayErrorPaymentTimeout() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorPaymentTimeout{}}
}

func (e LnUrlPayErrorPaymentTimeout) destroy() {
}

func (err LnUrlPayErrorPaymentTimeout) Error() string {
	return fmt.Sprintf("PaymentTimeout: %s", err.message)
}

func (self LnUrlPayErrorPaymentTimeout) Is(target error) bool {
	return target == ErrLnUrlPayErrorPaymentTimeout
}

type LnUrlPayErrorRouteNotFound struct {
	message string
}

func NewLnUrlPayErrorRouteNotFound() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorRouteNotFound{}}
}

func (e LnUrlPayErrorRouteNotFound) destroy() {
}

func (err LnUrlPayErrorRouteNotFound) Error() string {
	return fmt.Sprintf("RouteNotFound: %s", err.message)
}

func (self LnUrlPayErrorRouteNotFound) Is(target error) bool {
	return target == ErrLnUrlPayErrorRouteNotFound
}

type LnUrlPayErrorRouteTooExpensive struct {
	message string
}

func NewLnUrlPayErrorRouteTooExpensive() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorRouteTooExpensive{}}
}

func (e LnUrlPayErrorRouteTooExpensive) destroy() {
}

func (err LnUrlPayErrorRouteTooExpensive) Error() string {
	return fmt.Sprintf("RouteTooExpensive: %s", err.message)
}

func (self LnUrlPayErrorRouteTooExpensive) Is(target error) bool {
	return target == ErrLnUrlPayErrorRouteTooExpensive
}

type LnUrlPayErrorServiceConnectivity struct {
	message string
}

func NewLnUrlPayErrorServiceConnectivity() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorServiceConnectivity{}}
}

func (e LnUrlPayErrorServiceConnectivity) destroy() {
}

func (err LnUrlPayErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self LnUrlPayErrorServiceConnectivity) Is(target error) bool {
	return target == ErrLnUrlPayErrorServiceConnectivity
}

type LnUrlPayErrorInsufficientBalance struct {
	message string
}

func NewLnUrlPayErrorInsufficientBalance() *LnUrlPayError {
	return &LnUrlPayError{err: &LnUrlPayErrorInsufficientBalance{}}
}

func (e LnUrlPayErrorInsufficientBalance) destroy() {
}

func (err LnUrlPayErrorInsufficientBalance) Error() string {
	return fmt.Sprintf("InsufficientBalance: %s", err.message)
}

func (self LnUrlPayErrorInsufficientBalance) Is(target error) bool {
	return target == ErrLnUrlPayErrorInsufficientBalance
}

type FfiConverterLnUrlPayError struct{}

var FfiConverterLnUrlPayErrorINSTANCE = FfiConverterLnUrlPayError{}

func (c FfiConverterLnUrlPayError) Lift(eb RustBufferI) *LnUrlPayError {
	return LiftFromRustBuffer[*LnUrlPayError](c, eb)
}

func (c FfiConverterLnUrlPayError) Lower(value *LnUrlPayError) C.RustBuffer {
	return LowerIntoRustBuffer[*LnUrlPayError](c, value)
}

func (c FfiConverterLnUrlPayError) Read(reader io.Reader) *LnUrlPayError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &LnUrlPayError{&LnUrlPayErrorAlreadyPaid{message}}
	case 2:
		return &LnUrlPayError{&LnUrlPayErrorGeneric{message}}
	case 3:
		return &LnUrlPayError{&LnUrlPayErrorInvalidAmount{message}}
	case 4:
		return &LnUrlPayError{&LnUrlPayErrorInvalidInvoice{message}}
	case 5:
		return &LnUrlPayError{&LnUrlPayErrorInvalidNetwork{message}}
	case 6:
		return &LnUrlPayError{&LnUrlPayErrorInvalidUri{message}}
	case 7:
		return &LnUrlPayError{&LnUrlPayErrorInvoiceExpired{message}}
	case 8:
		return &LnUrlPayError{&LnUrlPayErrorPaymentFailed{message}}
	case 9:
		return &LnUrlPayError{&LnUrlPayErrorPaymentTimeout{message}}
	case 10:
		return &LnUrlPayError{&LnUrlPayErrorRouteNotFound{message}}
	case 11:
		return &LnUrlPayError{&LnUrlPayErrorRouteTooExpensive{message}}
	case 12:
		return &LnUrlPayError{&LnUrlPayErrorServiceConnectivity{message}}
	case 13:
		return &LnUrlPayError{&LnUrlPayErrorInsufficientBalance{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterLnUrlPayError.Read()", errorID))
	}

}

func (c FfiConverterLnUrlPayError) Write(writer io.Writer, value *LnUrlPayError) {
	switch variantValue := value.err.(type) {
	case *LnUrlPayErrorAlreadyPaid:
		writeInt32(writer, 1)
	case *LnUrlPayErrorGeneric:
		writeInt32(writer, 2)
	case *LnUrlPayErrorInvalidAmount:
		writeInt32(writer, 3)
	case *LnUrlPayErrorInvalidInvoice:
		writeInt32(writer, 4)
	case *LnUrlPayErrorInvalidNetwork:
		writeInt32(writer, 5)
	case *LnUrlPayErrorInvalidUri:
		writeInt32(writer, 6)
	case *LnUrlPayErrorInvoiceExpired:
		writeInt32(writer, 7)
	case *LnUrlPayErrorPaymentFailed:
		writeInt32(writer, 8)
	case *LnUrlPayErrorPaymentTimeout:
		writeInt32(writer, 9)
	case *LnUrlPayErrorRouteNotFound:
		writeInt32(writer, 10)
	case *LnUrlPayErrorRouteTooExpensive:
		writeInt32(writer, 11)
	case *LnUrlPayErrorServiceConnectivity:
		writeInt32(writer, 12)
	case *LnUrlPayErrorInsufficientBalance:
		writeInt32(writer, 13)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterLnUrlPayError.Write", value))
	}
}

type FfiDestroyerLnUrlPayError struct{}

func (_ FfiDestroyerLnUrlPayError) Destroy(value *LnUrlPayError) {
	switch variantValue := value.err.(type) {
	case LnUrlPayErrorAlreadyPaid:
		variantValue.destroy()
	case LnUrlPayErrorGeneric:
		variantValue.destroy()
	case LnUrlPayErrorInvalidAmount:
		variantValue.destroy()
	case LnUrlPayErrorInvalidInvoice:
		variantValue.destroy()
	case LnUrlPayErrorInvalidNetwork:
		variantValue.destroy()
	case LnUrlPayErrorInvalidUri:
		variantValue.destroy()
	case LnUrlPayErrorInvoiceExpired:
		variantValue.destroy()
	case LnUrlPayErrorPaymentFailed:
		variantValue.destroy()
	case LnUrlPayErrorPaymentTimeout:
		variantValue.destroy()
	case LnUrlPayErrorRouteNotFound:
		variantValue.destroy()
	case LnUrlPayErrorRouteTooExpensive:
		variantValue.destroy()
	case LnUrlPayErrorServiceConnectivity:
		variantValue.destroy()
	case LnUrlPayErrorInsufficientBalance:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerLnUrlPayError.Destroy", value))
	}
}

type LnUrlPayResult interface {
	Destroy()
}
type LnUrlPayResultEndpointSuccess struct {
	Data LnUrlPaySuccessData
}

func (e LnUrlPayResultEndpointSuccess) Destroy() {
	FfiDestroyerLnUrlPaySuccessData{}.Destroy(e.Data)
}

type LnUrlPayResultEndpointError struct {
	Data LnUrlErrorData
}

func (e LnUrlPayResultEndpointError) Destroy() {
	FfiDestroyerLnUrlErrorData{}.Destroy(e.Data)
}

type LnUrlPayResultPayError struct {
	Data LnUrlPayErrorData
}

func (e LnUrlPayResultPayError) Destroy() {
	FfiDestroyerLnUrlPayErrorData{}.Destroy(e.Data)
}

type FfiConverterLnUrlPayResult struct{}

var FfiConverterLnUrlPayResultINSTANCE = FfiConverterLnUrlPayResult{}

func (c FfiConverterLnUrlPayResult) Lift(rb RustBufferI) LnUrlPayResult {
	return LiftFromRustBuffer[LnUrlPayResult](c, rb)
}

func (c FfiConverterLnUrlPayResult) Lower(value LnUrlPayResult) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayResult](c, value)
}
func (FfiConverterLnUrlPayResult) Read(reader io.Reader) LnUrlPayResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlPayResultEndpointSuccess{
			FfiConverterLnUrlPaySuccessDataINSTANCE.Read(reader),
		}
	case 2:
		return LnUrlPayResultEndpointError{
			FfiConverterLnUrlErrorDataINSTANCE.Read(reader),
		}
	case 3:
		return LnUrlPayResultPayError{
			FfiConverterLnUrlPayErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterLnUrlPayResult.Read()", id))
	}
}

func (FfiConverterLnUrlPayResult) Write(writer io.Writer, value LnUrlPayResult) {
	switch variant_value := value.(type) {
	case LnUrlPayResultEndpointSuccess:
		writeInt32(writer, 1)
		FfiConverterLnUrlPaySuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlPayResultEndpointError:
		writeInt32(writer, 2)
		FfiConverterLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlPayResultPayError:
		writeInt32(writer, 3)
		FfiConverterLnUrlPayErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterLnUrlPayResult.Write", value))
	}
}

type FfiDestroyerLnUrlPayResult struct{}

func (_ FfiDestroyerLnUrlPayResult) Destroy(value LnUrlPayResult) {
	value.Destroy()
}

type LnUrlWithdrawError struct {
	err error
}

// Convience method to turn *LnUrlWithdrawError into error
// Avoiding treating nil pointer as non nil error interface
func (err *LnUrlWithdrawError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err LnUrlWithdrawError) Error() string {
	return fmt.Sprintf("LnUrlWithdrawError: %s", err.err.Error())
}

func (err LnUrlWithdrawError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrLnUrlWithdrawErrorGeneric = fmt.Errorf("LnUrlWithdrawErrorGeneric")
var ErrLnUrlWithdrawErrorInvalidAmount = fmt.Errorf("LnUrlWithdrawErrorInvalidAmount")
var ErrLnUrlWithdrawErrorInvalidInvoice = fmt.Errorf("LnUrlWithdrawErrorInvalidInvoice")
var ErrLnUrlWithdrawErrorInvalidUri = fmt.Errorf("LnUrlWithdrawErrorInvalidUri")
var ErrLnUrlWithdrawErrorServiceConnectivity = fmt.Errorf("LnUrlWithdrawErrorServiceConnectivity")
var ErrLnUrlWithdrawErrorInvoiceNoRoutingHints = fmt.Errorf("LnUrlWithdrawErrorInvoiceNoRoutingHints")

// Variant structs
type LnUrlWithdrawErrorGeneric struct {
	message string
}

func NewLnUrlWithdrawErrorGeneric() *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorGeneric{}}
}

func (e LnUrlWithdrawErrorGeneric) destroy() {
}

func (err LnUrlWithdrawErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self LnUrlWithdrawErrorGeneric) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorGeneric
}

type LnUrlWithdrawErrorInvalidAmount struct {
	message string
}

func NewLnUrlWithdrawErrorInvalidAmount() *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorInvalidAmount{}}
}

func (e LnUrlWithdrawErrorInvalidAmount) destroy() {
}

func (err LnUrlWithdrawErrorInvalidAmount) Error() string {
	return fmt.Sprintf("InvalidAmount: %s", err.message)
}

func (self LnUrlWithdrawErrorInvalidAmount) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorInvalidAmount
}

type LnUrlWithdrawErrorInvalidInvoice struct {
	message string
}

func NewLnUrlWithdrawErrorInvalidInvoice() *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorInvalidInvoice{}}
}

func (e LnUrlWithdrawErrorInvalidInvoice) destroy() {
}

func (err LnUrlWithdrawErrorInvalidInvoice) Error() string {
	return fmt.Sprintf("InvalidInvoice: %s", err.message)
}

func (self LnUrlWithdrawErrorInvalidInvoice) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorInvalidInvoice
}

type LnUrlWithdrawErrorInvalidUri struct {
	message string
}

func NewLnUrlWithdrawErrorInvalidUri() *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorInvalidUri{}}
}

func (e LnUrlWithdrawErrorInvalidUri) destroy() {
}

func (err LnUrlWithdrawErrorInvalidUri) Error() string {
	return fmt.Sprintf("InvalidUri: %s", err.message)
}

func (self LnUrlWithdrawErrorInvalidUri) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorInvalidUri
}

type LnUrlWithdrawErrorServiceConnectivity struct {
	message string
}

func NewLnUrlWithdrawErrorServiceConnectivity() *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorServiceConnectivity{}}
}

func (e LnUrlWithdrawErrorServiceConnectivity) destroy() {
}

func (err LnUrlWithdrawErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self LnUrlWithdrawErrorServiceConnectivity) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorServiceConnectivity
}

type LnUrlWithdrawErrorInvoiceNoRoutingHints struct {
	message string
}

func NewLnUrlWithdrawErrorInvoiceNoRoutingHints() *LnUrlWithdrawError {
	return &LnUrlWithdrawError{err: &LnUrlWithdrawErrorInvoiceNoRoutingHints{}}
}

func (e LnUrlWithdrawErrorInvoiceNoRoutingHints) destroy() {
}

func (err LnUrlWithdrawErrorInvoiceNoRoutingHints) Error() string {
	return fmt.Sprintf("InvoiceNoRoutingHints: %s", err.message)
}

func (self LnUrlWithdrawErrorInvoiceNoRoutingHints) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorInvoiceNoRoutingHints
}

type FfiConverterLnUrlWithdrawError struct{}

var FfiConverterLnUrlWithdrawErrorINSTANCE = FfiConverterLnUrlWithdrawError{}

func (c FfiConverterLnUrlWithdrawError) Lift(eb RustBufferI) *LnUrlWithdrawError {
	return LiftFromRustBuffer[*LnUrlWithdrawError](c, eb)
}

func (c FfiConverterLnUrlWithdrawError) Lower(value *LnUrlWithdrawError) C.RustBuffer {
	return LowerIntoRustBuffer[*LnUrlWithdrawError](c, value)
}

func (c FfiConverterLnUrlWithdrawError) Read(reader io.Reader) *LnUrlWithdrawError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorGeneric{message}}
	case 2:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorInvalidAmount{message}}
	case 3:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorInvalidInvoice{message}}
	case 4:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorInvalidUri{message}}
	case 5:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorServiceConnectivity{message}}
	case 6:
		return &LnUrlWithdrawError{&LnUrlWithdrawErrorInvoiceNoRoutingHints{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterLnUrlWithdrawError.Read()", errorID))
	}

}

func (c FfiConverterLnUrlWithdrawError) Write(writer io.Writer, value *LnUrlWithdrawError) {
	switch variantValue := value.err.(type) {
	case *LnUrlWithdrawErrorGeneric:
		writeInt32(writer, 1)
	case *LnUrlWithdrawErrorInvalidAmount:
		writeInt32(writer, 2)
	case *LnUrlWithdrawErrorInvalidInvoice:
		writeInt32(writer, 3)
	case *LnUrlWithdrawErrorInvalidUri:
		writeInt32(writer, 4)
	case *LnUrlWithdrawErrorServiceConnectivity:
		writeInt32(writer, 5)
	case *LnUrlWithdrawErrorInvoiceNoRoutingHints:
		writeInt32(writer, 6)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterLnUrlWithdrawError.Write", value))
	}
}

type FfiDestroyerLnUrlWithdrawError struct{}

func (_ FfiDestroyerLnUrlWithdrawError) Destroy(value *LnUrlWithdrawError) {
	switch variantValue := value.err.(type) {
	case LnUrlWithdrawErrorGeneric:
		variantValue.destroy()
	case LnUrlWithdrawErrorInvalidAmount:
		variantValue.destroy()
	case LnUrlWithdrawErrorInvalidInvoice:
		variantValue.destroy()
	case LnUrlWithdrawErrorInvalidUri:
		variantValue.destroy()
	case LnUrlWithdrawErrorServiceConnectivity:
		variantValue.destroy()
	case LnUrlWithdrawErrorInvoiceNoRoutingHints:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerLnUrlWithdrawError.Destroy", value))
	}
}

type LnUrlWithdrawResult interface {
	Destroy()
}
type LnUrlWithdrawResultOk struct {
	Data LnUrlWithdrawSuccessData
}

func (e LnUrlWithdrawResultOk) Destroy() {
	FfiDestroyerLnUrlWithdrawSuccessData{}.Destroy(e.Data)
}

type LnUrlWithdrawResultTimeout struct {
	Data LnUrlWithdrawSuccessData
}

func (e LnUrlWithdrawResultTimeout) Destroy() {
	FfiDestroyerLnUrlWithdrawSuccessData{}.Destroy(e.Data)
}

type LnUrlWithdrawResultErrorStatus struct {
	Data LnUrlErrorData
}

func (e LnUrlWithdrawResultErrorStatus) Destroy() {
	FfiDestroyerLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterLnUrlWithdrawResult struct{}

var FfiConverterLnUrlWithdrawResultINSTANCE = FfiConverterLnUrlWithdrawResult{}

func (c FfiConverterLnUrlWithdrawResult) Lift(rb RustBufferI) LnUrlWithdrawResult {
	return LiftFromRustBuffer[LnUrlWithdrawResult](c, rb)
}

func (c FfiConverterLnUrlWithdrawResult) Lower(value LnUrlWithdrawResult) C.RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawResult](c, value)
}
func (FfiConverterLnUrlWithdrawResult) Read(reader io.Reader) LnUrlWithdrawResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlWithdrawResultOk{
			FfiConverterLnUrlWithdrawSuccessDataINSTANCE.Read(reader),
		}
	case 2:
		return LnUrlWithdrawResultTimeout{
			FfiConverterLnUrlWithdrawSuccessDataINSTANCE.Read(reader),
		}
	case 3:
		return LnUrlWithdrawResultErrorStatus{
			FfiConverterLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterLnUrlWithdrawResult.Read()", id))
	}
}

func (FfiConverterLnUrlWithdrawResult) Write(writer io.Writer, value LnUrlWithdrawResult) {
	switch variant_value := value.(type) {
	case LnUrlWithdrawResultOk:
		writeInt32(writer, 1)
		FfiConverterLnUrlWithdrawSuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlWithdrawResultTimeout:
		writeInt32(writer, 2)
		FfiConverterLnUrlWithdrawSuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlWithdrawResultErrorStatus:
		writeInt32(writer, 3)
		FfiConverterLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterLnUrlWithdrawResult.Write", value))
	}
}

type FfiDestroyerLnUrlWithdrawResult struct{}

func (_ FfiDestroyerLnUrlWithdrawResult) Destroy(value LnUrlWithdrawResult) {
	value.Destroy()
}

type Network uint

const (
	NetworkBitcoin Network = 1
	NetworkTestnet Network = 2
	NetworkSignet  Network = 3
	NetworkRegtest Network = 4
)

type FfiConverterNetwork struct{}

var FfiConverterNetworkINSTANCE = FfiConverterNetwork{}

func (c FfiConverterNetwork) Lift(rb RustBufferI) Network {
	return LiftFromRustBuffer[Network](c, rb)
}

func (c FfiConverterNetwork) Lower(value Network) C.RustBuffer {
	return LowerIntoRustBuffer[Network](c, value)
}
func (FfiConverterNetwork) Read(reader io.Reader) Network {
	id := readInt32(reader)
	return Network(id)
}

func (FfiConverterNetwork) Write(writer io.Writer, value Network) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerNetwork struct{}

func (_ FfiDestroyerNetwork) Destroy(value Network) {
}

type NodeConfig interface {
	Destroy()
}
type NodeConfigGreenlight struct {
	Config GreenlightNodeConfig
}

func (e NodeConfigGreenlight) Destroy() {
	FfiDestroyerGreenlightNodeConfig{}.Destroy(e.Config)
}

type FfiConverterNodeConfig struct{}

var FfiConverterNodeConfigINSTANCE = FfiConverterNodeConfig{}

func (c FfiConverterNodeConfig) Lift(rb RustBufferI) NodeConfig {
	return LiftFromRustBuffer[NodeConfig](c, rb)
}

func (c FfiConverterNodeConfig) Lower(value NodeConfig) C.RustBuffer {
	return LowerIntoRustBuffer[NodeConfig](c, value)
}
func (FfiConverterNodeConfig) Read(reader io.Reader) NodeConfig {
	id := readInt32(reader)
	switch id {
	case 1:
		return NodeConfigGreenlight{
			FfiConverterGreenlightNodeConfigINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterNodeConfig.Read()", id))
	}
}

func (FfiConverterNodeConfig) Write(writer io.Writer, value NodeConfig) {
	switch variant_value := value.(type) {
	case NodeConfigGreenlight:
		writeInt32(writer, 1)
		FfiConverterGreenlightNodeConfigINSTANCE.Write(writer, variant_value.Config)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterNodeConfig.Write", value))
	}
}

type FfiDestroyerNodeConfig struct{}

func (_ FfiDestroyerNodeConfig) Destroy(value NodeConfig) {
	value.Destroy()
}

type NodeCredentials interface {
	Destroy()
}
type NodeCredentialsGreenlight struct {
	Credentials GreenlightDeviceCredentials
}

func (e NodeCredentialsGreenlight) Destroy() {
	FfiDestroyerGreenlightDeviceCredentials{}.Destroy(e.Credentials)
}

type FfiConverterNodeCredentials struct{}

var FfiConverterNodeCredentialsINSTANCE = FfiConverterNodeCredentials{}

func (c FfiConverterNodeCredentials) Lift(rb RustBufferI) NodeCredentials {
	return LiftFromRustBuffer[NodeCredentials](c, rb)
}

func (c FfiConverterNodeCredentials) Lower(value NodeCredentials) C.RustBuffer {
	return LowerIntoRustBuffer[NodeCredentials](c, value)
}
func (FfiConverterNodeCredentials) Read(reader io.Reader) NodeCredentials {
	id := readInt32(reader)
	switch id {
	case 1:
		return NodeCredentialsGreenlight{
			FfiConverterGreenlightDeviceCredentialsINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterNodeCredentials.Read()", id))
	}
}

func (FfiConverterNodeCredentials) Write(writer io.Writer, value NodeCredentials) {
	switch variant_value := value.(type) {
	case NodeCredentialsGreenlight:
		writeInt32(writer, 1)
		FfiConverterGreenlightDeviceCredentialsINSTANCE.Write(writer, variant_value.Credentials)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterNodeCredentials.Write", value))
	}
}

type FfiDestroyerNodeCredentials struct{}

func (_ FfiDestroyerNodeCredentials) Destroy(value NodeCredentials) {
	value.Destroy()
}

type PaymentDetails interface {
	Destroy()
}
type PaymentDetailsLn struct {
	Data LnPaymentDetails
}

func (e PaymentDetailsLn) Destroy() {
	FfiDestroyerLnPaymentDetails{}.Destroy(e.Data)
}

type PaymentDetailsClosedChannel struct {
	Data ClosedChannelPaymentDetails
}

func (e PaymentDetailsClosedChannel) Destroy() {
	FfiDestroyerClosedChannelPaymentDetails{}.Destroy(e.Data)
}

type FfiConverterPaymentDetails struct{}

var FfiConverterPaymentDetailsINSTANCE = FfiConverterPaymentDetails{}

func (c FfiConverterPaymentDetails) Lift(rb RustBufferI) PaymentDetails {
	return LiftFromRustBuffer[PaymentDetails](c, rb)
}

func (c FfiConverterPaymentDetails) Lower(value PaymentDetails) C.RustBuffer {
	return LowerIntoRustBuffer[PaymentDetails](c, value)
}
func (FfiConverterPaymentDetails) Read(reader io.Reader) PaymentDetails {
	id := readInt32(reader)
	switch id {
	case 1:
		return PaymentDetailsLn{
			FfiConverterLnPaymentDetailsINSTANCE.Read(reader),
		}
	case 2:
		return PaymentDetailsClosedChannel{
			FfiConverterClosedChannelPaymentDetailsINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterPaymentDetails.Read()", id))
	}
}

func (FfiConverterPaymentDetails) Write(writer io.Writer, value PaymentDetails) {
	switch variant_value := value.(type) {
	case PaymentDetailsLn:
		writeInt32(writer, 1)
		FfiConverterLnPaymentDetailsINSTANCE.Write(writer, variant_value.Data)
	case PaymentDetailsClosedChannel:
		writeInt32(writer, 2)
		FfiConverterClosedChannelPaymentDetailsINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterPaymentDetails.Write", value))
	}
}

type FfiDestroyerPaymentDetails struct{}

func (_ FfiDestroyerPaymentDetails) Destroy(value PaymentDetails) {
	value.Destroy()
}

type PaymentStatus uint

const (
	PaymentStatusPending  PaymentStatus = 1
	PaymentStatusComplete PaymentStatus = 2
	PaymentStatusFailed   PaymentStatus = 3
)

type FfiConverterPaymentStatus struct{}

var FfiConverterPaymentStatusINSTANCE = FfiConverterPaymentStatus{}

func (c FfiConverterPaymentStatus) Lift(rb RustBufferI) PaymentStatus {
	return LiftFromRustBuffer[PaymentStatus](c, rb)
}

func (c FfiConverterPaymentStatus) Lower(value PaymentStatus) C.RustBuffer {
	return LowerIntoRustBuffer[PaymentStatus](c, value)
}
func (FfiConverterPaymentStatus) Read(reader io.Reader) PaymentStatus {
	id := readInt32(reader)
	return PaymentStatus(id)
}

func (FfiConverterPaymentStatus) Write(writer io.Writer, value PaymentStatus) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerPaymentStatus struct{}

func (_ FfiDestroyerPaymentStatus) Destroy(value PaymentStatus) {
}

type PaymentType uint

const (
	PaymentTypeSent          PaymentType = 1
	PaymentTypeReceived      PaymentType = 2
	PaymentTypeClosedChannel PaymentType = 3
)

type FfiConverterPaymentType struct{}

var FfiConverterPaymentTypeINSTANCE = FfiConverterPaymentType{}

func (c FfiConverterPaymentType) Lift(rb RustBufferI) PaymentType {
	return LiftFromRustBuffer[PaymentType](c, rb)
}

func (c FfiConverterPaymentType) Lower(value PaymentType) C.RustBuffer {
	return LowerIntoRustBuffer[PaymentType](c, value)
}
func (FfiConverterPaymentType) Read(reader io.Reader) PaymentType {
	id := readInt32(reader)
	return PaymentType(id)
}

func (FfiConverterPaymentType) Write(writer io.Writer, value PaymentType) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerPaymentType struct{}

func (_ FfiDestroyerPaymentType) Destroy(value PaymentType) {
}

type PaymentTypeFilter uint

const (
	PaymentTypeFilterSent          PaymentTypeFilter = 1
	PaymentTypeFilterReceived      PaymentTypeFilter = 2
	PaymentTypeFilterClosedChannel PaymentTypeFilter = 3
)

type FfiConverterPaymentTypeFilter struct{}

var FfiConverterPaymentTypeFilterINSTANCE = FfiConverterPaymentTypeFilter{}

func (c FfiConverterPaymentTypeFilter) Lift(rb RustBufferI) PaymentTypeFilter {
	return LiftFromRustBuffer[PaymentTypeFilter](c, rb)
}

func (c FfiConverterPaymentTypeFilter) Lower(value PaymentTypeFilter) C.RustBuffer {
	return LowerIntoRustBuffer[PaymentTypeFilter](c, value)
}
func (FfiConverterPaymentTypeFilter) Read(reader io.Reader) PaymentTypeFilter {
	id := readInt32(reader)
	return PaymentTypeFilter(id)
}

func (FfiConverterPaymentTypeFilter) Write(writer io.Writer, value PaymentTypeFilter) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerPaymentTypeFilter struct{}

func (_ FfiDestroyerPaymentTypeFilter) Destroy(value PaymentTypeFilter) {
}

type ReceiveOnchainError struct {
	err error
}

// Convience method to turn *ReceiveOnchainError into error
// Avoiding treating nil pointer as non nil error interface
func (err *ReceiveOnchainError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err ReceiveOnchainError) Error() string {
	return fmt.Sprintf("ReceiveOnchainError: %s", err.err.Error())
}

func (err ReceiveOnchainError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrReceiveOnchainErrorGeneric = fmt.Errorf("ReceiveOnchainErrorGeneric")
var ErrReceiveOnchainErrorServiceConnectivity = fmt.Errorf("ReceiveOnchainErrorServiceConnectivity")
var ErrReceiveOnchainErrorSwapInProgress = fmt.Errorf("ReceiveOnchainErrorSwapInProgress")

// Variant structs
type ReceiveOnchainErrorGeneric struct {
	message string
}

func NewReceiveOnchainErrorGeneric() *ReceiveOnchainError {
	return &ReceiveOnchainError{err: &ReceiveOnchainErrorGeneric{}}
}

func (e ReceiveOnchainErrorGeneric) destroy() {
}

func (err ReceiveOnchainErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self ReceiveOnchainErrorGeneric) Is(target error) bool {
	return target == ErrReceiveOnchainErrorGeneric
}

type ReceiveOnchainErrorServiceConnectivity struct {
	message string
}

func NewReceiveOnchainErrorServiceConnectivity() *ReceiveOnchainError {
	return &ReceiveOnchainError{err: &ReceiveOnchainErrorServiceConnectivity{}}
}

func (e ReceiveOnchainErrorServiceConnectivity) destroy() {
}

func (err ReceiveOnchainErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self ReceiveOnchainErrorServiceConnectivity) Is(target error) bool {
	return target == ErrReceiveOnchainErrorServiceConnectivity
}

type ReceiveOnchainErrorSwapInProgress struct {
	message string
}

func NewReceiveOnchainErrorSwapInProgress() *ReceiveOnchainError {
	return &ReceiveOnchainError{err: &ReceiveOnchainErrorSwapInProgress{}}
}

func (e ReceiveOnchainErrorSwapInProgress) destroy() {
}

func (err ReceiveOnchainErrorSwapInProgress) Error() string {
	return fmt.Sprintf("SwapInProgress: %s", err.message)
}

func (self ReceiveOnchainErrorSwapInProgress) Is(target error) bool {
	return target == ErrReceiveOnchainErrorSwapInProgress
}

type FfiConverterReceiveOnchainError struct{}

var FfiConverterReceiveOnchainErrorINSTANCE = FfiConverterReceiveOnchainError{}

func (c FfiConverterReceiveOnchainError) Lift(eb RustBufferI) *ReceiveOnchainError {
	return LiftFromRustBuffer[*ReceiveOnchainError](c, eb)
}

func (c FfiConverterReceiveOnchainError) Lower(value *ReceiveOnchainError) C.RustBuffer {
	return LowerIntoRustBuffer[*ReceiveOnchainError](c, value)
}

func (c FfiConverterReceiveOnchainError) Read(reader io.Reader) *ReceiveOnchainError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &ReceiveOnchainError{&ReceiveOnchainErrorGeneric{message}}
	case 2:
		return &ReceiveOnchainError{&ReceiveOnchainErrorServiceConnectivity{message}}
	case 3:
		return &ReceiveOnchainError{&ReceiveOnchainErrorSwapInProgress{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterReceiveOnchainError.Read()", errorID))
	}

}

func (c FfiConverterReceiveOnchainError) Write(writer io.Writer, value *ReceiveOnchainError) {
	switch variantValue := value.err.(type) {
	case *ReceiveOnchainErrorGeneric:
		writeInt32(writer, 1)
	case *ReceiveOnchainErrorServiceConnectivity:
		writeInt32(writer, 2)
	case *ReceiveOnchainErrorSwapInProgress:
		writeInt32(writer, 3)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterReceiveOnchainError.Write", value))
	}
}

type FfiDestroyerReceiveOnchainError struct{}

func (_ FfiDestroyerReceiveOnchainError) Destroy(value *ReceiveOnchainError) {
	switch variantValue := value.err.(type) {
	case ReceiveOnchainErrorGeneric:
		variantValue.destroy()
	case ReceiveOnchainErrorServiceConnectivity:
		variantValue.destroy()
	case ReceiveOnchainErrorSwapInProgress:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerReceiveOnchainError.Destroy", value))
	}
}

type ReceivePaymentError struct {
	err error
}

// Convience method to turn *ReceivePaymentError into error
// Avoiding treating nil pointer as non nil error interface
func (err *ReceivePaymentError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err ReceivePaymentError) Error() string {
	return fmt.Sprintf("ReceivePaymentError: %s", err.err.Error())
}

func (err ReceivePaymentError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrReceivePaymentErrorGeneric = fmt.Errorf("ReceivePaymentErrorGeneric")
var ErrReceivePaymentErrorInvalidAmount = fmt.Errorf("ReceivePaymentErrorInvalidAmount")
var ErrReceivePaymentErrorInvalidInvoice = fmt.Errorf("ReceivePaymentErrorInvalidInvoice")
var ErrReceivePaymentErrorInvoiceExpired = fmt.Errorf("ReceivePaymentErrorInvoiceExpired")
var ErrReceivePaymentErrorInvoiceNoDescription = fmt.Errorf("ReceivePaymentErrorInvoiceNoDescription")
var ErrReceivePaymentErrorInvoicePreimageAlreadyExists = fmt.Errorf("ReceivePaymentErrorInvoicePreimageAlreadyExists")
var ErrReceivePaymentErrorServiceConnectivity = fmt.Errorf("ReceivePaymentErrorServiceConnectivity")
var ErrReceivePaymentErrorInvoiceNoRoutingHints = fmt.Errorf("ReceivePaymentErrorInvoiceNoRoutingHints")

// Variant structs
type ReceivePaymentErrorGeneric struct {
	message string
}

func NewReceivePaymentErrorGeneric() *ReceivePaymentError {
	return &ReceivePaymentError{err: &ReceivePaymentErrorGeneric{}}
}

func (e ReceivePaymentErrorGeneric) destroy() {
}

func (err ReceivePaymentErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self ReceivePaymentErrorGeneric) Is(target error) bool {
	return target == ErrReceivePaymentErrorGeneric
}

type ReceivePaymentErrorInvalidAmount struct {
	message string
}

func NewReceivePaymentErrorInvalidAmount() *ReceivePaymentError {
	return &ReceivePaymentError{err: &ReceivePaymentErrorInvalidAmount{}}
}

func (e ReceivePaymentErrorInvalidAmount) destroy() {
}

func (err ReceivePaymentErrorInvalidAmount) Error() string {
	return fmt.Sprintf("InvalidAmount: %s", err.message)
}

func (self ReceivePaymentErrorInvalidAmount) Is(target error) bool {
	return target == ErrReceivePaymentErrorInvalidAmount
}

type ReceivePaymentErrorInvalidInvoice struct {
	message string
}

func NewReceivePaymentErrorInvalidInvoice() *ReceivePaymentError {
	return &ReceivePaymentError{err: &ReceivePaymentErrorInvalidInvoice{}}
}

func (e ReceivePaymentErrorInvalidInvoice) destroy() {
}

func (err ReceivePaymentErrorInvalidInvoice) Error() string {
	return fmt.Sprintf("InvalidInvoice: %s", err.message)
}

func (self ReceivePaymentErrorInvalidInvoice) Is(target error) bool {
	return target == ErrReceivePaymentErrorInvalidInvoice
}

type ReceivePaymentErrorInvoiceExpired struct {
	message string
}

func NewReceivePaymentErrorInvoiceExpired() *ReceivePaymentError {
	return &ReceivePaymentError{err: &ReceivePaymentErrorInvoiceExpired{}}
}

func (e ReceivePaymentErrorInvoiceExpired) destroy() {
}

func (err ReceivePaymentErrorInvoiceExpired) Error() string {
	return fmt.Sprintf("InvoiceExpired: %s", err.message)
}

func (self ReceivePaymentErrorInvoiceExpired) Is(target error) bool {
	return target == ErrReceivePaymentErrorInvoiceExpired
}

type ReceivePaymentErrorInvoiceNoDescription struct {
	message string
}

func NewReceivePaymentErrorInvoiceNoDescription() *ReceivePaymentError {
	return &ReceivePaymentError{err: &ReceivePaymentErrorInvoiceNoDescription{}}
}

func (e ReceivePaymentErrorInvoiceNoDescription) destroy() {
}

func (err ReceivePaymentErrorInvoiceNoDescription) Error() string {
	return fmt.Sprintf("InvoiceNoDescription: %s", err.message)
}

func (self ReceivePaymentErrorInvoiceNoDescription) Is(target error) bool {
	return target == ErrReceivePaymentErrorInvoiceNoDescription
}

type ReceivePaymentErrorInvoicePreimageAlreadyExists struct {
	message string
}

func NewReceivePaymentErrorInvoicePreimageAlreadyExists() *ReceivePaymentError {
	return &ReceivePaymentError{err: &ReceivePaymentErrorInvoicePreimageAlreadyExists{}}
}

func (e ReceivePaymentErrorInvoicePreimageAlreadyExists) destroy() {
}

func (err ReceivePaymentErrorInvoicePreimageAlreadyExists) Error() string {
	return fmt.Sprintf("InvoicePreimageAlreadyExists: %s", err.message)
}

func (self ReceivePaymentErrorInvoicePreimageAlreadyExists) Is(target error) bool {
	return target == ErrReceivePaymentErrorInvoicePreimageAlreadyExists
}

type ReceivePaymentErrorServiceConnectivity struct {
	message string
}

func NewReceivePaymentErrorServiceConnectivity() *ReceivePaymentError {
	return &ReceivePaymentError{err: &ReceivePaymentErrorServiceConnectivity{}}
}

func (e ReceivePaymentErrorServiceConnectivity) destroy() {
}

func (err ReceivePaymentErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self ReceivePaymentErrorServiceConnectivity) Is(target error) bool {
	return target == ErrReceivePaymentErrorServiceConnectivity
}

type ReceivePaymentErrorInvoiceNoRoutingHints struct {
	message string
}

func NewReceivePaymentErrorInvoiceNoRoutingHints() *ReceivePaymentError {
	return &ReceivePaymentError{err: &ReceivePaymentErrorInvoiceNoRoutingHints{}}
}

func (e ReceivePaymentErrorInvoiceNoRoutingHints) destroy() {
}

func (err ReceivePaymentErrorInvoiceNoRoutingHints) Error() string {
	return fmt.Sprintf("InvoiceNoRoutingHints: %s", err.message)
}

func (self ReceivePaymentErrorInvoiceNoRoutingHints) Is(target error) bool {
	return target == ErrReceivePaymentErrorInvoiceNoRoutingHints
}

type FfiConverterReceivePaymentError struct{}

var FfiConverterReceivePaymentErrorINSTANCE = FfiConverterReceivePaymentError{}

func (c FfiConverterReceivePaymentError) Lift(eb RustBufferI) *ReceivePaymentError {
	return LiftFromRustBuffer[*ReceivePaymentError](c, eb)
}

func (c FfiConverterReceivePaymentError) Lower(value *ReceivePaymentError) C.RustBuffer {
	return LowerIntoRustBuffer[*ReceivePaymentError](c, value)
}

func (c FfiConverterReceivePaymentError) Read(reader io.Reader) *ReceivePaymentError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &ReceivePaymentError{&ReceivePaymentErrorGeneric{message}}
	case 2:
		return &ReceivePaymentError{&ReceivePaymentErrorInvalidAmount{message}}
	case 3:
		return &ReceivePaymentError{&ReceivePaymentErrorInvalidInvoice{message}}
	case 4:
		return &ReceivePaymentError{&ReceivePaymentErrorInvoiceExpired{message}}
	case 5:
		return &ReceivePaymentError{&ReceivePaymentErrorInvoiceNoDescription{message}}
	case 6:
		return &ReceivePaymentError{&ReceivePaymentErrorInvoicePreimageAlreadyExists{message}}
	case 7:
		return &ReceivePaymentError{&ReceivePaymentErrorServiceConnectivity{message}}
	case 8:
		return &ReceivePaymentError{&ReceivePaymentErrorInvoiceNoRoutingHints{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterReceivePaymentError.Read()", errorID))
	}

}

func (c FfiConverterReceivePaymentError) Write(writer io.Writer, value *ReceivePaymentError) {
	switch variantValue := value.err.(type) {
	case *ReceivePaymentErrorGeneric:
		writeInt32(writer, 1)
	case *ReceivePaymentErrorInvalidAmount:
		writeInt32(writer, 2)
	case *ReceivePaymentErrorInvalidInvoice:
		writeInt32(writer, 3)
	case *ReceivePaymentErrorInvoiceExpired:
		writeInt32(writer, 4)
	case *ReceivePaymentErrorInvoiceNoDescription:
		writeInt32(writer, 5)
	case *ReceivePaymentErrorInvoicePreimageAlreadyExists:
		writeInt32(writer, 6)
	case *ReceivePaymentErrorServiceConnectivity:
		writeInt32(writer, 7)
	case *ReceivePaymentErrorInvoiceNoRoutingHints:
		writeInt32(writer, 8)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterReceivePaymentError.Write", value))
	}
}

type FfiDestroyerReceivePaymentError struct{}

func (_ FfiDestroyerReceivePaymentError) Destroy(value *ReceivePaymentError) {
	switch variantValue := value.err.(type) {
	case ReceivePaymentErrorGeneric:
		variantValue.destroy()
	case ReceivePaymentErrorInvalidAmount:
		variantValue.destroy()
	case ReceivePaymentErrorInvalidInvoice:
		variantValue.destroy()
	case ReceivePaymentErrorInvoiceExpired:
		variantValue.destroy()
	case ReceivePaymentErrorInvoiceNoDescription:
		variantValue.destroy()
	case ReceivePaymentErrorInvoicePreimageAlreadyExists:
		variantValue.destroy()
	case ReceivePaymentErrorServiceConnectivity:
		variantValue.destroy()
	case ReceivePaymentErrorInvoiceNoRoutingHints:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerReceivePaymentError.Destroy", value))
	}
}

type RedeemOnchainError struct {
	err error
}

// Convience method to turn *RedeemOnchainError into error
// Avoiding treating nil pointer as non nil error interface
func (err *RedeemOnchainError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err RedeemOnchainError) Error() string {
	return fmt.Sprintf("RedeemOnchainError: %s", err.err.Error())
}

func (err RedeemOnchainError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrRedeemOnchainErrorGeneric = fmt.Errorf("RedeemOnchainErrorGeneric")
var ErrRedeemOnchainErrorServiceConnectivity = fmt.Errorf("RedeemOnchainErrorServiceConnectivity")
var ErrRedeemOnchainErrorInsufficientFunds = fmt.Errorf("RedeemOnchainErrorInsufficientFunds")

// Variant structs
type RedeemOnchainErrorGeneric struct {
	message string
}

func NewRedeemOnchainErrorGeneric() *RedeemOnchainError {
	return &RedeemOnchainError{err: &RedeemOnchainErrorGeneric{}}
}

func (e RedeemOnchainErrorGeneric) destroy() {
}

func (err RedeemOnchainErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self RedeemOnchainErrorGeneric) Is(target error) bool {
	return target == ErrRedeemOnchainErrorGeneric
}

type RedeemOnchainErrorServiceConnectivity struct {
	message string
}

func NewRedeemOnchainErrorServiceConnectivity() *RedeemOnchainError {
	return &RedeemOnchainError{err: &RedeemOnchainErrorServiceConnectivity{}}
}

func (e RedeemOnchainErrorServiceConnectivity) destroy() {
}

func (err RedeemOnchainErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self RedeemOnchainErrorServiceConnectivity) Is(target error) bool {
	return target == ErrRedeemOnchainErrorServiceConnectivity
}

type RedeemOnchainErrorInsufficientFunds struct {
	message string
}

func NewRedeemOnchainErrorInsufficientFunds() *RedeemOnchainError {
	return &RedeemOnchainError{err: &RedeemOnchainErrorInsufficientFunds{}}
}

func (e RedeemOnchainErrorInsufficientFunds) destroy() {
}

func (err RedeemOnchainErrorInsufficientFunds) Error() string {
	return fmt.Sprintf("InsufficientFunds: %s", err.message)
}

func (self RedeemOnchainErrorInsufficientFunds) Is(target error) bool {
	return target == ErrRedeemOnchainErrorInsufficientFunds
}

type FfiConverterRedeemOnchainError struct{}

var FfiConverterRedeemOnchainErrorINSTANCE = FfiConverterRedeemOnchainError{}

func (c FfiConverterRedeemOnchainError) Lift(eb RustBufferI) *RedeemOnchainError {
	return LiftFromRustBuffer[*RedeemOnchainError](c, eb)
}

func (c FfiConverterRedeemOnchainError) Lower(value *RedeemOnchainError) C.RustBuffer {
	return LowerIntoRustBuffer[*RedeemOnchainError](c, value)
}

func (c FfiConverterRedeemOnchainError) Read(reader io.Reader) *RedeemOnchainError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &RedeemOnchainError{&RedeemOnchainErrorGeneric{message}}
	case 2:
		return &RedeemOnchainError{&RedeemOnchainErrorServiceConnectivity{message}}
	case 3:
		return &RedeemOnchainError{&RedeemOnchainErrorInsufficientFunds{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterRedeemOnchainError.Read()", errorID))
	}

}

func (c FfiConverterRedeemOnchainError) Write(writer io.Writer, value *RedeemOnchainError) {
	switch variantValue := value.err.(type) {
	case *RedeemOnchainErrorGeneric:
		writeInt32(writer, 1)
	case *RedeemOnchainErrorServiceConnectivity:
		writeInt32(writer, 2)
	case *RedeemOnchainErrorInsufficientFunds:
		writeInt32(writer, 3)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterRedeemOnchainError.Write", value))
	}
}

type FfiDestroyerRedeemOnchainError struct{}

func (_ FfiDestroyerRedeemOnchainError) Destroy(value *RedeemOnchainError) {
	switch variantValue := value.err.(type) {
	case RedeemOnchainErrorGeneric:
		variantValue.destroy()
	case RedeemOnchainErrorServiceConnectivity:
		variantValue.destroy()
	case RedeemOnchainErrorInsufficientFunds:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerRedeemOnchainError.Destroy", value))
	}
}

type ReportIssueRequest interface {
	Destroy()
}
type ReportIssueRequestPaymentFailure struct {
	Data ReportPaymentFailureDetails
}

func (e ReportIssueRequestPaymentFailure) Destroy() {
	FfiDestroyerReportPaymentFailureDetails{}.Destroy(e.Data)
}

type FfiConverterReportIssueRequest struct{}

var FfiConverterReportIssueRequestINSTANCE = FfiConverterReportIssueRequest{}

func (c FfiConverterReportIssueRequest) Lift(rb RustBufferI) ReportIssueRequest {
	return LiftFromRustBuffer[ReportIssueRequest](c, rb)
}

func (c FfiConverterReportIssueRequest) Lower(value ReportIssueRequest) C.RustBuffer {
	return LowerIntoRustBuffer[ReportIssueRequest](c, value)
}
func (FfiConverterReportIssueRequest) Read(reader io.Reader) ReportIssueRequest {
	id := readInt32(reader)
	switch id {
	case 1:
		return ReportIssueRequestPaymentFailure{
			FfiConverterReportPaymentFailureDetailsINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterReportIssueRequest.Read()", id))
	}
}

func (FfiConverterReportIssueRequest) Write(writer io.Writer, value ReportIssueRequest) {
	switch variant_value := value.(type) {
	case ReportIssueRequestPaymentFailure:
		writeInt32(writer, 1)
		FfiConverterReportPaymentFailureDetailsINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterReportIssueRequest.Write", value))
	}
}

type FfiDestroyerReportIssueRequest struct{}

func (_ FfiDestroyerReportIssueRequest) Destroy(value ReportIssueRequest) {
	value.Destroy()
}

type ReverseSwapStatus uint

const (
	ReverseSwapStatusInitial            ReverseSwapStatus = 1
	ReverseSwapStatusInProgress         ReverseSwapStatus = 2
	ReverseSwapStatusCancelled          ReverseSwapStatus = 3
	ReverseSwapStatusCompletedSeen      ReverseSwapStatus = 4
	ReverseSwapStatusCompletedConfirmed ReverseSwapStatus = 5
)

type FfiConverterReverseSwapStatus struct{}

var FfiConverterReverseSwapStatusINSTANCE = FfiConverterReverseSwapStatus{}

func (c FfiConverterReverseSwapStatus) Lift(rb RustBufferI) ReverseSwapStatus {
	return LiftFromRustBuffer[ReverseSwapStatus](c, rb)
}

func (c FfiConverterReverseSwapStatus) Lower(value ReverseSwapStatus) C.RustBuffer {
	return LowerIntoRustBuffer[ReverseSwapStatus](c, value)
}
func (FfiConverterReverseSwapStatus) Read(reader io.Reader) ReverseSwapStatus {
	id := readInt32(reader)
	return ReverseSwapStatus(id)
}

func (FfiConverterReverseSwapStatus) Write(writer io.Writer, value ReverseSwapStatus) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerReverseSwapStatus struct{}

func (_ FfiDestroyerReverseSwapStatus) Destroy(value ReverseSwapStatus) {
}

type SdkError struct {
	err error
}

// Convience method to turn *SdkError into error
// Avoiding treating nil pointer as non nil error interface
func (err *SdkError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err SdkError) Error() string {
	return fmt.Sprintf("SdkError: %s", err.err.Error())
}

func (err SdkError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrSdkErrorGeneric = fmt.Errorf("SdkErrorGeneric")
var ErrSdkErrorServiceConnectivity = fmt.Errorf("SdkErrorServiceConnectivity")

// Variant structs
type SdkErrorGeneric struct {
	message string
}

func NewSdkErrorGeneric() *SdkError {
	return &SdkError{err: &SdkErrorGeneric{}}
}

func (e SdkErrorGeneric) destroy() {
}

func (err SdkErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self SdkErrorGeneric) Is(target error) bool {
	return target == ErrSdkErrorGeneric
}

type SdkErrorServiceConnectivity struct {
	message string
}

func NewSdkErrorServiceConnectivity() *SdkError {
	return &SdkError{err: &SdkErrorServiceConnectivity{}}
}

func (e SdkErrorServiceConnectivity) destroy() {
}

func (err SdkErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self SdkErrorServiceConnectivity) Is(target error) bool {
	return target == ErrSdkErrorServiceConnectivity
}

type FfiConverterSdkError struct{}

var FfiConverterSdkErrorINSTANCE = FfiConverterSdkError{}

func (c FfiConverterSdkError) Lift(eb RustBufferI) *SdkError {
	return LiftFromRustBuffer[*SdkError](c, eb)
}

func (c FfiConverterSdkError) Lower(value *SdkError) C.RustBuffer {
	return LowerIntoRustBuffer[*SdkError](c, value)
}

func (c FfiConverterSdkError) Read(reader io.Reader) *SdkError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &SdkError{&SdkErrorGeneric{message}}
	case 2:
		return &SdkError{&SdkErrorServiceConnectivity{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterSdkError.Read()", errorID))
	}

}

func (c FfiConverterSdkError) Write(writer io.Writer, value *SdkError) {
	switch variantValue := value.err.(type) {
	case *SdkErrorGeneric:
		writeInt32(writer, 1)
	case *SdkErrorServiceConnectivity:
		writeInt32(writer, 2)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterSdkError.Write", value))
	}
}

type FfiDestroyerSdkError struct{}

func (_ FfiDestroyerSdkError) Destroy(value *SdkError) {
	switch variantValue := value.err.(type) {
	case SdkErrorGeneric:
		variantValue.destroy()
	case SdkErrorServiceConnectivity:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerSdkError.Destroy", value))
	}
}

type SendOnchainError struct {
	err error
}

// Convience method to turn *SendOnchainError into error
// Avoiding treating nil pointer as non nil error interface
func (err *SendOnchainError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err SendOnchainError) Error() string {
	return fmt.Sprintf("SendOnchainError: %s", err.err.Error())
}

func (err SendOnchainError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrSendOnchainErrorGeneric = fmt.Errorf("SendOnchainErrorGeneric")
var ErrSendOnchainErrorInvalidDestinationAddress = fmt.Errorf("SendOnchainErrorInvalidDestinationAddress")
var ErrSendOnchainErrorOutOfRange = fmt.Errorf("SendOnchainErrorOutOfRange")
var ErrSendOnchainErrorPaymentFailed = fmt.Errorf("SendOnchainErrorPaymentFailed")
var ErrSendOnchainErrorPaymentTimeout = fmt.Errorf("SendOnchainErrorPaymentTimeout")
var ErrSendOnchainErrorServiceConnectivity = fmt.Errorf("SendOnchainErrorServiceConnectivity")

// Variant structs
type SendOnchainErrorGeneric struct {
	message string
}

func NewSendOnchainErrorGeneric() *SendOnchainError {
	return &SendOnchainError{err: &SendOnchainErrorGeneric{}}
}

func (e SendOnchainErrorGeneric) destroy() {
}

func (err SendOnchainErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self SendOnchainErrorGeneric) Is(target error) bool {
	return target == ErrSendOnchainErrorGeneric
}

type SendOnchainErrorInvalidDestinationAddress struct {
	message string
}

func NewSendOnchainErrorInvalidDestinationAddress() *SendOnchainError {
	return &SendOnchainError{err: &SendOnchainErrorInvalidDestinationAddress{}}
}

func (e SendOnchainErrorInvalidDestinationAddress) destroy() {
}

func (err SendOnchainErrorInvalidDestinationAddress) Error() string {
	return fmt.Sprintf("InvalidDestinationAddress: %s", err.message)
}

func (self SendOnchainErrorInvalidDestinationAddress) Is(target error) bool {
	return target == ErrSendOnchainErrorInvalidDestinationAddress
}

type SendOnchainErrorOutOfRange struct {
	message string
}

func NewSendOnchainErrorOutOfRange() *SendOnchainError {
	return &SendOnchainError{err: &SendOnchainErrorOutOfRange{}}
}

func (e SendOnchainErrorOutOfRange) destroy() {
}

func (err SendOnchainErrorOutOfRange) Error() string {
	return fmt.Sprintf("OutOfRange: %s", err.message)
}

func (self SendOnchainErrorOutOfRange) Is(target error) bool {
	return target == ErrSendOnchainErrorOutOfRange
}

type SendOnchainErrorPaymentFailed struct {
	message string
}

func NewSendOnchainErrorPaymentFailed() *SendOnchainError {
	return &SendOnchainError{err: &SendOnchainErrorPaymentFailed{}}
}

func (e SendOnchainErrorPaymentFailed) destroy() {
}

func (err SendOnchainErrorPaymentFailed) Error() string {
	return fmt.Sprintf("PaymentFailed: %s", err.message)
}

func (self SendOnchainErrorPaymentFailed) Is(target error) bool {
	return target == ErrSendOnchainErrorPaymentFailed
}

type SendOnchainErrorPaymentTimeout struct {
	message string
}

func NewSendOnchainErrorPaymentTimeout() *SendOnchainError {
	return &SendOnchainError{err: &SendOnchainErrorPaymentTimeout{}}
}

func (e SendOnchainErrorPaymentTimeout) destroy() {
}

func (err SendOnchainErrorPaymentTimeout) Error() string {
	return fmt.Sprintf("PaymentTimeout: %s", err.message)
}

func (self SendOnchainErrorPaymentTimeout) Is(target error) bool {
	return target == ErrSendOnchainErrorPaymentTimeout
}

type SendOnchainErrorServiceConnectivity struct {
	message string
}

func NewSendOnchainErrorServiceConnectivity() *SendOnchainError {
	return &SendOnchainError{err: &SendOnchainErrorServiceConnectivity{}}
}

func (e SendOnchainErrorServiceConnectivity) destroy() {
}

func (err SendOnchainErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self SendOnchainErrorServiceConnectivity) Is(target error) bool {
	return target == ErrSendOnchainErrorServiceConnectivity
}

type FfiConverterSendOnchainError struct{}

var FfiConverterSendOnchainErrorINSTANCE = FfiConverterSendOnchainError{}

func (c FfiConverterSendOnchainError) Lift(eb RustBufferI) *SendOnchainError {
	return LiftFromRustBuffer[*SendOnchainError](c, eb)
}

func (c FfiConverterSendOnchainError) Lower(value *SendOnchainError) C.RustBuffer {
	return LowerIntoRustBuffer[*SendOnchainError](c, value)
}

func (c FfiConverterSendOnchainError) Read(reader io.Reader) *SendOnchainError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &SendOnchainError{&SendOnchainErrorGeneric{message}}
	case 2:
		return &SendOnchainError{&SendOnchainErrorInvalidDestinationAddress{message}}
	case 3:
		return &SendOnchainError{&SendOnchainErrorOutOfRange{message}}
	case 4:
		return &SendOnchainError{&SendOnchainErrorPaymentFailed{message}}
	case 5:
		return &SendOnchainError{&SendOnchainErrorPaymentTimeout{message}}
	case 6:
		return &SendOnchainError{&SendOnchainErrorServiceConnectivity{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterSendOnchainError.Read()", errorID))
	}

}

func (c FfiConverterSendOnchainError) Write(writer io.Writer, value *SendOnchainError) {
	switch variantValue := value.err.(type) {
	case *SendOnchainErrorGeneric:
		writeInt32(writer, 1)
	case *SendOnchainErrorInvalidDestinationAddress:
		writeInt32(writer, 2)
	case *SendOnchainErrorOutOfRange:
		writeInt32(writer, 3)
	case *SendOnchainErrorPaymentFailed:
		writeInt32(writer, 4)
	case *SendOnchainErrorPaymentTimeout:
		writeInt32(writer, 5)
	case *SendOnchainErrorServiceConnectivity:
		writeInt32(writer, 6)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterSendOnchainError.Write", value))
	}
}

type FfiDestroyerSendOnchainError struct{}

func (_ FfiDestroyerSendOnchainError) Destroy(value *SendOnchainError) {
	switch variantValue := value.err.(type) {
	case SendOnchainErrorGeneric:
		variantValue.destroy()
	case SendOnchainErrorInvalidDestinationAddress:
		variantValue.destroy()
	case SendOnchainErrorOutOfRange:
		variantValue.destroy()
	case SendOnchainErrorPaymentFailed:
		variantValue.destroy()
	case SendOnchainErrorPaymentTimeout:
		variantValue.destroy()
	case SendOnchainErrorServiceConnectivity:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerSendOnchainError.Destroy", value))
	}
}

type SendPaymentError struct {
	err error
}

// Convience method to turn *SendPaymentError into error
// Avoiding treating nil pointer as non nil error interface
func (err *SendPaymentError) AsError() error {
	if err == nil {
		return nil
	} else {
		return err
	}
}

func (err SendPaymentError) Error() string {
	return fmt.Sprintf("SendPaymentError: %s", err.err.Error())
}

func (err SendPaymentError) Unwrap() error {
	return err.err
}

// Err* are used for checking error type with `errors.Is`
var ErrSendPaymentErrorAlreadyPaid = fmt.Errorf("SendPaymentErrorAlreadyPaid")
var ErrSendPaymentErrorGeneric = fmt.Errorf("SendPaymentErrorGeneric")
var ErrSendPaymentErrorInvalidAmount = fmt.Errorf("SendPaymentErrorInvalidAmount")
var ErrSendPaymentErrorInvalidInvoice = fmt.Errorf("SendPaymentErrorInvalidInvoice")
var ErrSendPaymentErrorInvoiceExpired = fmt.Errorf("SendPaymentErrorInvoiceExpired")
var ErrSendPaymentErrorInvalidNetwork = fmt.Errorf("SendPaymentErrorInvalidNetwork")
var ErrSendPaymentErrorPaymentFailed = fmt.Errorf("SendPaymentErrorPaymentFailed")
var ErrSendPaymentErrorPaymentTimeout = fmt.Errorf("SendPaymentErrorPaymentTimeout")
var ErrSendPaymentErrorRouteNotFound = fmt.Errorf("SendPaymentErrorRouteNotFound")
var ErrSendPaymentErrorRouteTooExpensive = fmt.Errorf("SendPaymentErrorRouteTooExpensive")
var ErrSendPaymentErrorServiceConnectivity = fmt.Errorf("SendPaymentErrorServiceConnectivity")
var ErrSendPaymentErrorInsufficientBalance = fmt.Errorf("SendPaymentErrorInsufficientBalance")

// Variant structs
type SendPaymentErrorAlreadyPaid struct {
	message string
}

func NewSendPaymentErrorAlreadyPaid() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorAlreadyPaid{}}
}

func (e SendPaymentErrorAlreadyPaid) destroy() {
}

func (err SendPaymentErrorAlreadyPaid) Error() string {
	return fmt.Sprintf("AlreadyPaid: %s", err.message)
}

func (self SendPaymentErrorAlreadyPaid) Is(target error) bool {
	return target == ErrSendPaymentErrorAlreadyPaid
}

type SendPaymentErrorGeneric struct {
	message string
}

func NewSendPaymentErrorGeneric() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorGeneric{}}
}

func (e SendPaymentErrorGeneric) destroy() {
}

func (err SendPaymentErrorGeneric) Error() string {
	return fmt.Sprintf("Generic: %s", err.message)
}

func (self SendPaymentErrorGeneric) Is(target error) bool {
	return target == ErrSendPaymentErrorGeneric
}

type SendPaymentErrorInvalidAmount struct {
	message string
}

func NewSendPaymentErrorInvalidAmount() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorInvalidAmount{}}
}

func (e SendPaymentErrorInvalidAmount) destroy() {
}

func (err SendPaymentErrorInvalidAmount) Error() string {
	return fmt.Sprintf("InvalidAmount: %s", err.message)
}

func (self SendPaymentErrorInvalidAmount) Is(target error) bool {
	return target == ErrSendPaymentErrorInvalidAmount
}

type SendPaymentErrorInvalidInvoice struct {
	message string
}

func NewSendPaymentErrorInvalidInvoice() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorInvalidInvoice{}}
}

func (e SendPaymentErrorInvalidInvoice) destroy() {
}

func (err SendPaymentErrorInvalidInvoice) Error() string {
	return fmt.Sprintf("InvalidInvoice: %s", err.message)
}

func (self SendPaymentErrorInvalidInvoice) Is(target error) bool {
	return target == ErrSendPaymentErrorInvalidInvoice
}

type SendPaymentErrorInvoiceExpired struct {
	message string
}

func NewSendPaymentErrorInvoiceExpired() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorInvoiceExpired{}}
}

func (e SendPaymentErrorInvoiceExpired) destroy() {
}

func (err SendPaymentErrorInvoiceExpired) Error() string {
	return fmt.Sprintf("InvoiceExpired: %s", err.message)
}

func (self SendPaymentErrorInvoiceExpired) Is(target error) bool {
	return target == ErrSendPaymentErrorInvoiceExpired
}

type SendPaymentErrorInvalidNetwork struct {
	message string
}

func NewSendPaymentErrorInvalidNetwork() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorInvalidNetwork{}}
}

func (e SendPaymentErrorInvalidNetwork) destroy() {
}

func (err SendPaymentErrorInvalidNetwork) Error() string {
	return fmt.Sprintf("InvalidNetwork: %s", err.message)
}

func (self SendPaymentErrorInvalidNetwork) Is(target error) bool {
	return target == ErrSendPaymentErrorInvalidNetwork
}

type SendPaymentErrorPaymentFailed struct {
	message string
}

func NewSendPaymentErrorPaymentFailed() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorPaymentFailed{}}
}

func (e SendPaymentErrorPaymentFailed) destroy() {
}

func (err SendPaymentErrorPaymentFailed) Error() string {
	return fmt.Sprintf("PaymentFailed: %s", err.message)
}

func (self SendPaymentErrorPaymentFailed) Is(target error) bool {
	return target == ErrSendPaymentErrorPaymentFailed
}

type SendPaymentErrorPaymentTimeout struct {
	message string
}

func NewSendPaymentErrorPaymentTimeout() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorPaymentTimeout{}}
}

func (e SendPaymentErrorPaymentTimeout) destroy() {
}

func (err SendPaymentErrorPaymentTimeout) Error() string {
	return fmt.Sprintf("PaymentTimeout: %s", err.message)
}

func (self SendPaymentErrorPaymentTimeout) Is(target error) bool {
	return target == ErrSendPaymentErrorPaymentTimeout
}

type SendPaymentErrorRouteNotFound struct {
	message string
}

func NewSendPaymentErrorRouteNotFound() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorRouteNotFound{}}
}

func (e SendPaymentErrorRouteNotFound) destroy() {
}

func (err SendPaymentErrorRouteNotFound) Error() string {
	return fmt.Sprintf("RouteNotFound: %s", err.message)
}

func (self SendPaymentErrorRouteNotFound) Is(target error) bool {
	return target == ErrSendPaymentErrorRouteNotFound
}

type SendPaymentErrorRouteTooExpensive struct {
	message string
}

func NewSendPaymentErrorRouteTooExpensive() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorRouteTooExpensive{}}
}

func (e SendPaymentErrorRouteTooExpensive) destroy() {
}

func (err SendPaymentErrorRouteTooExpensive) Error() string {
	return fmt.Sprintf("RouteTooExpensive: %s", err.message)
}

func (self SendPaymentErrorRouteTooExpensive) Is(target error) bool {
	return target == ErrSendPaymentErrorRouteTooExpensive
}

type SendPaymentErrorServiceConnectivity struct {
	message string
}

func NewSendPaymentErrorServiceConnectivity() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorServiceConnectivity{}}
}

func (e SendPaymentErrorServiceConnectivity) destroy() {
}

func (err SendPaymentErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self SendPaymentErrorServiceConnectivity) Is(target error) bool {
	return target == ErrSendPaymentErrorServiceConnectivity
}

type SendPaymentErrorInsufficientBalance struct {
	message string
}

func NewSendPaymentErrorInsufficientBalance() *SendPaymentError {
	return &SendPaymentError{err: &SendPaymentErrorInsufficientBalance{}}
}

func (e SendPaymentErrorInsufficientBalance) destroy() {
}

func (err SendPaymentErrorInsufficientBalance) Error() string {
	return fmt.Sprintf("InsufficientBalance: %s", err.message)
}

func (self SendPaymentErrorInsufficientBalance) Is(target error) bool {
	return target == ErrSendPaymentErrorInsufficientBalance
}

type FfiConverterSendPaymentError struct{}

var FfiConverterSendPaymentErrorINSTANCE = FfiConverterSendPaymentError{}

func (c FfiConverterSendPaymentError) Lift(eb RustBufferI) *SendPaymentError {
	return LiftFromRustBuffer[*SendPaymentError](c, eb)
}

func (c FfiConverterSendPaymentError) Lower(value *SendPaymentError) C.RustBuffer {
	return LowerIntoRustBuffer[*SendPaymentError](c, value)
}

func (c FfiConverterSendPaymentError) Read(reader io.Reader) *SendPaymentError {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &SendPaymentError{&SendPaymentErrorAlreadyPaid{message}}
	case 2:
		return &SendPaymentError{&SendPaymentErrorGeneric{message}}
	case 3:
		return &SendPaymentError{&SendPaymentErrorInvalidAmount{message}}
	case 4:
		return &SendPaymentError{&SendPaymentErrorInvalidInvoice{message}}
	case 5:
		return &SendPaymentError{&SendPaymentErrorInvoiceExpired{message}}
	case 6:
		return &SendPaymentError{&SendPaymentErrorInvalidNetwork{message}}
	case 7:
		return &SendPaymentError{&SendPaymentErrorPaymentFailed{message}}
	case 8:
		return &SendPaymentError{&SendPaymentErrorPaymentTimeout{message}}
	case 9:
		return &SendPaymentError{&SendPaymentErrorRouteNotFound{message}}
	case 10:
		return &SendPaymentError{&SendPaymentErrorRouteTooExpensive{message}}
	case 11:
		return &SendPaymentError{&SendPaymentErrorServiceConnectivity{message}}
	case 12:
		return &SendPaymentError{&SendPaymentErrorInsufficientBalance{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterSendPaymentError.Read()", errorID))
	}

}

func (c FfiConverterSendPaymentError) Write(writer io.Writer, value *SendPaymentError) {
	switch variantValue := value.err.(type) {
	case *SendPaymentErrorAlreadyPaid:
		writeInt32(writer, 1)
	case *SendPaymentErrorGeneric:
		writeInt32(writer, 2)
	case *SendPaymentErrorInvalidAmount:
		writeInt32(writer, 3)
	case *SendPaymentErrorInvalidInvoice:
		writeInt32(writer, 4)
	case *SendPaymentErrorInvoiceExpired:
		writeInt32(writer, 5)
	case *SendPaymentErrorInvalidNetwork:
		writeInt32(writer, 6)
	case *SendPaymentErrorPaymentFailed:
		writeInt32(writer, 7)
	case *SendPaymentErrorPaymentTimeout:
		writeInt32(writer, 8)
	case *SendPaymentErrorRouteNotFound:
		writeInt32(writer, 9)
	case *SendPaymentErrorRouteTooExpensive:
		writeInt32(writer, 10)
	case *SendPaymentErrorServiceConnectivity:
		writeInt32(writer, 11)
	case *SendPaymentErrorInsufficientBalance:
		writeInt32(writer, 12)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterSendPaymentError.Write", value))
	}
}

type FfiDestroyerSendPaymentError struct{}

func (_ FfiDestroyerSendPaymentError) Destroy(value *SendPaymentError) {
	switch variantValue := value.err.(type) {
	case SendPaymentErrorAlreadyPaid:
		variantValue.destroy()
	case SendPaymentErrorGeneric:
		variantValue.destroy()
	case SendPaymentErrorInvalidAmount:
		variantValue.destroy()
	case SendPaymentErrorInvalidInvoice:
		variantValue.destroy()
	case SendPaymentErrorInvoiceExpired:
		variantValue.destroy()
	case SendPaymentErrorInvalidNetwork:
		variantValue.destroy()
	case SendPaymentErrorPaymentFailed:
		variantValue.destroy()
	case SendPaymentErrorPaymentTimeout:
		variantValue.destroy()
	case SendPaymentErrorRouteNotFound:
		variantValue.destroy()
	case SendPaymentErrorRouteTooExpensive:
		variantValue.destroy()
	case SendPaymentErrorServiceConnectivity:
		variantValue.destroy()
	case SendPaymentErrorInsufficientBalance:
		variantValue.destroy()
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiDestroyerSendPaymentError.Destroy", value))
	}
}

type SuccessActionProcessed interface {
	Destroy()
}
type SuccessActionProcessedAes struct {
	Result AesSuccessActionDataResult
}

func (e SuccessActionProcessedAes) Destroy() {
	FfiDestroyerAesSuccessActionDataResult{}.Destroy(e.Result)
}

type SuccessActionProcessedMessage struct {
	Data MessageSuccessActionData
}

func (e SuccessActionProcessedMessage) Destroy() {
	FfiDestroyerMessageSuccessActionData{}.Destroy(e.Data)
}

type SuccessActionProcessedUrl struct {
	Data UrlSuccessActionData
}

func (e SuccessActionProcessedUrl) Destroy() {
	FfiDestroyerUrlSuccessActionData{}.Destroy(e.Data)
}

type FfiConverterSuccessActionProcessed struct{}

var FfiConverterSuccessActionProcessedINSTANCE = FfiConverterSuccessActionProcessed{}

func (c FfiConverterSuccessActionProcessed) Lift(rb RustBufferI) SuccessActionProcessed {
	return LiftFromRustBuffer[SuccessActionProcessed](c, rb)
}

func (c FfiConverterSuccessActionProcessed) Lower(value SuccessActionProcessed) C.RustBuffer {
	return LowerIntoRustBuffer[SuccessActionProcessed](c, value)
}
func (FfiConverterSuccessActionProcessed) Read(reader io.Reader) SuccessActionProcessed {
	id := readInt32(reader)
	switch id {
	case 1:
		return SuccessActionProcessedAes{
			FfiConverterAesSuccessActionDataResultINSTANCE.Read(reader),
		}
	case 2:
		return SuccessActionProcessedMessage{
			FfiConverterMessageSuccessActionDataINSTANCE.Read(reader),
		}
	case 3:
		return SuccessActionProcessedUrl{
			FfiConverterUrlSuccessActionDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterSuccessActionProcessed.Read()", id))
	}
}

func (FfiConverterSuccessActionProcessed) Write(writer io.Writer, value SuccessActionProcessed) {
	switch variant_value := value.(type) {
	case SuccessActionProcessedAes:
		writeInt32(writer, 1)
		FfiConverterAesSuccessActionDataResultINSTANCE.Write(writer, variant_value.Result)
	case SuccessActionProcessedMessage:
		writeInt32(writer, 2)
		FfiConverterMessageSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	case SuccessActionProcessedUrl:
		writeInt32(writer, 3)
		FfiConverterUrlSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterSuccessActionProcessed.Write", value))
	}
}

type FfiDestroyerSuccessActionProcessed struct{}

func (_ FfiDestroyerSuccessActionProcessed) Destroy(value SuccessActionProcessed) {
	value.Destroy()
}

type SwapAmountType uint

const (
	SwapAmountTypeSend    SwapAmountType = 1
	SwapAmountTypeReceive SwapAmountType = 2
)

type FfiConverterSwapAmountType struct{}

var FfiConverterSwapAmountTypeINSTANCE = FfiConverterSwapAmountType{}

func (c FfiConverterSwapAmountType) Lift(rb RustBufferI) SwapAmountType {
	return LiftFromRustBuffer[SwapAmountType](c, rb)
}

func (c FfiConverterSwapAmountType) Lower(value SwapAmountType) C.RustBuffer {
	return LowerIntoRustBuffer[SwapAmountType](c, value)
}
func (FfiConverterSwapAmountType) Read(reader io.Reader) SwapAmountType {
	id := readInt32(reader)
	return SwapAmountType(id)
}

func (FfiConverterSwapAmountType) Write(writer io.Writer, value SwapAmountType) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerSwapAmountType struct{}

func (_ FfiDestroyerSwapAmountType) Destroy(value SwapAmountType) {
}

type SwapStatus uint

const (
	SwapStatusInitial             SwapStatus = 1
	SwapStatusWaitingConfirmation SwapStatus = 2
	SwapStatusRedeemable          SwapStatus = 3
	SwapStatusRedeemed            SwapStatus = 4
	SwapStatusRefundable          SwapStatus = 5
	SwapStatusCompleted           SwapStatus = 6
)

type FfiConverterSwapStatus struct{}

var FfiConverterSwapStatusINSTANCE = FfiConverterSwapStatus{}

func (c FfiConverterSwapStatus) Lift(rb RustBufferI) SwapStatus {
	return LiftFromRustBuffer[SwapStatus](c, rb)
}

func (c FfiConverterSwapStatus) Lower(value SwapStatus) C.RustBuffer {
	return LowerIntoRustBuffer[SwapStatus](c, value)
}
func (FfiConverterSwapStatus) Read(reader io.Reader) SwapStatus {
	id := readInt32(reader)
	return SwapStatus(id)
}

func (FfiConverterSwapStatus) Write(writer io.Writer, value SwapStatus) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerSwapStatus struct{}

func (_ FfiDestroyerSwapStatus) Destroy(value SwapStatus) {
}

type EventListener interface {
	OnEvent(e BreezEvent)
}

type FfiConverterCallbackInterfaceEventListener struct {
	handleMap *concurrentHandleMap[EventListener]
}

var FfiConverterCallbackInterfaceEventListenerINSTANCE = FfiConverterCallbackInterfaceEventListener{
	handleMap: newConcurrentHandleMap[EventListener](),
}

func (c FfiConverterCallbackInterfaceEventListener) Lift(handle uint64) EventListener {
	val, ok := c.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}
	return val
}

func (c FfiConverterCallbackInterfaceEventListener) Read(reader io.Reader) EventListener {
	return c.Lift(readUint64(reader))
}

func (c FfiConverterCallbackInterfaceEventListener) Lower(value EventListener) C.uint64_t {
	return C.uint64_t(c.handleMap.insert(value))
}

func (c FfiConverterCallbackInterfaceEventListener) Write(writer io.Writer, value EventListener) {
	writeUint64(writer, uint64(c.Lower(value)))
}

type FfiDestroyerCallbackInterfaceEventListener struct{}

func (FfiDestroyerCallbackInterfaceEventListener) Destroy(value EventListener) {}

type uniffiCallbackResult C.int8_t

const (
	uniffiIdxCallbackFree               uniffiCallbackResult = 0
	uniffiCallbackResultSuccess         uniffiCallbackResult = 0
	uniffiCallbackResultError           uniffiCallbackResult = 1
	uniffiCallbackUnexpectedResultError uniffiCallbackResult = 2
	uniffiCallbackCancelled             uniffiCallbackResult = 3
)

type concurrentHandleMap[T any] struct {
	handles       map[uint64]T
	currentHandle uint64
	lock          sync.RWMutex
}

func newConcurrentHandleMap[T any]() *concurrentHandleMap[T] {
	return &concurrentHandleMap[T]{
		handles: map[uint64]T{},
	}
}

func (cm *concurrentHandleMap[T]) insert(obj T) uint64 {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.currentHandle = cm.currentHandle + 1
	cm.handles[cm.currentHandle] = obj
	return cm.currentHandle
}

func (cm *concurrentHandleMap[T]) remove(handle uint64) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	delete(cm.handles, handle)
}

func (cm *concurrentHandleMap[T]) tryGet(handle uint64) (T, bool) {
	cm.lock.RLock()
	defer cm.lock.RUnlock()

	val, ok := cm.handles[handle]
	return val, ok
}

//export breez_sdk_bindings_cgo_dispatchCallbackInterfaceEventListenerMethod0
func breez_sdk_bindings_cgo_dispatchCallbackInterfaceEventListenerMethod0(uniffiHandle C.uint64_t, e C.RustBuffer, uniffiOutReturn *C.void, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceEventListenerINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	uniffiObj.OnEvent(
		FfiConverterBreezEventINSTANCE.Lift(GoRustBuffer{
			inner: e,
		}),
	)

}

var UniffiVTableCallbackInterfaceEventListenerINSTANCE = C.UniffiVTableCallbackInterfaceEventListener{
	onEvent: (C.UniffiCallbackInterfaceEventListenerMethod0)(C.breez_sdk_bindings_cgo_dispatchCallbackInterfaceEventListenerMethod0),

	uniffiFree: (C.UniffiCallbackInterfaceFree)(C.breez_sdk_bindings_cgo_dispatchCallbackInterfaceEventListenerFree),
}

//export breez_sdk_bindings_cgo_dispatchCallbackInterfaceEventListenerFree
func breez_sdk_bindings_cgo_dispatchCallbackInterfaceEventListenerFree(handle C.uint64_t) {
	FfiConverterCallbackInterfaceEventListenerINSTANCE.handleMap.remove(uint64(handle))
}

func (c FfiConverterCallbackInterfaceEventListener) register() {
	C.uniffi_breez_sdk_bindings_fn_init_callback_vtable_eventlistener(&UniffiVTableCallbackInterfaceEventListenerINSTANCE)
}

type LogStream interface {
	Log(l LogEntry)
}

type FfiConverterCallbackInterfaceLogStream struct {
	handleMap *concurrentHandleMap[LogStream]
}

var FfiConverterCallbackInterfaceLogStreamINSTANCE = FfiConverterCallbackInterfaceLogStream{
	handleMap: newConcurrentHandleMap[LogStream](),
}

func (c FfiConverterCallbackInterfaceLogStream) Lift(handle uint64) LogStream {
	val, ok := c.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}
	return val
}

func (c FfiConverterCallbackInterfaceLogStream) Read(reader io.Reader) LogStream {
	return c.Lift(readUint64(reader))
}

func (c FfiConverterCallbackInterfaceLogStream) Lower(value LogStream) C.uint64_t {
	return C.uint64_t(c.handleMap.insert(value))
}

func (c FfiConverterCallbackInterfaceLogStream) Write(writer io.Writer, value LogStream) {
	writeUint64(writer, uint64(c.Lower(value)))
}

type FfiDestroyerCallbackInterfaceLogStream struct{}

func (FfiDestroyerCallbackInterfaceLogStream) Destroy(value LogStream) {}

//export breez_sdk_bindings_cgo_dispatchCallbackInterfaceLogStreamMethod0
func breez_sdk_bindings_cgo_dispatchCallbackInterfaceLogStreamMethod0(uniffiHandle C.uint64_t, l C.RustBuffer, uniffiOutReturn *C.void, callStatus *C.RustCallStatus) {
	handle := uint64(uniffiHandle)
	uniffiObj, ok := FfiConverterCallbackInterfaceLogStreamINSTANCE.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}

	uniffiObj.Log(
		FfiConverterLogEntryINSTANCE.Lift(GoRustBuffer{
			inner: l,
		}),
	)

}

var UniffiVTableCallbackInterfaceLogStreamINSTANCE = C.UniffiVTableCallbackInterfaceLogStream{
	log: (C.UniffiCallbackInterfaceLogStreamMethod0)(C.breez_sdk_bindings_cgo_dispatchCallbackInterfaceLogStreamMethod0),

	uniffiFree: (C.UniffiCallbackInterfaceFree)(C.breez_sdk_bindings_cgo_dispatchCallbackInterfaceLogStreamFree),
}

//export breez_sdk_bindings_cgo_dispatchCallbackInterfaceLogStreamFree
func breez_sdk_bindings_cgo_dispatchCallbackInterfaceLogStreamFree(handle C.uint64_t) {
	FfiConverterCallbackInterfaceLogStreamINSTANCE.handleMap.remove(uint64(handle))
}

func (c FfiConverterCallbackInterfaceLogStream) register() {
	C.uniffi_breez_sdk_bindings_fn_init_callback_vtable_logstream(&UniffiVTableCallbackInterfaceLogStreamINSTANCE)
}

type FfiConverterOptionalUint32 struct{}

var FfiConverterOptionalUint32INSTANCE = FfiConverterOptionalUint32{}

func (c FfiConverterOptionalUint32) Lift(rb RustBufferI) *uint32 {
	return LiftFromRustBuffer[*uint32](c, rb)
}

func (_ FfiConverterOptionalUint32) Read(reader io.Reader) *uint32 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterUint32INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalUint32) Lower(value *uint32) C.RustBuffer {
	return LowerIntoRustBuffer[*uint32](c, value)
}

func (_ FfiConverterOptionalUint32) Write(writer io.Writer, value *uint32) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterUint32INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalUint32 struct{}

func (_ FfiDestroyerOptionalUint32) Destroy(value *uint32) {
	if value != nil {
		FfiDestroyerUint32{}.Destroy(*value)
	}
}

type FfiConverterOptionalUint64 struct{}

var FfiConverterOptionalUint64INSTANCE = FfiConverterOptionalUint64{}

func (c FfiConverterOptionalUint64) Lift(rb RustBufferI) *uint64 {
	return LiftFromRustBuffer[*uint64](c, rb)
}

func (_ FfiConverterOptionalUint64) Read(reader io.Reader) *uint64 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterUint64INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalUint64) Lower(value *uint64) C.RustBuffer {
	return LowerIntoRustBuffer[*uint64](c, value)
}

func (_ FfiConverterOptionalUint64) Write(writer io.Writer, value *uint64) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterUint64INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalUint64 struct{}

func (_ FfiDestroyerOptionalUint64) Destroy(value *uint64) {
	if value != nil {
		FfiDestroyerUint64{}.Destroy(*value)
	}
}

type FfiConverterOptionalInt64 struct{}

var FfiConverterOptionalInt64INSTANCE = FfiConverterOptionalInt64{}

func (c FfiConverterOptionalInt64) Lift(rb RustBufferI) *int64 {
	return LiftFromRustBuffer[*int64](c, rb)
}

func (_ FfiConverterOptionalInt64) Read(reader io.Reader) *int64 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterInt64INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalInt64) Lower(value *int64) C.RustBuffer {
	return LowerIntoRustBuffer[*int64](c, value)
}

func (_ FfiConverterOptionalInt64) Write(writer io.Writer, value *int64) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterInt64INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalInt64 struct{}

func (_ FfiDestroyerOptionalInt64) Destroy(value *int64) {
	if value != nil {
		FfiDestroyerInt64{}.Destroy(*value)
	}
}

type FfiConverterOptionalBool struct{}

var FfiConverterOptionalBoolINSTANCE = FfiConverterOptionalBool{}

func (c FfiConverterOptionalBool) Lift(rb RustBufferI) *bool {
	return LiftFromRustBuffer[*bool](c, rb)
}

func (_ FfiConverterOptionalBool) Read(reader io.Reader) *bool {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterBoolINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalBool) Lower(value *bool) C.RustBuffer {
	return LowerIntoRustBuffer[*bool](c, value)
}

func (_ FfiConverterOptionalBool) Write(writer io.Writer, value *bool) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterBoolINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalBool struct{}

func (_ FfiDestroyerOptionalBool) Destroy(value *bool) {
	if value != nil {
		FfiDestroyerBool{}.Destroy(*value)
	}
}

type FfiConverterOptionalString struct{}

var FfiConverterOptionalStringINSTANCE = FfiConverterOptionalString{}

func (c FfiConverterOptionalString) Lift(rb RustBufferI) *string {
	return LiftFromRustBuffer[*string](c, rb)
}

func (_ FfiConverterOptionalString) Read(reader io.Reader) *string {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterStringINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalString) Lower(value *string) C.RustBuffer {
	return LowerIntoRustBuffer[*string](c, value)
}

func (_ FfiConverterOptionalString) Write(writer io.Writer, value *string) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterStringINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalString struct{}

func (_ FfiDestroyerOptionalString) Destroy(value *string) {
	if value != nil {
		FfiDestroyerString{}.Destroy(*value)
	}
}

type FfiConverterOptionalGreenlightCredentials struct{}

var FfiConverterOptionalGreenlightCredentialsINSTANCE = FfiConverterOptionalGreenlightCredentials{}

func (c FfiConverterOptionalGreenlightCredentials) Lift(rb RustBufferI) *GreenlightCredentials {
	return LiftFromRustBuffer[*GreenlightCredentials](c, rb)
}

func (_ FfiConverterOptionalGreenlightCredentials) Read(reader io.Reader) *GreenlightCredentials {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterGreenlightCredentialsINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalGreenlightCredentials) Lower(value *GreenlightCredentials) C.RustBuffer {
	return LowerIntoRustBuffer[*GreenlightCredentials](c, value)
}

func (_ FfiConverterOptionalGreenlightCredentials) Write(writer io.Writer, value *GreenlightCredentials) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterGreenlightCredentialsINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalGreenlightCredentials struct{}

func (_ FfiDestroyerOptionalGreenlightCredentials) Destroy(value *GreenlightCredentials) {
	if value != nil {
		FfiDestroyerGreenlightCredentials{}.Destroy(*value)
	}
}

type FfiConverterOptionalLnInvoice struct{}

var FfiConverterOptionalLnInvoiceINSTANCE = FfiConverterOptionalLnInvoice{}

func (c FfiConverterOptionalLnInvoice) Lift(rb RustBufferI) *LnInvoice {
	return LiftFromRustBuffer[*LnInvoice](c, rb)
}

func (_ FfiConverterOptionalLnInvoice) Read(reader io.Reader) *LnInvoice {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterLnInvoiceINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalLnInvoice) Lower(value *LnInvoice) C.RustBuffer {
	return LowerIntoRustBuffer[*LnInvoice](c, value)
}

func (_ FfiConverterOptionalLnInvoice) Write(writer io.Writer, value *LnInvoice) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterLnInvoiceINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalLnInvoice struct{}

func (_ FfiDestroyerOptionalLnInvoice) Destroy(value *LnInvoice) {
	if value != nil {
		FfiDestroyerLnInvoice{}.Destroy(*value)
	}
}

type FfiConverterOptionalLspInformation struct{}

var FfiConverterOptionalLspInformationINSTANCE = FfiConverterOptionalLspInformation{}

func (c FfiConverterOptionalLspInformation) Lift(rb RustBufferI) *LspInformation {
	return LiftFromRustBuffer[*LspInformation](c, rb)
}

func (_ FfiConverterOptionalLspInformation) Read(reader io.Reader) *LspInformation {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterLspInformationINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalLspInformation) Lower(value *LspInformation) C.RustBuffer {
	return LowerIntoRustBuffer[*LspInformation](c, value)
}

func (_ FfiConverterOptionalLspInformation) Write(writer io.Writer, value *LspInformation) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterLspInformationINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalLspInformation struct{}

func (_ FfiDestroyerOptionalLspInformation) Destroy(value *LspInformation) {
	if value != nil {
		FfiDestroyerLspInformation{}.Destroy(*value)
	}
}

type FfiConverterOptionalOpeningFeeParams struct{}

var FfiConverterOptionalOpeningFeeParamsINSTANCE = FfiConverterOptionalOpeningFeeParams{}

func (c FfiConverterOptionalOpeningFeeParams) Lift(rb RustBufferI) *OpeningFeeParams {
	return LiftFromRustBuffer[*OpeningFeeParams](c, rb)
}

func (_ FfiConverterOptionalOpeningFeeParams) Read(reader io.Reader) *OpeningFeeParams {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterOpeningFeeParamsINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalOpeningFeeParams) Lower(value *OpeningFeeParams) C.RustBuffer {
	return LowerIntoRustBuffer[*OpeningFeeParams](c, value)
}

func (_ FfiConverterOptionalOpeningFeeParams) Write(writer io.Writer, value *OpeningFeeParams) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterOpeningFeeParamsINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalOpeningFeeParams struct{}

func (_ FfiDestroyerOptionalOpeningFeeParams) Destroy(value *OpeningFeeParams) {
	if value != nil {
		FfiDestroyerOpeningFeeParams{}.Destroy(*value)
	}
}

type FfiConverterOptionalPayment struct{}

var FfiConverterOptionalPaymentINSTANCE = FfiConverterOptionalPayment{}

func (c FfiConverterOptionalPayment) Lift(rb RustBufferI) *Payment {
	return LiftFromRustBuffer[*Payment](c, rb)
}

func (_ FfiConverterOptionalPayment) Read(reader io.Reader) *Payment {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterPaymentINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalPayment) Lower(value *Payment) C.RustBuffer {
	return LowerIntoRustBuffer[*Payment](c, value)
}

func (_ FfiConverterOptionalPayment) Write(writer io.Writer, value *Payment) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterPaymentINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalPayment struct{}

func (_ FfiDestroyerOptionalPayment) Destroy(value *Payment) {
	if value != nil {
		FfiDestroyerPayment{}.Destroy(*value)
	}
}

type FfiConverterOptionalReverseSwapInfo struct{}

var FfiConverterOptionalReverseSwapInfoINSTANCE = FfiConverterOptionalReverseSwapInfo{}

func (c FfiConverterOptionalReverseSwapInfo) Lift(rb RustBufferI) *ReverseSwapInfo {
	return LiftFromRustBuffer[*ReverseSwapInfo](c, rb)
}

func (_ FfiConverterOptionalReverseSwapInfo) Read(reader io.Reader) *ReverseSwapInfo {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterReverseSwapInfoINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalReverseSwapInfo) Lower(value *ReverseSwapInfo) C.RustBuffer {
	return LowerIntoRustBuffer[*ReverseSwapInfo](c, value)
}

func (_ FfiConverterOptionalReverseSwapInfo) Write(writer io.Writer, value *ReverseSwapInfo) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterReverseSwapInfoINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalReverseSwapInfo struct{}

func (_ FfiDestroyerOptionalReverseSwapInfo) Destroy(value *ReverseSwapInfo) {
	if value != nil {
		FfiDestroyerReverseSwapInfo{}.Destroy(*value)
	}
}

type FfiConverterOptionalSwapInfo struct{}

var FfiConverterOptionalSwapInfoINSTANCE = FfiConverterOptionalSwapInfo{}

func (c FfiConverterOptionalSwapInfo) Lift(rb RustBufferI) *SwapInfo {
	return LiftFromRustBuffer[*SwapInfo](c, rb)
}

func (_ FfiConverterOptionalSwapInfo) Read(reader io.Reader) *SwapInfo {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSwapInfoINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSwapInfo) Lower(value *SwapInfo) C.RustBuffer {
	return LowerIntoRustBuffer[*SwapInfo](c, value)
}

func (_ FfiConverterOptionalSwapInfo) Write(writer io.Writer, value *SwapInfo) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSwapInfoINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSwapInfo struct{}

func (_ FfiDestroyerOptionalSwapInfo) Destroy(value *SwapInfo) {
	if value != nil {
		FfiDestroyerSwapInfo{}.Destroy(*value)
	}
}

type FfiConverterOptionalSymbol struct{}

var FfiConverterOptionalSymbolINSTANCE = FfiConverterOptionalSymbol{}

func (c FfiConverterOptionalSymbol) Lift(rb RustBufferI) *Symbol {
	return LiftFromRustBuffer[*Symbol](c, rb)
}

func (_ FfiConverterOptionalSymbol) Read(reader io.Reader) *Symbol {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSymbolINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSymbol) Lower(value *Symbol) C.RustBuffer {
	return LowerIntoRustBuffer[*Symbol](c, value)
}

func (_ FfiConverterOptionalSymbol) Write(writer io.Writer, value *Symbol) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSymbolINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSymbol struct{}

func (_ FfiDestroyerOptionalSymbol) Destroy(value *Symbol) {
	if value != nil {
		FfiDestroyerSymbol{}.Destroy(*value)
	}
}

type FfiConverterOptionalLevelFilter struct{}

var FfiConverterOptionalLevelFilterINSTANCE = FfiConverterOptionalLevelFilter{}

func (c FfiConverterOptionalLevelFilter) Lift(rb RustBufferI) *LevelFilter {
	return LiftFromRustBuffer[*LevelFilter](c, rb)
}

func (_ FfiConverterOptionalLevelFilter) Read(reader io.Reader) *LevelFilter {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterLevelFilterINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalLevelFilter) Lower(value *LevelFilter) C.RustBuffer {
	return LowerIntoRustBuffer[*LevelFilter](c, value)
}

func (_ FfiConverterOptionalLevelFilter) Write(writer io.Writer, value *LevelFilter) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterLevelFilterINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalLevelFilter struct{}

func (_ FfiDestroyerOptionalLevelFilter) Destroy(value *LevelFilter) {
	if value != nil {
		FfiDestroyerLevelFilter{}.Destroy(*value)
	}
}

type FfiConverterOptionalNodeCredentials struct{}

var FfiConverterOptionalNodeCredentialsINSTANCE = FfiConverterOptionalNodeCredentials{}

func (c FfiConverterOptionalNodeCredentials) Lift(rb RustBufferI) *NodeCredentials {
	return LiftFromRustBuffer[*NodeCredentials](c, rb)
}

func (_ FfiConverterOptionalNodeCredentials) Read(reader io.Reader) *NodeCredentials {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterNodeCredentialsINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalNodeCredentials) Lower(value *NodeCredentials) C.RustBuffer {
	return LowerIntoRustBuffer[*NodeCredentials](c, value)
}

func (_ FfiConverterOptionalNodeCredentials) Write(writer io.Writer, value *NodeCredentials) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterNodeCredentialsINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalNodeCredentials struct{}

func (_ FfiDestroyerOptionalNodeCredentials) Destroy(value *NodeCredentials) {
	if value != nil {
		FfiDestroyerNodeCredentials{}.Destroy(*value)
	}
}

type FfiConverterOptionalSuccessActionProcessed struct{}

var FfiConverterOptionalSuccessActionProcessedINSTANCE = FfiConverterOptionalSuccessActionProcessed{}

func (c FfiConverterOptionalSuccessActionProcessed) Lift(rb RustBufferI) *SuccessActionProcessed {
	return LiftFromRustBuffer[*SuccessActionProcessed](c, rb)
}

func (_ FfiConverterOptionalSuccessActionProcessed) Read(reader io.Reader) *SuccessActionProcessed {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSuccessActionProcessedINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSuccessActionProcessed) Lower(value *SuccessActionProcessed) C.RustBuffer {
	return LowerIntoRustBuffer[*SuccessActionProcessed](c, value)
}

func (_ FfiConverterOptionalSuccessActionProcessed) Write(writer io.Writer, value *SuccessActionProcessed) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSuccessActionProcessedINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSuccessActionProcessed struct{}

func (_ FfiDestroyerOptionalSuccessActionProcessed) Destroy(value *SuccessActionProcessed) {
	if value != nil {
		FfiDestroyerSuccessActionProcessed{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceUint8 struct{}

var FfiConverterOptionalSequenceUint8INSTANCE = FfiConverterOptionalSequenceUint8{}

func (c FfiConverterOptionalSequenceUint8) Lift(rb RustBufferI) *[]uint8 {
	return LiftFromRustBuffer[*[]uint8](c, rb)
}

func (_ FfiConverterOptionalSequenceUint8) Read(reader io.Reader) *[]uint8 {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceUint8INSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceUint8) Lower(value *[]uint8) C.RustBuffer {
	return LowerIntoRustBuffer[*[]uint8](c, value)
}

func (_ FfiConverterOptionalSequenceUint8) Write(writer io.Writer, value *[]uint8) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceUint8INSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceUint8 struct{}

func (_ FfiDestroyerOptionalSequenceUint8) Destroy(value *[]uint8) {
	if value != nil {
		FfiDestroyerSequenceUint8{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceString struct{}

var FfiConverterOptionalSequenceStringINSTANCE = FfiConverterOptionalSequenceString{}

func (c FfiConverterOptionalSequenceString) Lift(rb RustBufferI) *[]string {
	return LiftFromRustBuffer[*[]string](c, rb)
}

func (_ FfiConverterOptionalSequenceString) Read(reader io.Reader) *[]string {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceStringINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceString) Lower(value *[]string) C.RustBuffer {
	return LowerIntoRustBuffer[*[]string](c, value)
}

func (_ FfiConverterOptionalSequenceString) Write(writer io.Writer, value *[]string) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceStringINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceString struct{}

func (_ FfiDestroyerOptionalSequenceString) Destroy(value *[]string) {
	if value != nil {
		FfiDestroyerSequenceString{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceMetadataFilter struct{}

var FfiConverterOptionalSequenceMetadataFilterINSTANCE = FfiConverterOptionalSequenceMetadataFilter{}

func (c FfiConverterOptionalSequenceMetadataFilter) Lift(rb RustBufferI) *[]MetadataFilter {
	return LiftFromRustBuffer[*[]MetadataFilter](c, rb)
}

func (_ FfiConverterOptionalSequenceMetadataFilter) Read(reader io.Reader) *[]MetadataFilter {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceMetadataFilterINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceMetadataFilter) Lower(value *[]MetadataFilter) C.RustBuffer {
	return LowerIntoRustBuffer[*[]MetadataFilter](c, value)
}

func (_ FfiConverterOptionalSequenceMetadataFilter) Write(writer io.Writer, value *[]MetadataFilter) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceMetadataFilterINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceMetadataFilter struct{}

func (_ FfiDestroyerOptionalSequenceMetadataFilter) Destroy(value *[]MetadataFilter) {
	if value != nil {
		FfiDestroyerSequenceMetadataFilter{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceTlvEntry struct{}

var FfiConverterOptionalSequenceTlvEntryINSTANCE = FfiConverterOptionalSequenceTlvEntry{}

func (c FfiConverterOptionalSequenceTlvEntry) Lift(rb RustBufferI) *[]TlvEntry {
	return LiftFromRustBuffer[*[]TlvEntry](c, rb)
}

func (_ FfiConverterOptionalSequenceTlvEntry) Read(reader io.Reader) *[]TlvEntry {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceTlvEntryINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceTlvEntry) Lower(value *[]TlvEntry) C.RustBuffer {
	return LowerIntoRustBuffer[*[]TlvEntry](c, value)
}

func (_ FfiConverterOptionalSequenceTlvEntry) Write(writer io.Writer, value *[]TlvEntry) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceTlvEntryINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceTlvEntry struct{}

func (_ FfiDestroyerOptionalSequenceTlvEntry) Destroy(value *[]TlvEntry) {
	if value != nil {
		FfiDestroyerSequenceTlvEntry{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequencePaymentTypeFilter struct{}

var FfiConverterOptionalSequencePaymentTypeFilterINSTANCE = FfiConverterOptionalSequencePaymentTypeFilter{}

func (c FfiConverterOptionalSequencePaymentTypeFilter) Lift(rb RustBufferI) *[]PaymentTypeFilter {
	return LiftFromRustBuffer[*[]PaymentTypeFilter](c, rb)
}

func (_ FfiConverterOptionalSequencePaymentTypeFilter) Read(reader io.Reader) *[]PaymentTypeFilter {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequencePaymentTypeFilterINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequencePaymentTypeFilter) Lower(value *[]PaymentTypeFilter) C.RustBuffer {
	return LowerIntoRustBuffer[*[]PaymentTypeFilter](c, value)
}

func (_ FfiConverterOptionalSequencePaymentTypeFilter) Write(writer io.Writer, value *[]PaymentTypeFilter) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequencePaymentTypeFilterINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequencePaymentTypeFilter struct{}

func (_ FfiDestroyerOptionalSequencePaymentTypeFilter) Destroy(value *[]PaymentTypeFilter) {
	if value != nil {
		FfiDestroyerSequencePaymentTypeFilter{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceSwapStatus struct{}

var FfiConverterOptionalSequenceSwapStatusINSTANCE = FfiConverterOptionalSequenceSwapStatus{}

func (c FfiConverterOptionalSequenceSwapStatus) Lift(rb RustBufferI) *[]SwapStatus {
	return LiftFromRustBuffer[*[]SwapStatus](c, rb)
}

func (_ FfiConverterOptionalSequenceSwapStatus) Read(reader io.Reader) *[]SwapStatus {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceSwapStatusINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceSwapStatus) Lower(value *[]SwapStatus) C.RustBuffer {
	return LowerIntoRustBuffer[*[]SwapStatus](c, value)
}

func (_ FfiConverterOptionalSequenceSwapStatus) Write(writer io.Writer, value *[]SwapStatus) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceSwapStatusINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceSwapStatus struct{}

func (_ FfiDestroyerOptionalSequenceSwapStatus) Destroy(value *[]SwapStatus) {
	if value != nil {
		FfiDestroyerSequenceSwapStatus{}.Destroy(*value)
	}
}

type FfiConverterSequenceUint8 struct{}

var FfiConverterSequenceUint8INSTANCE = FfiConverterSequenceUint8{}

func (c FfiConverterSequenceUint8) Lift(rb RustBufferI) []uint8 {
	return LiftFromRustBuffer[[]uint8](c, rb)
}

func (c FfiConverterSequenceUint8) Read(reader io.Reader) []uint8 {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]uint8, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterUint8INSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceUint8) Lower(value []uint8) C.RustBuffer {
	return LowerIntoRustBuffer[[]uint8](c, value)
}

func (c FfiConverterSequenceUint8) Write(writer io.Writer, value []uint8) {
	if len(value) > math.MaxInt32 {
		panic("[]uint8 is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterUint8INSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceUint8 struct{}

func (FfiDestroyerSequenceUint8) Destroy(sequence []uint8) {
	for _, value := range sequence {
		FfiDestroyerUint8{}.Destroy(value)
	}
}

type FfiConverterSequenceString struct{}

var FfiConverterSequenceStringINSTANCE = FfiConverterSequenceString{}

func (c FfiConverterSequenceString) Lift(rb RustBufferI) []string {
	return LiftFromRustBuffer[[]string](c, rb)
}

func (c FfiConverterSequenceString) Read(reader io.Reader) []string {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]string, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterStringINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceString) Lower(value []string) C.RustBuffer {
	return LowerIntoRustBuffer[[]string](c, value)
}

func (c FfiConverterSequenceString) Write(writer io.Writer, value []string) {
	if len(value) > math.MaxInt32 {
		panic("[]string is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterStringINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceString struct{}

func (FfiDestroyerSequenceString) Destroy(sequence []string) {
	for _, value := range sequence {
		FfiDestroyerString{}.Destroy(value)
	}
}

type FfiConverterSequenceFiatCurrency struct{}

var FfiConverterSequenceFiatCurrencyINSTANCE = FfiConverterSequenceFiatCurrency{}

func (c FfiConverterSequenceFiatCurrency) Lift(rb RustBufferI) []FiatCurrency {
	return LiftFromRustBuffer[[]FiatCurrency](c, rb)
}

func (c FfiConverterSequenceFiatCurrency) Read(reader io.Reader) []FiatCurrency {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]FiatCurrency, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterFiatCurrencyINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceFiatCurrency) Lower(value []FiatCurrency) C.RustBuffer {
	return LowerIntoRustBuffer[[]FiatCurrency](c, value)
}

func (c FfiConverterSequenceFiatCurrency) Write(writer io.Writer, value []FiatCurrency) {
	if len(value) > math.MaxInt32 {
		panic("[]FiatCurrency is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterFiatCurrencyINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceFiatCurrency struct{}

func (FfiDestroyerSequenceFiatCurrency) Destroy(sequence []FiatCurrency) {
	for _, value := range sequence {
		FfiDestroyerFiatCurrency{}.Destroy(value)
	}
}

type FfiConverterSequenceLocaleOverrides struct{}

var FfiConverterSequenceLocaleOverridesINSTANCE = FfiConverterSequenceLocaleOverrides{}

func (c FfiConverterSequenceLocaleOverrides) Lift(rb RustBufferI) []LocaleOverrides {
	return LiftFromRustBuffer[[]LocaleOverrides](c, rb)
}

func (c FfiConverterSequenceLocaleOverrides) Read(reader io.Reader) []LocaleOverrides {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LocaleOverrides, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterLocaleOverridesINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceLocaleOverrides) Lower(value []LocaleOverrides) C.RustBuffer {
	return LowerIntoRustBuffer[[]LocaleOverrides](c, value)
}

func (c FfiConverterSequenceLocaleOverrides) Write(writer io.Writer, value []LocaleOverrides) {
	if len(value) > math.MaxInt32 {
		panic("[]LocaleOverrides is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterLocaleOverridesINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceLocaleOverrides struct{}

func (FfiDestroyerSequenceLocaleOverrides) Destroy(sequence []LocaleOverrides) {
	for _, value := range sequence {
		FfiDestroyerLocaleOverrides{}.Destroy(value)
	}
}

type FfiConverterSequenceLocalizedName struct{}

var FfiConverterSequenceLocalizedNameINSTANCE = FfiConverterSequenceLocalizedName{}

func (c FfiConverterSequenceLocalizedName) Lift(rb RustBufferI) []LocalizedName {
	return LiftFromRustBuffer[[]LocalizedName](c, rb)
}

func (c FfiConverterSequenceLocalizedName) Read(reader io.Reader) []LocalizedName {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LocalizedName, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterLocalizedNameINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceLocalizedName) Lower(value []LocalizedName) C.RustBuffer {
	return LowerIntoRustBuffer[[]LocalizedName](c, value)
}

func (c FfiConverterSequenceLocalizedName) Write(writer io.Writer, value []LocalizedName) {
	if len(value) > math.MaxInt32 {
		panic("[]LocalizedName is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterLocalizedNameINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceLocalizedName struct{}

func (FfiDestroyerSequenceLocalizedName) Destroy(sequence []LocalizedName) {
	for _, value := range sequence {
		FfiDestroyerLocalizedName{}.Destroy(value)
	}
}

type FfiConverterSequenceLspInformation struct{}

var FfiConverterSequenceLspInformationINSTANCE = FfiConverterSequenceLspInformation{}

func (c FfiConverterSequenceLspInformation) Lift(rb RustBufferI) []LspInformation {
	return LiftFromRustBuffer[[]LspInformation](c, rb)
}

func (c FfiConverterSequenceLspInformation) Read(reader io.Reader) []LspInformation {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LspInformation, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterLspInformationINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceLspInformation) Lower(value []LspInformation) C.RustBuffer {
	return LowerIntoRustBuffer[[]LspInformation](c, value)
}

func (c FfiConverterSequenceLspInformation) Write(writer io.Writer, value []LspInformation) {
	if len(value) > math.MaxInt32 {
		panic("[]LspInformation is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterLspInformationINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceLspInformation struct{}

func (FfiDestroyerSequenceLspInformation) Destroy(sequence []LspInformation) {
	for _, value := range sequence {
		FfiDestroyerLspInformation{}.Destroy(value)
	}
}

type FfiConverterSequenceMetadataFilter struct{}

var FfiConverterSequenceMetadataFilterINSTANCE = FfiConverterSequenceMetadataFilter{}

func (c FfiConverterSequenceMetadataFilter) Lift(rb RustBufferI) []MetadataFilter {
	return LiftFromRustBuffer[[]MetadataFilter](c, rb)
}

func (c FfiConverterSequenceMetadataFilter) Read(reader io.Reader) []MetadataFilter {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]MetadataFilter, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterMetadataFilterINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceMetadataFilter) Lower(value []MetadataFilter) C.RustBuffer {
	return LowerIntoRustBuffer[[]MetadataFilter](c, value)
}

func (c FfiConverterSequenceMetadataFilter) Write(writer io.Writer, value []MetadataFilter) {
	if len(value) > math.MaxInt32 {
		panic("[]MetadataFilter is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterMetadataFilterINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceMetadataFilter struct{}

func (FfiDestroyerSequenceMetadataFilter) Destroy(sequence []MetadataFilter) {
	for _, value := range sequence {
		FfiDestroyerMetadataFilter{}.Destroy(value)
	}
}

type FfiConverterSequenceOpeningFeeParams struct{}

var FfiConverterSequenceOpeningFeeParamsINSTANCE = FfiConverterSequenceOpeningFeeParams{}

func (c FfiConverterSequenceOpeningFeeParams) Lift(rb RustBufferI) []OpeningFeeParams {
	return LiftFromRustBuffer[[]OpeningFeeParams](c, rb)
}

func (c FfiConverterSequenceOpeningFeeParams) Read(reader io.Reader) []OpeningFeeParams {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]OpeningFeeParams, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterOpeningFeeParamsINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceOpeningFeeParams) Lower(value []OpeningFeeParams) C.RustBuffer {
	return LowerIntoRustBuffer[[]OpeningFeeParams](c, value)
}

func (c FfiConverterSequenceOpeningFeeParams) Write(writer io.Writer, value []OpeningFeeParams) {
	if len(value) > math.MaxInt32 {
		panic("[]OpeningFeeParams is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterOpeningFeeParamsINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceOpeningFeeParams struct{}

func (FfiDestroyerSequenceOpeningFeeParams) Destroy(sequence []OpeningFeeParams) {
	for _, value := range sequence {
		FfiDestroyerOpeningFeeParams{}.Destroy(value)
	}
}

type FfiConverterSequencePayment struct{}

var FfiConverterSequencePaymentINSTANCE = FfiConverterSequencePayment{}

func (c FfiConverterSequencePayment) Lift(rb RustBufferI) []Payment {
	return LiftFromRustBuffer[[]Payment](c, rb)
}

func (c FfiConverterSequencePayment) Read(reader io.Reader) []Payment {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]Payment, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterPaymentINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequencePayment) Lower(value []Payment) C.RustBuffer {
	return LowerIntoRustBuffer[[]Payment](c, value)
}

func (c FfiConverterSequencePayment) Write(writer io.Writer, value []Payment) {
	if len(value) > math.MaxInt32 {
		panic("[]Payment is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterPaymentINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequencePayment struct{}

func (FfiDestroyerSequencePayment) Destroy(sequence []Payment) {
	for _, value := range sequence {
		FfiDestroyerPayment{}.Destroy(value)
	}
}

type FfiConverterSequenceRate struct{}

var FfiConverterSequenceRateINSTANCE = FfiConverterSequenceRate{}

func (c FfiConverterSequenceRate) Lift(rb RustBufferI) []Rate {
	return LiftFromRustBuffer[[]Rate](c, rb)
}

func (c FfiConverterSequenceRate) Read(reader io.Reader) []Rate {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]Rate, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterRateINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceRate) Lower(value []Rate) C.RustBuffer {
	return LowerIntoRustBuffer[[]Rate](c, value)
}

func (c FfiConverterSequenceRate) Write(writer io.Writer, value []Rate) {
	if len(value) > math.MaxInt32 {
		panic("[]Rate is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterRateINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceRate struct{}

func (FfiDestroyerSequenceRate) Destroy(sequence []Rate) {
	for _, value := range sequence {
		FfiDestroyerRate{}.Destroy(value)
	}
}

type FfiConverterSequenceReverseSwapInfo struct{}

var FfiConverterSequenceReverseSwapInfoINSTANCE = FfiConverterSequenceReverseSwapInfo{}

func (c FfiConverterSequenceReverseSwapInfo) Lift(rb RustBufferI) []ReverseSwapInfo {
	return LiftFromRustBuffer[[]ReverseSwapInfo](c, rb)
}

func (c FfiConverterSequenceReverseSwapInfo) Read(reader io.Reader) []ReverseSwapInfo {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]ReverseSwapInfo, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterReverseSwapInfoINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceReverseSwapInfo) Lower(value []ReverseSwapInfo) C.RustBuffer {
	return LowerIntoRustBuffer[[]ReverseSwapInfo](c, value)
}

func (c FfiConverterSequenceReverseSwapInfo) Write(writer io.Writer, value []ReverseSwapInfo) {
	if len(value) > math.MaxInt32 {
		panic("[]ReverseSwapInfo is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterReverseSwapInfoINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceReverseSwapInfo struct{}

func (FfiDestroyerSequenceReverseSwapInfo) Destroy(sequence []ReverseSwapInfo) {
	for _, value := range sequence {
		FfiDestroyerReverseSwapInfo{}.Destroy(value)
	}
}

type FfiConverterSequenceRouteHint struct{}

var FfiConverterSequenceRouteHintINSTANCE = FfiConverterSequenceRouteHint{}

func (c FfiConverterSequenceRouteHint) Lift(rb RustBufferI) []RouteHint {
	return LiftFromRustBuffer[[]RouteHint](c, rb)
}

func (c FfiConverterSequenceRouteHint) Read(reader io.Reader) []RouteHint {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]RouteHint, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterRouteHintINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceRouteHint) Lower(value []RouteHint) C.RustBuffer {
	return LowerIntoRustBuffer[[]RouteHint](c, value)
}

func (c FfiConverterSequenceRouteHint) Write(writer io.Writer, value []RouteHint) {
	if len(value) > math.MaxInt32 {
		panic("[]RouteHint is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterRouteHintINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceRouteHint struct{}

func (FfiDestroyerSequenceRouteHint) Destroy(sequence []RouteHint) {
	for _, value := range sequence {
		FfiDestroyerRouteHint{}.Destroy(value)
	}
}

type FfiConverterSequenceRouteHintHop struct{}

var FfiConverterSequenceRouteHintHopINSTANCE = FfiConverterSequenceRouteHintHop{}

func (c FfiConverterSequenceRouteHintHop) Lift(rb RustBufferI) []RouteHintHop {
	return LiftFromRustBuffer[[]RouteHintHop](c, rb)
}

func (c FfiConverterSequenceRouteHintHop) Read(reader io.Reader) []RouteHintHop {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]RouteHintHop, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterRouteHintHopINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceRouteHintHop) Lower(value []RouteHintHop) C.RustBuffer {
	return LowerIntoRustBuffer[[]RouteHintHop](c, value)
}

func (c FfiConverterSequenceRouteHintHop) Write(writer io.Writer, value []RouteHintHop) {
	if len(value) > math.MaxInt32 {
		panic("[]RouteHintHop is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterRouteHintHopINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceRouteHintHop struct{}

func (FfiDestroyerSequenceRouteHintHop) Destroy(sequence []RouteHintHop) {
	for _, value := range sequence {
		FfiDestroyerRouteHintHop{}.Destroy(value)
	}
}

type FfiConverterSequenceSwapInfo struct{}

var FfiConverterSequenceSwapInfoINSTANCE = FfiConverterSequenceSwapInfo{}

func (c FfiConverterSequenceSwapInfo) Lift(rb RustBufferI) []SwapInfo {
	return LiftFromRustBuffer[[]SwapInfo](c, rb)
}

func (c FfiConverterSequenceSwapInfo) Read(reader io.Reader) []SwapInfo {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]SwapInfo, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterSwapInfoINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceSwapInfo) Lower(value []SwapInfo) C.RustBuffer {
	return LowerIntoRustBuffer[[]SwapInfo](c, value)
}

func (c FfiConverterSequenceSwapInfo) Write(writer io.Writer, value []SwapInfo) {
	if len(value) > math.MaxInt32 {
		panic("[]SwapInfo is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterSwapInfoINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceSwapInfo struct{}

func (FfiDestroyerSequenceSwapInfo) Destroy(sequence []SwapInfo) {
	for _, value := range sequence {
		FfiDestroyerSwapInfo{}.Destroy(value)
	}
}

type FfiConverterSequenceTlvEntry struct{}

var FfiConverterSequenceTlvEntryINSTANCE = FfiConverterSequenceTlvEntry{}

func (c FfiConverterSequenceTlvEntry) Lift(rb RustBufferI) []TlvEntry {
	return LiftFromRustBuffer[[]TlvEntry](c, rb)
}

func (c FfiConverterSequenceTlvEntry) Read(reader io.Reader) []TlvEntry {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]TlvEntry, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTlvEntryINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTlvEntry) Lower(value []TlvEntry) C.RustBuffer {
	return LowerIntoRustBuffer[[]TlvEntry](c, value)
}

func (c FfiConverterSequenceTlvEntry) Write(writer io.Writer, value []TlvEntry) {
	if len(value) > math.MaxInt32 {
		panic("[]TlvEntry is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTlvEntryINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTlvEntry struct{}

func (FfiDestroyerSequenceTlvEntry) Destroy(sequence []TlvEntry) {
	for _, value := range sequence {
		FfiDestroyerTlvEntry{}.Destroy(value)
	}
}

type FfiConverterSequenceUnspentTransactionOutput struct{}

var FfiConverterSequenceUnspentTransactionOutputINSTANCE = FfiConverterSequenceUnspentTransactionOutput{}

func (c FfiConverterSequenceUnspentTransactionOutput) Lift(rb RustBufferI) []UnspentTransactionOutput {
	return LiftFromRustBuffer[[]UnspentTransactionOutput](c, rb)
}

func (c FfiConverterSequenceUnspentTransactionOutput) Read(reader io.Reader) []UnspentTransactionOutput {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]UnspentTransactionOutput, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterUnspentTransactionOutputINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceUnspentTransactionOutput) Lower(value []UnspentTransactionOutput) C.RustBuffer {
	return LowerIntoRustBuffer[[]UnspentTransactionOutput](c, value)
}

func (c FfiConverterSequenceUnspentTransactionOutput) Write(writer io.Writer, value []UnspentTransactionOutput) {
	if len(value) > math.MaxInt32 {
		panic("[]UnspentTransactionOutput is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterUnspentTransactionOutputINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceUnspentTransactionOutput struct{}

func (FfiDestroyerSequenceUnspentTransactionOutput) Destroy(sequence []UnspentTransactionOutput) {
	for _, value := range sequence {
		FfiDestroyerUnspentTransactionOutput{}.Destroy(value)
	}
}

type FfiConverterSequencePaymentTypeFilter struct{}

var FfiConverterSequencePaymentTypeFilterINSTANCE = FfiConverterSequencePaymentTypeFilter{}

func (c FfiConverterSequencePaymentTypeFilter) Lift(rb RustBufferI) []PaymentTypeFilter {
	return LiftFromRustBuffer[[]PaymentTypeFilter](c, rb)
}

func (c FfiConverterSequencePaymentTypeFilter) Read(reader io.Reader) []PaymentTypeFilter {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]PaymentTypeFilter, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterPaymentTypeFilterINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequencePaymentTypeFilter) Lower(value []PaymentTypeFilter) C.RustBuffer {
	return LowerIntoRustBuffer[[]PaymentTypeFilter](c, value)
}

func (c FfiConverterSequencePaymentTypeFilter) Write(writer io.Writer, value []PaymentTypeFilter) {
	if len(value) > math.MaxInt32 {
		panic("[]PaymentTypeFilter is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterPaymentTypeFilterINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequencePaymentTypeFilter struct{}

func (FfiDestroyerSequencePaymentTypeFilter) Destroy(sequence []PaymentTypeFilter) {
	for _, value := range sequence {
		FfiDestroyerPaymentTypeFilter{}.Destroy(value)
	}
}

type FfiConverterSequenceSwapStatus struct{}

var FfiConverterSequenceSwapStatusINSTANCE = FfiConverterSequenceSwapStatus{}

func (c FfiConverterSequenceSwapStatus) Lift(rb RustBufferI) []SwapStatus {
	return LiftFromRustBuffer[[]SwapStatus](c, rb)
}

func (c FfiConverterSequenceSwapStatus) Read(reader io.Reader) []SwapStatus {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]SwapStatus, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterSwapStatusINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceSwapStatus) Lower(value []SwapStatus) C.RustBuffer {
	return LowerIntoRustBuffer[[]SwapStatus](c, value)
}

func (c FfiConverterSequenceSwapStatus) Write(writer io.Writer, value []SwapStatus) {
	if len(value) > math.MaxInt32 {
		panic("[]SwapStatus is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterSwapStatusINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceSwapStatus struct{}

func (FfiDestroyerSequenceSwapStatus) Destroy(sequence []SwapStatus) {
	for _, value := range sequence {
		FfiDestroyerSwapStatus{}.Destroy(value)
	}
}

func Connect(req ConnectRequest, listener EventListener) (*BlockingBreezServices, *ConnectError) {
	_uniffiRV, _uniffiErr := rustCallWithError[ConnectError](FfiConverterConnectError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_breez_sdk_bindings_fn_func_connect(FfiConverterConnectRequestINSTANCE.Lower(req), FfiConverterCallbackInterfaceEventListenerINSTANCE.Lower(listener), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *BlockingBreezServices
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterBlockingBreezServicesINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func DefaultConfig(envType EnvironmentType, apiKey string, nodeConfig NodeConfig) Config {
	return FfiConverterConfigINSTANCE.Lift(rustCall(func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_func_default_config(FfiConverterEnvironmentTypeINSTANCE.Lower(envType), FfiConverterStringINSTANCE.Lower(apiKey), FfiConverterNodeConfigINSTANCE.Lower(nodeConfig), _uniffiStatus),
		}
	}))
}

func MnemonicToSeed(phrase string) ([]uint8, *SdkError) {
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_func_mnemonic_to_seed(FfiConverterStringINSTANCE.Lower(phrase), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []uint8
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceUint8INSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func ParseInput(s string) (InputType, *SdkError) {
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_func_parse_input(FfiConverterStringINSTANCE.Lower(s), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue InputType
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterInputTypeINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func ParseInvoice(invoice string) (LnInvoice, *SdkError) {
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_func_parse_invoice(FfiConverterStringINSTANCE.Lower(invoice), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnInvoice
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterLnInvoiceINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func ServiceHealthCheck(apiKey string) (ServiceHealthCheckResponse, *SdkError) {
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_func_service_health_check(FfiConverterStringINSTANCE.Lower(apiKey), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ServiceHealthCheckResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterServiceHealthCheckResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func SetLogStream(logStream LogStream, filterLevel *LevelFilter) *SdkError {
	_, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_func_set_log_stream(FfiConverterCallbackInterfaceLogStreamINSTANCE.Lower(logStream), FfiConverterOptionalLevelFilterINSTANCE.Lower(filterLevel), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func StaticBackup(req StaticBackupRequest) (StaticBackupResponse, *SdkError) {
	_uniffiRV, _uniffiErr := rustCallWithError[SdkError](FfiConverterSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return GoRustBuffer{
			inner: C.uniffi_breez_sdk_bindings_fn_func_static_backup(FfiConverterStaticBackupRequestINSTANCE.Lower(req), _uniffiStatus),
		}
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue StaticBackupResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterStaticBackupResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}
