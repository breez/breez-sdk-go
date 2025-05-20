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

type RustBuffer = C.RustBuffer

type RustBufferI interface {
	AsReader() *bytes.Reader
	Free()
	ToGoBytes() []byte
	Data() unsafe.Pointer
	Len() int
	Capacity() int
}

func RustBufferFromExternal(b RustBufferI) RustBuffer {
	return RustBuffer{
		capacity: C.int(b.Capacity()),
		len:      C.int(b.Len()),
		data:     (*C.uchar)(b.Data()),
	}
}

func (cb RustBuffer) Capacity() int {
	return int(cb.capacity)
}

func (cb RustBuffer) Len() int {
	return int(cb.len)
}

func (cb RustBuffer) Data() unsafe.Pointer {
	return unsafe.Pointer(cb.data)
}

func (cb RustBuffer) AsReader() *bytes.Reader {
	b := unsafe.Slice((*byte)(cb.data), C.int(cb.len))
	return bytes.NewReader(b)
}

func (cb RustBuffer) Free() {
	rustCall(func(status *C.RustCallStatus) bool {
		C.ffi_breez_sdk_bindings_rustbuffer_free(cb, status)
		return false
	})
}

func (cb RustBuffer) ToGoBytes() []byte {
	return C.GoBytes(unsafe.Pointer(cb.data), C.int(cb.len))
}

func stringToRustBuffer(str string) RustBuffer {
	return bytesToRustBuffer([]byte(str))
}

func bytesToRustBuffer(b []byte) RustBuffer {
	if len(b) == 0 {
		return RustBuffer{}
	}
	// We can pass the pointer along here, as it is pinned
	// for the duration of this call
	foreign := C.ForeignBytes{
		len:  C.int(len(b)),
		data: (*C.uchar)(unsafe.Pointer(&b[0])),
	}

	return rustCall(func(status *C.RustCallStatus) RustBuffer {
		return C.ffi_breez_sdk_bindings_rustbuffer_from_bytes(foreign, status)
	})
}

type BufLifter[GoType any] interface {
	Lift(value RustBufferI) GoType
}

type BufLowerer[GoType any] interface {
	Lower(value GoType) RustBuffer
}

type FfiConverter[GoType any, FfiType any] interface {
	Lift(value FfiType) GoType
	Lower(value GoType) FfiType
}

type BufReader[GoType any] interface {
	Read(reader io.Reader) GoType
}

type BufWriter[GoType any] interface {
	Write(writer io.Writer, value GoType)
}

type FfiRustBufConverter[GoType any, FfiType any] interface {
	FfiConverter[GoType, FfiType]
	BufReader[GoType]
}

func LowerIntoRustBuffer[GoType any](bufWriter BufWriter[GoType], value GoType) RustBuffer {
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

func rustCallWithError[U any](converter BufLifter[error], callback func(*C.RustCallStatus) U) (U, error) {
	var status C.RustCallStatus
	returnValue := callback(&status)
	err := checkCallStatus(converter, status)

	return returnValue, err
}

func checkCallStatus(converter BufLifter[error], status C.RustCallStatus) error {
	switch status.code {
	case 0:
		return nil
	case 1:
		return converter.Lift(status.errorBuf)
	case 2:
		// when the rust code sees a panic, it tries to construct a rustbuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(status.errorBuf)))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		return fmt.Errorf("unknown status code: %d", status.code)
	}
}

func checkCallStatusUnknown(status C.RustCallStatus) error {
	switch status.code {
	case 0:
		return nil
	case 1:
		panic(fmt.Errorf("function not returning an error returned an error"))
	case 2:
		// when the rust code sees a panic, it tries to construct a rustbuffer
		// with the message.  but if that code panics, then it just sends back
		// an empty buffer.
		if status.errorBuf.len > 0 {
			panic(fmt.Errorf("%s", FfiConverterStringINSTANCE.Lift(status.errorBuf)))
		} else {
			panic(fmt.Errorf("Rust panicked while handling Rust panic"))
		}
	default:
		return fmt.Errorf("unknown status code: %d", status.code)
	}
}

func rustCall[U any](callback func(*C.RustCallStatus) U) U {
	returnValue, err := rustCallWithError(nil, callback)
	if err != nil {
		panic(err)
	}
	return returnValue
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

	(&FfiConverterCallbackInterfaceEventListener{}).register()
	(&FfiConverterCallbackInterfaceLogStream{}).register()
	uniffiCheckChecksums()
}

func uniffiCheckChecksums() {
	// Get the bindings contract version from our ComponentInterface
	bindingsContractVersion := 24
	// Get the scaffolding contract version by calling the into the dylib
	scaffoldingContractVersion := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint32_t {
		return C.ffi_breez_sdk_bindings_uniffi_contract_version(uniffiStatus)
	})
	if bindingsContractVersion != int(scaffoldingContractVersion) {
		// If this happens try cleaning and rebuilding your project
		panic("breez_sdk: UniFFI contract version mismatch")
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_connect(uniffiStatus)
		})
		if checksum != 3084 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_connect: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_default_config(uniffiStatus)
		})
		if checksum != 55271 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_default_config: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_mnemonic_to_seed(uniffiStatus)
		})
		if checksum != 18103 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_mnemonic_to_seed: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_parse_input(uniffiStatus)
		})
		if checksum != 25972 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_parse_input: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_parse_invoice(uniffiStatus)
		})
		if checksum != 56304 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_parse_invoice: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_service_health_check(uniffiStatus)
		})
		if checksum != 41271 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_service_health_check: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_set_log_stream(uniffiStatus)
		})
		if checksum != 9785 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_set_log_stream: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_func_static_backup(uniffiStatus)
		})
		if checksum != 8435 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_func_static_backup: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_backup(uniffiStatus)
		})
		if checksum != 22471 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_backup: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_backup_status(uniffiStatus)
		})
		if checksum != 15762 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_backup_status: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_buy_bitcoin(uniffiStatus)
		})
		if checksum != 44754 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_buy_bitcoin: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_check_message(uniffiStatus)
		})
		if checksum != 58323 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_check_message: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_claim_reverse_swap(uniffiStatus)
		})
		if checksum != 13553 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_claim_reverse_swap: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_close_lsp_channels(uniffiStatus)
		})
		if checksum != 21012 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_close_lsp_channels: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_configure_node(uniffiStatus)
		})
		if checksum != 59468 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_configure_node: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_connect_lsp(uniffiStatus)
		})
		if checksum != 29943 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_connect_lsp: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_disconnect(uniffiStatus)
		})
		if checksum != 10013 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_disconnect: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_execute_dev_command(uniffiStatus)
		})
		if checksum != 33535 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_execute_dev_command: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_fiat_rates(uniffiStatus)
		})
		if checksum != 5305 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_fiat_rates: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_lsp_info(uniffiStatus)
		})
		if checksum != 10652 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_lsp_info: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_reverse_swap_fees(uniffiStatus)
		})
		if checksum != 15380 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_fetch_reverse_swap_fees: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_generate_diagnostic_data(uniffiStatus)
		})
		if checksum != 16670 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_generate_diagnostic_data: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_in_progress_onchain_payments(uniffiStatus)
		})
		if checksum != 34385 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_in_progress_onchain_payments: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_in_progress_swap(uniffiStatus)
		})
		if checksum != 48764 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_in_progress_swap: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_fiat_currencies(uniffiStatus)
		})
		if checksum != 33187 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_fiat_currencies: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_lsps(uniffiStatus)
		})
		if checksum != 15408 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_lsps: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_payments(uniffiStatus)
		})
		if checksum != 50574 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_payments: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_refundables(uniffiStatus)
		})
		if checksum != 37329 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_refundables: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_swaps(uniffiStatus)
		})
		if checksum != 30887 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_list_swaps: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lnurl_auth(uniffiStatus)
		})
		if checksum != 36311 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lnurl_auth: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lsp_id(uniffiStatus)
		})
		if checksum != 27141 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lsp_id: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lsp_info(uniffiStatus)
		})
		if checksum != 54374 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_lsp_info: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_node_credentials(uniffiStatus)
		})
		if checksum != 1357 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_node_credentials: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_node_info(uniffiStatus)
		})
		if checksum != 43663 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_node_info: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_onchain_payment_limits(uniffiStatus)
		})
		if checksum != 39466 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_onchain_payment_limits: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_open_channel_fee(uniffiStatus)
		})
		if checksum != 53480 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_open_channel_fee: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_pay_lnurl(uniffiStatus)
		})
		if checksum != 13299 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_pay_lnurl: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_pay_onchain(uniffiStatus)
		})
		if checksum != 38697 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_pay_onchain: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_payment_by_hash(uniffiStatus)
		})
		if checksum != 1074 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_payment_by_hash: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_onchain_payment(uniffiStatus)
		})
		if checksum != 52417 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_onchain_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_redeem_onchain_funds(uniffiStatus)
		})
		if checksum != 57459 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_redeem_onchain_funds: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_refund(uniffiStatus)
		})
		if checksum != 47982 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_prepare_refund: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_receive_onchain(uniffiStatus)
		})
		if checksum != 54453 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_receive_onchain: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_receive_payment(uniffiStatus)
		})
		if checksum != 37604 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_receive_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_recommended_fees(uniffiStatus)
		})
		if checksum != 18957 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_recommended_fees: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_redeem_onchain_funds(uniffiStatus)
		})
		if checksum != 37392 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_redeem_onchain_funds: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_redeem_swap(uniffiStatus)
		})
		if checksum != 3224 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_redeem_swap: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_refund(uniffiStatus)
		})
		if checksum != 21319 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_refund: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_register_webhook(uniffiStatus)
		})
		if checksum != 19609 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_register_webhook: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_report_issue(uniffiStatus)
		})
		if checksum != 29646 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_report_issue: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_rescan_swaps(uniffiStatus)
		})
		if checksum != 37007 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_rescan_swaps: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_send_payment(uniffiStatus)
		})
		if checksum != 55910 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_send_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_send_spontaneous_payment(uniffiStatus)
		})
		if checksum != 60758 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_send_spontaneous_payment: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_set_payment_metadata(uniffiStatus)
		})
		if checksum != 49297 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_set_payment_metadata: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_sign_message(uniffiStatus)
		})
		if checksum != 16285 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_sign_message: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_sync(uniffiStatus)
		})
		if checksum != 1413 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_sync: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_unregister_webhook(uniffiStatus)
		})
		if checksum != 31988 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_unregister_webhook: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_withdraw_lnurl(uniffiStatus)
		})
		if checksum != 32871 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_blockingbreezservices_withdraw_lnurl: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_eventlistener_on_event(uniffiStatus)
		})
		if checksum != 44010 {
			// If this happens try cleaning and rebuilding your project
			panic("breez_sdk: uniffi_breez_sdk_bindings_checksum_method_eventlistener_on_event: UniFFI API checksum mismatch")
		}
	}
	{
		checksum := rustCall(func(uniffiStatus *C.RustCallStatus) C.uint16_t {
			return C.uniffi_breez_sdk_bindings_checksum_method_logstream_log(uniffiStatus)
		})
		if checksum != 62103 {
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

func (FfiConverterString) Lower(value string) RustBuffer {
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
	pointer      unsafe.Pointer
	callCounter  atomic.Int64
	freeFunction func(unsafe.Pointer, *C.RustCallStatus)
	destroyed    atomic.Bool
}

func newFfiObject(pointer unsafe.Pointer, freeFunction func(unsafe.Pointer, *C.RustCallStatus)) FfiObject {
	return FfiObject{
		pointer:      pointer,
		freeFunction: freeFunction,
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

	return ffiObject.pointer
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

type BlockingBreezServices struct {
	ffiObject FfiObject
}

func (_self *BlockingBreezServices) Backup() error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_backup(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) BackupStatus() (BackupStatus, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_backup_status(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue BackupStatus
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeBackupStatusINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) BuyBitcoin(req BuyBitcoinRequest) (BuyBitcoinResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeReceiveOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_buy_bitcoin(
			_pointer, FfiConverterTypeBuyBitcoinRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue BuyBitcoinResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeBuyBitcoinResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) CheckMessage(req CheckMessageRequest) (CheckMessageResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_check_message(
			_pointer, FfiConverterTypeCheckMessageRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue CheckMessageResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeCheckMessageResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ClaimReverseSwap(lockupAddress string) error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_claim_reverse_swap(
			_pointer, FfiConverterStringINSTANCE.Lower(lockupAddress), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) CloseLspChannels() error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_close_lsp_channels(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) ConfigureNode(req ConfigureNodeRequest) error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_configure_node(
			_pointer, FfiConverterTypeConfigureNodeRequestINSTANCE.Lower(req), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) ConnectLsp(lspId string) error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_connect_lsp(
			_pointer, FfiConverterStringINSTANCE.Lower(lspId), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) Disconnect() error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_disconnect(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) ExecuteDevCommand(command string) (string, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_execute_dev_command(
			_pointer, FfiConverterStringINSTANCE.Lower(command), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterStringINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) FetchFiatRates() ([]Rate, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_fetch_fiat_rates(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []Rate
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceTypeRateINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) FetchLspInfo(lspId string) (*LspInformation, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_fetch_lsp_info(
			_pointer, FfiConverterStringINSTANCE.Lower(lspId), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *LspInformation
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalTypeLspInformationINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) FetchReverseSwapFees(req ReverseSwapFeesRequest) (ReverseSwapPairInfo, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_fetch_reverse_swap_fees(
			_pointer, FfiConverterTypeReverseSwapFeesRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ReverseSwapPairInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeReverseSwapPairInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) GenerateDiagnosticData() (string, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_generate_diagnostic_data(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterStringINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) InProgressOnchainPayments() ([]ReverseSwapInfo, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_in_progress_onchain_payments(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []ReverseSwapInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceTypeReverseSwapInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) InProgressSwap() (*SwapInfo, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_in_progress_swap(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *SwapInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalTypeSwapInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ListFiatCurrencies() ([]FiatCurrency, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_list_fiat_currencies(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []FiatCurrency
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceTypeFiatCurrencyINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ListLsps() ([]LspInformation, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_list_lsps(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []LspInformation
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceTypeLspInformationINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ListPayments(req ListPaymentsRequest) ([]Payment, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_list_payments(
			_pointer, FfiConverterTypeListPaymentsRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []Payment
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceTypePaymentINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ListRefundables() ([]SwapInfo, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_list_refundables(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []SwapInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceTypeSwapInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ListSwaps(req ListSwapsRequest) ([]SwapInfo, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_list_swaps(
			_pointer, FfiConverterTypeListSwapsRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []SwapInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceTypeSwapInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) LnurlAuth(reqData LnUrlAuthRequestData) (LnUrlCallbackStatus, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeLnUrlAuthError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_lnurl_auth(
			_pointer, FfiConverterTypeLnUrlAuthRequestDataINSTANCE.Lower(reqData), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlCallbackStatus
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeLnUrlCallbackStatusINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) LspId() (*string, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_lsp_id(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *string
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalStringINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) LspInfo() (LspInformation, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_lsp_info(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LspInformation
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeLspInformationINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) NodeCredentials() (*NodeCredentials, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_node_credentials(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *NodeCredentials
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalTypeNodeCredentialsINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) NodeInfo() (NodeState, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_node_info(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue NodeState
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeNodeStateINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) OnchainPaymentLimits() (OnchainPaymentLimitsResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_onchain_payment_limits(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue OnchainPaymentLimitsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeOnchainPaymentLimitsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) OpenChannelFee(req OpenChannelFeeRequest) (OpenChannelFeeResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_open_channel_fee(
			_pointer, FfiConverterTypeOpenChannelFeeRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue OpenChannelFeeResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeOpenChannelFeeResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PayLnurl(req LnUrlPayRequest) (LnUrlPayResult, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeLnUrlPayError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_pay_lnurl(
			_pointer, FfiConverterTypeLnUrlPayRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlPayResult
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeLnUrlPayResultINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PayOnchain(req PayOnchainRequest) (PayOnchainResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSendOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_pay_onchain(
			_pointer, FfiConverterTypePayOnchainRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PayOnchainResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePayOnchainResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PaymentByHash(hash string) (*Payment, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_payment_by_hash(
			_pointer, FfiConverterStringINSTANCE.Lower(hash), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *Payment
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterOptionalTypePaymentINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PrepareOnchainPayment(req PrepareOnchainPaymentRequest) (PrepareOnchainPaymentResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSendOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_prepare_onchain_payment(
			_pointer, FfiConverterTypePrepareOnchainPaymentRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareOnchainPaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePrepareOnchainPaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PrepareRedeemOnchainFunds(req PrepareRedeemOnchainFundsRequest) (PrepareRedeemOnchainFundsResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeRedeemOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_prepare_redeem_onchain_funds(
			_pointer, FfiConverterTypePrepareRedeemOnchainFundsRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareRedeemOnchainFundsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePrepareRedeemOnchainFundsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) PrepareRefund(req PrepareRefundRequest) (PrepareRefundResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_prepare_refund(
			_pointer, FfiConverterTypePrepareRefundRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue PrepareRefundResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypePrepareRefundResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ReceiveOnchain(req ReceiveOnchainRequest) (SwapInfo, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeReceiveOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_receive_onchain(
			_pointer, FfiConverterTypeReceiveOnchainRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SwapInfo
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeSwapInfoINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) ReceivePayment(req ReceivePaymentRequest) (ReceivePaymentResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeReceivePaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_receive_payment(
			_pointer, FfiConverterTypeReceivePaymentRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ReceivePaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeReceivePaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) RecommendedFees() (RecommendedFees, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_recommended_fees(
			_pointer, _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue RecommendedFees
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeRecommendedFeesINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) RedeemOnchainFunds(req RedeemOnchainFundsRequest) (RedeemOnchainFundsResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeRedeemOnchainError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_redeem_onchain_funds(
			_pointer, FfiConverterTypeRedeemOnchainFundsRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue RedeemOnchainFundsResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeRedeemOnchainFundsResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) RedeemSwap(swapAddress string) error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_redeem_swap(
			_pointer, FfiConverterStringINSTANCE.Lower(swapAddress), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) Refund(req RefundRequest) (RefundResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_refund(
			_pointer, FfiConverterTypeRefundRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue RefundResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeRefundResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) RegisterWebhook(webhookUrl string) error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_register_webhook(
			_pointer, FfiConverterStringINSTANCE.Lower(webhookUrl), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) ReportIssue(req ReportIssueRequest) error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_report_issue(
			_pointer, FfiConverterTypeReportIssueRequestINSTANCE.Lower(req), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) RescanSwaps() error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_rescan_swaps(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) SendPayment(req SendPaymentRequest) (SendPaymentResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSendPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_send_payment(
			_pointer, FfiConverterTypeSendPaymentRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SendPaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeSendPaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) SendSpontaneousPayment(req SendSpontaneousPaymentRequest) (SendPaymentResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSendPaymentError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_send_spontaneous_payment(
			_pointer, FfiConverterTypeSendSpontaneousPaymentRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SendPaymentResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeSendPaymentResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) SetPaymentMetadata(hash string, metadata string) error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_set_payment_metadata(
			_pointer, FfiConverterStringINSTANCE.Lower(hash), FfiConverterStringINSTANCE.Lower(metadata), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) SignMessage(req SignMessageRequest) (SignMessageResponse, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_sign_message(
			_pointer, FfiConverterTypeSignMessageRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue SignMessageResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeSignMessageResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func (_self *BlockingBreezServices) Sync() error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_sync(
			_pointer, _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) UnregisterWebhook(webhookUrl string) error {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_unregister_webhook(
			_pointer, FfiConverterStringINSTANCE.Lower(webhookUrl), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func (_self *BlockingBreezServices) WithdrawLnurl(request LnUrlWithdrawRequest) (LnUrlWithdrawResult, error) {
	_pointer := _self.ffiObject.incrementPointer("*BlockingBreezServices")
	defer _self.ffiObject.decrementPointer()
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeLnUrlWithdrawError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_method_blockingbreezservices_withdraw_lnurl(
			_pointer, FfiConverterTypeLnUrlWithdrawRequestINSTANCE.Lower(request), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnUrlWithdrawResult
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeLnUrlWithdrawResultINSTANCE.Lift(_uniffiRV), _uniffiErr
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
			func(pointer unsafe.Pointer, status *C.RustCallStatus) {
				C.uniffi_breez_sdk_bindings_fn_free_blockingbreezservices(pointer, status)
			}),
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

type FfiConverterTypeAesSuccessActionDataDecrypted struct{}

var FfiConverterTypeAesSuccessActionDataDecryptedINSTANCE = FfiConverterTypeAesSuccessActionDataDecrypted{}

func (c FfiConverterTypeAesSuccessActionDataDecrypted) Lift(rb RustBufferI) AesSuccessActionDataDecrypted {
	return LiftFromRustBuffer[AesSuccessActionDataDecrypted](c, rb)
}

func (c FfiConverterTypeAesSuccessActionDataDecrypted) Read(reader io.Reader) AesSuccessActionDataDecrypted {
	return AesSuccessActionDataDecrypted{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeAesSuccessActionDataDecrypted) Lower(value AesSuccessActionDataDecrypted) RustBuffer {
	return LowerIntoRustBuffer[AesSuccessActionDataDecrypted](c, value)
}

func (c FfiConverterTypeAesSuccessActionDataDecrypted) Write(writer io.Writer, value AesSuccessActionDataDecrypted) {
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.Plaintext)
}

type FfiDestroyerTypeAesSuccessActionDataDecrypted struct{}

func (_ FfiDestroyerTypeAesSuccessActionDataDecrypted) Destroy(value AesSuccessActionDataDecrypted) {
	value.Destroy()
}

type BackupFailedData struct {
	Error string
}

func (r *BackupFailedData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Error)
}

type FfiConverterTypeBackupFailedData struct{}

var FfiConverterTypeBackupFailedDataINSTANCE = FfiConverterTypeBackupFailedData{}

func (c FfiConverterTypeBackupFailedData) Lift(rb RustBufferI) BackupFailedData {
	return LiftFromRustBuffer[BackupFailedData](c, rb)
}

func (c FfiConverterTypeBackupFailedData) Read(reader io.Reader) BackupFailedData {
	return BackupFailedData{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeBackupFailedData) Lower(value BackupFailedData) RustBuffer {
	return LowerIntoRustBuffer[BackupFailedData](c, value)
}

func (c FfiConverterTypeBackupFailedData) Write(writer io.Writer, value BackupFailedData) {
	FfiConverterStringINSTANCE.Write(writer, value.Error)
}

type FfiDestroyerTypeBackupFailedData struct{}

func (_ FfiDestroyerTypeBackupFailedData) Destroy(value BackupFailedData) {
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

type FfiConverterTypeBackupStatus struct{}

var FfiConverterTypeBackupStatusINSTANCE = FfiConverterTypeBackupStatus{}

func (c FfiConverterTypeBackupStatus) Lift(rb RustBufferI) BackupStatus {
	return LiftFromRustBuffer[BackupStatus](c, rb)
}

func (c FfiConverterTypeBackupStatus) Read(reader io.Reader) BackupStatus {
	return BackupStatus{
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeBackupStatus) Lower(value BackupStatus) RustBuffer {
	return LowerIntoRustBuffer[BackupStatus](c, value)
}

func (c FfiConverterTypeBackupStatus) Write(writer io.Writer, value BackupStatus) {
	FfiConverterBoolINSTANCE.Write(writer, value.BackedUp)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.LastBackupTime)
}

type FfiDestroyerTypeBackupStatus struct{}

func (_ FfiDestroyerTypeBackupStatus) Destroy(value BackupStatus) {
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
	FfiDestroyerTypeNetwork{}.Destroy(r.Network)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountSat)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
	FfiDestroyerOptionalString{}.Destroy(r.Message)
}

type FfiConverterTypeBitcoinAddressData struct{}

var FfiConverterTypeBitcoinAddressDataINSTANCE = FfiConverterTypeBitcoinAddressData{}

func (c FfiConverterTypeBitcoinAddressData) Lift(rb RustBufferI) BitcoinAddressData {
	return LiftFromRustBuffer[BitcoinAddressData](c, rb)
}

func (c FfiConverterTypeBitcoinAddressData) Read(reader io.Reader) BitcoinAddressData {
	return BitcoinAddressData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypeNetworkINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeBitcoinAddressData) Lower(value BitcoinAddressData) RustBuffer {
	return LowerIntoRustBuffer[BitcoinAddressData](c, value)
}

func (c FfiConverterTypeBitcoinAddressData) Write(writer io.Writer, value BitcoinAddressData) {
	FfiConverterStringINSTANCE.Write(writer, value.Address)
	FfiConverterTypeNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerTypeBitcoinAddressData struct{}

func (_ FfiDestroyerTypeBitcoinAddressData) Destroy(value BitcoinAddressData) {
	value.Destroy()
}

type BuyBitcoinRequest struct {
	Provider         BuyBitcoinProvider
	OpeningFeeParams *OpeningFeeParams
	RedirectUrl      *string
}

func (r *BuyBitcoinRequest) Destroy() {
	FfiDestroyerTypeBuyBitcoinProvider{}.Destroy(r.Provider)
	FfiDestroyerOptionalTypeOpeningFeeParams{}.Destroy(r.OpeningFeeParams)
	FfiDestroyerOptionalString{}.Destroy(r.RedirectUrl)
}

type FfiConverterTypeBuyBitcoinRequest struct{}

var FfiConverterTypeBuyBitcoinRequestINSTANCE = FfiConverterTypeBuyBitcoinRequest{}

func (c FfiConverterTypeBuyBitcoinRequest) Lift(rb RustBufferI) BuyBitcoinRequest {
	return LiftFromRustBuffer[BuyBitcoinRequest](c, rb)
}

func (c FfiConverterTypeBuyBitcoinRequest) Read(reader io.Reader) BuyBitcoinRequest {
	return BuyBitcoinRequest{
		FfiConverterTypeBuyBitcoinProviderINSTANCE.Read(reader),
		FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeBuyBitcoinRequest) Lower(value BuyBitcoinRequest) RustBuffer {
	return LowerIntoRustBuffer[BuyBitcoinRequest](c, value)
}

func (c FfiConverterTypeBuyBitcoinRequest) Write(writer io.Writer, value BuyBitcoinRequest) {
	FfiConverterTypeBuyBitcoinProviderINSTANCE.Write(writer, value.Provider)
	FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Write(writer, value.OpeningFeeParams)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.RedirectUrl)
}

type FfiDestroyerTypeBuyBitcoinRequest struct{}

func (_ FfiDestroyerTypeBuyBitcoinRequest) Destroy(value BuyBitcoinRequest) {
	value.Destroy()
}

type BuyBitcoinResponse struct {
	Url              string
	OpeningFeeParams *OpeningFeeParams
}

func (r *BuyBitcoinResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Url)
	FfiDestroyerOptionalTypeOpeningFeeParams{}.Destroy(r.OpeningFeeParams)
}

type FfiConverterTypeBuyBitcoinResponse struct{}

var FfiConverterTypeBuyBitcoinResponseINSTANCE = FfiConverterTypeBuyBitcoinResponse{}

func (c FfiConverterTypeBuyBitcoinResponse) Lift(rb RustBufferI) BuyBitcoinResponse {
	return LiftFromRustBuffer[BuyBitcoinResponse](c, rb)
}

func (c FfiConverterTypeBuyBitcoinResponse) Read(reader io.Reader) BuyBitcoinResponse {
	return BuyBitcoinResponse{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeBuyBitcoinResponse) Lower(value BuyBitcoinResponse) RustBuffer {
	return LowerIntoRustBuffer[BuyBitcoinResponse](c, value)
}

func (c FfiConverterTypeBuyBitcoinResponse) Write(writer io.Writer, value BuyBitcoinResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Url)
	FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Write(writer, value.OpeningFeeParams)
}

type FfiDestroyerTypeBuyBitcoinResponse struct{}

func (_ FfiDestroyerTypeBuyBitcoinResponse) Destroy(value BuyBitcoinResponse) {
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

type FfiConverterTypeCheckMessageRequest struct{}

var FfiConverterTypeCheckMessageRequestINSTANCE = FfiConverterTypeCheckMessageRequest{}

func (c FfiConverterTypeCheckMessageRequest) Lift(rb RustBufferI) CheckMessageRequest {
	return LiftFromRustBuffer[CheckMessageRequest](c, rb)
}

func (c FfiConverterTypeCheckMessageRequest) Read(reader io.Reader) CheckMessageRequest {
	return CheckMessageRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeCheckMessageRequest) Lower(value CheckMessageRequest) RustBuffer {
	return LowerIntoRustBuffer[CheckMessageRequest](c, value)
}

func (c FfiConverterTypeCheckMessageRequest) Write(writer io.Writer, value CheckMessageRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
	FfiConverterStringINSTANCE.Write(writer, value.Pubkey)
	FfiConverterStringINSTANCE.Write(writer, value.Signature)
}

type FfiDestroyerTypeCheckMessageRequest struct{}

func (_ FfiDestroyerTypeCheckMessageRequest) Destroy(value CheckMessageRequest) {
	value.Destroy()
}

type CheckMessageResponse struct {
	IsValid bool
}

func (r *CheckMessageResponse) Destroy() {
	FfiDestroyerBool{}.Destroy(r.IsValid)
}

type FfiConverterTypeCheckMessageResponse struct{}

var FfiConverterTypeCheckMessageResponseINSTANCE = FfiConverterTypeCheckMessageResponse{}

func (c FfiConverterTypeCheckMessageResponse) Lift(rb RustBufferI) CheckMessageResponse {
	return LiftFromRustBuffer[CheckMessageResponse](c, rb)
}

func (c FfiConverterTypeCheckMessageResponse) Read(reader io.Reader) CheckMessageResponse {
	return CheckMessageResponse{
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeCheckMessageResponse) Lower(value CheckMessageResponse) RustBuffer {
	return LowerIntoRustBuffer[CheckMessageResponse](c, value)
}

func (c FfiConverterTypeCheckMessageResponse) Write(writer io.Writer, value CheckMessageResponse) {
	FfiConverterBoolINSTANCE.Write(writer, value.IsValid)
}

type FfiDestroyerTypeCheckMessageResponse struct{}

func (_ FfiDestroyerTypeCheckMessageResponse) Destroy(value CheckMessageResponse) {
	value.Destroy()
}

type ClosedChannelPaymentDetails struct {
	State          ChannelState
	FundingTxid    string
	ShortChannelId *string
	ClosingTxid    *string
}

func (r *ClosedChannelPaymentDetails) Destroy() {
	FfiDestroyerTypeChannelState{}.Destroy(r.State)
	FfiDestroyerString{}.Destroy(r.FundingTxid)
	FfiDestroyerOptionalString{}.Destroy(r.ShortChannelId)
	FfiDestroyerOptionalString{}.Destroy(r.ClosingTxid)
}

type FfiConverterTypeClosedChannelPaymentDetails struct{}

var FfiConverterTypeClosedChannelPaymentDetailsINSTANCE = FfiConverterTypeClosedChannelPaymentDetails{}

func (c FfiConverterTypeClosedChannelPaymentDetails) Lift(rb RustBufferI) ClosedChannelPaymentDetails {
	return LiftFromRustBuffer[ClosedChannelPaymentDetails](c, rb)
}

func (c FfiConverterTypeClosedChannelPaymentDetails) Read(reader io.Reader) ClosedChannelPaymentDetails {
	return ClosedChannelPaymentDetails{
		FfiConverterTypeChannelStateINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeClosedChannelPaymentDetails) Lower(value ClosedChannelPaymentDetails) RustBuffer {
	return LowerIntoRustBuffer[ClosedChannelPaymentDetails](c, value)
}

func (c FfiConverterTypeClosedChannelPaymentDetails) Write(writer io.Writer, value ClosedChannelPaymentDetails) {
	FfiConverterTypeChannelStateINSTANCE.Write(writer, value.State)
	FfiConverterStringINSTANCE.Write(writer, value.FundingTxid)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.ShortChannelId)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.ClosingTxid)
}

type FfiDestroyerTypeClosedChannelPaymentDetails struct{}

func (_ FfiDestroyerTypeClosedChannelPaymentDetails) Destroy(value ClosedChannelPaymentDetails) {
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
	FfiDestroyerTypeNetwork{}.Destroy(r.Network)
	FfiDestroyerUint32{}.Destroy(r.PaymentTimeoutSec)
	FfiDestroyerOptionalString{}.Destroy(r.DefaultLspId)
	FfiDestroyerOptionalString{}.Destroy(r.ApiKey)
	FfiDestroyerFloat64{}.Destroy(r.MaxfeePercent)
	FfiDestroyerUint64{}.Destroy(r.ExemptfeeMsat)
	FfiDestroyerTypeNodeConfig{}.Destroy(r.NodeConfig)
}

type FfiConverterTypeConfig struct{}

var FfiConverterTypeConfigINSTANCE = FfiConverterTypeConfig{}

func (c FfiConverterTypeConfig) Lift(rb RustBufferI) Config {
	return LiftFromRustBuffer[Config](c, rb)
}

func (c FfiConverterTypeConfig) Read(reader io.Reader) Config {
	return Config{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypeNetworkINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterFloat64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterTypeNodeConfigINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeConfig) Lower(value Config) RustBuffer {
	return LowerIntoRustBuffer[Config](c, value)
}

func (c FfiConverterTypeConfig) Write(writer io.Writer, value Config) {
	FfiConverterStringINSTANCE.Write(writer, value.Breezserver)
	FfiConverterStringINSTANCE.Write(writer, value.ChainnotifierUrl)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.MempoolspaceUrl)
	FfiConverterStringINSTANCE.Write(writer, value.WorkingDir)
	FfiConverterTypeNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterUint32INSTANCE.Write(writer, value.PaymentTimeoutSec)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.DefaultLspId)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.ApiKey)
	FfiConverterFloat64INSTANCE.Write(writer, value.MaxfeePercent)
	FfiConverterUint64INSTANCE.Write(writer, value.ExemptfeeMsat)
	FfiConverterTypeNodeConfigINSTANCE.Write(writer, value.NodeConfig)
}

type FfiDestroyerTypeConfig struct{}

func (_ FfiDestroyerTypeConfig) Destroy(value Config) {
	value.Destroy()
}

type ConfigureNodeRequest struct {
	CloseToAddress *string
}

func (r *ConfigureNodeRequest) Destroy() {
	FfiDestroyerOptionalString{}.Destroy(r.CloseToAddress)
}

type FfiConverterTypeConfigureNodeRequest struct{}

var FfiConverterTypeConfigureNodeRequestINSTANCE = FfiConverterTypeConfigureNodeRequest{}

func (c FfiConverterTypeConfigureNodeRequest) Lift(rb RustBufferI) ConfigureNodeRequest {
	return LiftFromRustBuffer[ConfigureNodeRequest](c, rb)
}

func (c FfiConverterTypeConfigureNodeRequest) Read(reader io.Reader) ConfigureNodeRequest {
	return ConfigureNodeRequest{
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeConfigureNodeRequest) Lower(value ConfigureNodeRequest) RustBuffer {
	return LowerIntoRustBuffer[ConfigureNodeRequest](c, value)
}

func (c FfiConverterTypeConfigureNodeRequest) Write(writer io.Writer, value ConfigureNodeRequest) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.CloseToAddress)
}

type FfiDestroyerTypeConfigureNodeRequest struct{}

func (_ FfiDestroyerTypeConfigureNodeRequest) Destroy(value ConfigureNodeRequest) {
	value.Destroy()
}

type ConnectRequest struct {
	Config      Config
	Seed        []uint8
	RestoreOnly *bool
}

func (r *ConnectRequest) Destroy() {
	FfiDestroyerTypeConfig{}.Destroy(r.Config)
	FfiDestroyerSequenceUint8{}.Destroy(r.Seed)
	FfiDestroyerOptionalBool{}.Destroy(r.RestoreOnly)
}

type FfiConverterTypeConnectRequest struct{}

var FfiConverterTypeConnectRequestINSTANCE = FfiConverterTypeConnectRequest{}

func (c FfiConverterTypeConnectRequest) Lift(rb RustBufferI) ConnectRequest {
	return LiftFromRustBuffer[ConnectRequest](c, rb)
}

func (c FfiConverterTypeConnectRequest) Read(reader io.Reader) ConnectRequest {
	return ConnectRequest{
		FfiConverterTypeConfigINSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeConnectRequest) Lower(value ConnectRequest) RustBuffer {
	return LowerIntoRustBuffer[ConnectRequest](c, value)
}

func (c FfiConverterTypeConnectRequest) Write(writer io.Writer, value ConnectRequest) {
	FfiConverterTypeConfigINSTANCE.Write(writer, value.Config)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Seed)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.RestoreOnly)
}

type FfiDestroyerTypeConnectRequest struct{}

func (_ FfiDestroyerTypeConnectRequest) Destroy(value ConnectRequest) {
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
	FfiDestroyerOptionalTypeSymbol{}.Destroy(r.Symbol)
	FfiDestroyerOptionalTypeSymbol{}.Destroy(r.UniqSymbol)
	FfiDestroyerSequenceTypeLocalizedName{}.Destroy(r.LocalizedName)
	FfiDestroyerSequenceTypeLocaleOverrides{}.Destroy(r.LocaleOverrides)
}

type FfiConverterTypeCurrencyInfo struct{}

var FfiConverterTypeCurrencyInfoINSTANCE = FfiConverterTypeCurrencyInfo{}

func (c FfiConverterTypeCurrencyInfo) Lift(rb RustBufferI) CurrencyInfo {
	return LiftFromRustBuffer[CurrencyInfo](c, rb)
}

func (c FfiConverterTypeCurrencyInfo) Read(reader io.Reader) CurrencyInfo {
	return CurrencyInfo{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalTypeSymbolINSTANCE.Read(reader),
		FfiConverterOptionalTypeSymbolINSTANCE.Read(reader),
		FfiConverterSequenceTypeLocalizedNameINSTANCE.Read(reader),
		FfiConverterSequenceTypeLocaleOverridesINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeCurrencyInfo) Lower(value CurrencyInfo) RustBuffer {
	return LowerIntoRustBuffer[CurrencyInfo](c, value)
}

func (c FfiConverterTypeCurrencyInfo) Write(writer io.Writer, value CurrencyInfo) {
	FfiConverterStringINSTANCE.Write(writer, value.Name)
	FfiConverterUint32INSTANCE.Write(writer, value.FractionSize)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Spacing)
	FfiConverterOptionalTypeSymbolINSTANCE.Write(writer, value.Symbol)
	FfiConverterOptionalTypeSymbolINSTANCE.Write(writer, value.UniqSymbol)
	FfiConverterSequenceTypeLocalizedNameINSTANCE.Write(writer, value.LocalizedName)
	FfiConverterSequenceTypeLocaleOverridesINSTANCE.Write(writer, value.LocaleOverrides)
}

type FfiDestroyerTypeCurrencyInfo struct{}

func (_ FfiDestroyerTypeCurrencyInfo) Destroy(value CurrencyInfo) {
	value.Destroy()
}

type FiatCurrency struct {
	Id   string
	Info CurrencyInfo
}

func (r *FiatCurrency) Destroy() {
	FfiDestroyerString{}.Destroy(r.Id)
	FfiDestroyerTypeCurrencyInfo{}.Destroy(r.Info)
}

type FfiConverterTypeFiatCurrency struct{}

var FfiConverterTypeFiatCurrencyINSTANCE = FfiConverterTypeFiatCurrency{}

func (c FfiConverterTypeFiatCurrency) Lift(rb RustBufferI) FiatCurrency {
	return LiftFromRustBuffer[FiatCurrency](c, rb)
}

func (c FfiConverterTypeFiatCurrency) Read(reader io.Reader) FiatCurrency {
	return FiatCurrency{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypeCurrencyInfoINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeFiatCurrency) Lower(value FiatCurrency) RustBuffer {
	return LowerIntoRustBuffer[FiatCurrency](c, value)
}

func (c FfiConverterTypeFiatCurrency) Write(writer io.Writer, value FiatCurrency) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterTypeCurrencyInfoINSTANCE.Write(writer, value.Info)
}

type FfiDestroyerTypeFiatCurrency struct{}

func (_ FfiDestroyerTypeFiatCurrency) Destroy(value FiatCurrency) {
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

type FfiConverterTypeGreenlightCredentials struct{}

var FfiConverterTypeGreenlightCredentialsINSTANCE = FfiConverterTypeGreenlightCredentials{}

func (c FfiConverterTypeGreenlightCredentials) Lift(rb RustBufferI) GreenlightCredentials {
	return LiftFromRustBuffer[GreenlightCredentials](c, rb)
}

func (c FfiConverterTypeGreenlightCredentials) Read(reader io.Reader) GreenlightCredentials {
	return GreenlightCredentials{
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeGreenlightCredentials) Lower(value GreenlightCredentials) RustBuffer {
	return LowerIntoRustBuffer[GreenlightCredentials](c, value)
}

func (c FfiConverterTypeGreenlightCredentials) Write(writer io.Writer, value GreenlightCredentials) {
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.DeveloperKey)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.DeveloperCert)
}

type FfiDestroyerTypeGreenlightCredentials struct{}

func (_ FfiDestroyerTypeGreenlightCredentials) Destroy(value GreenlightCredentials) {
	value.Destroy()
}

type GreenlightDeviceCredentials struct {
	Device []uint8
}

func (r *GreenlightDeviceCredentials) Destroy() {
	FfiDestroyerSequenceUint8{}.Destroy(r.Device)
}

type FfiConverterTypeGreenlightDeviceCredentials struct{}

var FfiConverterTypeGreenlightDeviceCredentialsINSTANCE = FfiConverterTypeGreenlightDeviceCredentials{}

func (c FfiConverterTypeGreenlightDeviceCredentials) Lift(rb RustBufferI) GreenlightDeviceCredentials {
	return LiftFromRustBuffer[GreenlightDeviceCredentials](c, rb)
}

func (c FfiConverterTypeGreenlightDeviceCredentials) Read(reader io.Reader) GreenlightDeviceCredentials {
	return GreenlightDeviceCredentials{
		FfiConverterSequenceUint8INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeGreenlightDeviceCredentials) Lower(value GreenlightDeviceCredentials) RustBuffer {
	return LowerIntoRustBuffer[GreenlightDeviceCredentials](c, value)
}

func (c FfiConverterTypeGreenlightDeviceCredentials) Write(writer io.Writer, value GreenlightDeviceCredentials) {
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Device)
}

type FfiDestroyerTypeGreenlightDeviceCredentials struct{}

func (_ FfiDestroyerTypeGreenlightDeviceCredentials) Destroy(value GreenlightDeviceCredentials) {
	value.Destroy()
}

type GreenlightNodeConfig struct {
	PartnerCredentials *GreenlightCredentials
	InviteCode         *string
}

func (r *GreenlightNodeConfig) Destroy() {
	FfiDestroyerOptionalTypeGreenlightCredentials{}.Destroy(r.PartnerCredentials)
	FfiDestroyerOptionalString{}.Destroy(r.InviteCode)
}

type FfiConverterTypeGreenlightNodeConfig struct{}

var FfiConverterTypeGreenlightNodeConfigINSTANCE = FfiConverterTypeGreenlightNodeConfig{}

func (c FfiConverterTypeGreenlightNodeConfig) Lift(rb RustBufferI) GreenlightNodeConfig {
	return LiftFromRustBuffer[GreenlightNodeConfig](c, rb)
}

func (c FfiConverterTypeGreenlightNodeConfig) Read(reader io.Reader) GreenlightNodeConfig {
	return GreenlightNodeConfig{
		FfiConverterOptionalTypeGreenlightCredentialsINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeGreenlightNodeConfig) Lower(value GreenlightNodeConfig) RustBuffer {
	return LowerIntoRustBuffer[GreenlightNodeConfig](c, value)
}

func (c FfiConverterTypeGreenlightNodeConfig) Write(writer io.Writer, value GreenlightNodeConfig) {
	FfiConverterOptionalTypeGreenlightCredentialsINSTANCE.Write(writer, value.PartnerCredentials)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.InviteCode)
}

type FfiDestroyerTypeGreenlightNodeConfig struct{}

func (_ FfiDestroyerTypeGreenlightNodeConfig) Destroy(value GreenlightNodeConfig) {
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
	FfiDestroyerOptionalTypePayment{}.Destroy(r.Payment)
}

type FfiConverterTypeInvoicePaidDetails struct{}

var FfiConverterTypeInvoicePaidDetailsINSTANCE = FfiConverterTypeInvoicePaidDetails{}

func (c FfiConverterTypeInvoicePaidDetails) Lift(rb RustBufferI) InvoicePaidDetails {
	return LiftFromRustBuffer[InvoicePaidDetails](c, rb)
}

func (c FfiConverterTypeInvoicePaidDetails) Read(reader io.Reader) InvoicePaidDetails {
	return InvoicePaidDetails{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalTypePaymentINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeInvoicePaidDetails) Lower(value InvoicePaidDetails) RustBuffer {
	return LowerIntoRustBuffer[InvoicePaidDetails](c, value)
}

func (c FfiConverterTypeInvoicePaidDetails) Write(writer io.Writer, value InvoicePaidDetails) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterOptionalTypePaymentINSTANCE.Write(writer, value.Payment)
}

type FfiDestroyerTypeInvoicePaidDetails struct{}

func (_ FfiDestroyerTypeInvoicePaidDetails) Destroy(value InvoicePaidDetails) {
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
	FfiDestroyerTypeNetwork{}.Destroy(r.Network)
	FfiDestroyerString{}.Destroy(r.PayeePubkey)
	FfiDestroyerString{}.Destroy(r.PaymentHash)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerOptionalString{}.Destroy(r.DescriptionHash)
	FfiDestroyerOptionalUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerUint64{}.Destroy(r.Timestamp)
	FfiDestroyerUint64{}.Destroy(r.Expiry)
	FfiDestroyerSequenceTypeRouteHint{}.Destroy(r.RoutingHints)
	FfiDestroyerSequenceUint8{}.Destroy(r.PaymentSecret)
	FfiDestroyerUint64{}.Destroy(r.MinFinalCltvExpiryDelta)
}

type FfiConverterTypeLNInvoice struct{}

var FfiConverterTypeLNInvoiceINSTANCE = FfiConverterTypeLNInvoice{}

func (c FfiConverterTypeLNInvoice) Lift(rb RustBufferI) LnInvoice {
	return LiftFromRustBuffer[LnInvoice](c, rb)
}

func (c FfiConverterTypeLNInvoice) Read(reader io.Reader) LnInvoice {
	return LnInvoice{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypeNetworkINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSequenceTypeRouteHintINSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLNInvoice) Lower(value LnInvoice) RustBuffer {
	return LowerIntoRustBuffer[LnInvoice](c, value)
}

func (c FfiConverterTypeLNInvoice) Write(writer io.Writer, value LnInvoice) {
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterTypeNetworkINSTANCE.Write(writer, value.Network)
	FfiConverterStringINSTANCE.Write(writer, value.PayeePubkey)
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.DescriptionHash)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.Timestamp)
	FfiConverterUint64INSTANCE.Write(writer, value.Expiry)
	FfiConverterSequenceTypeRouteHintINSTANCE.Write(writer, value.RoutingHints)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.PaymentSecret)
	FfiConverterUint64INSTANCE.Write(writer, value.MinFinalCltvExpiryDelta)
}

type FfiDestroyerTypeLnInvoice struct{}

func (_ FfiDestroyerTypeLnInvoice) Destroy(value LnInvoice) {
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
	FfiDestroyerOptionalSequenceTypePaymentTypeFilter{}.Destroy(r.Filters)
	FfiDestroyerOptionalSequenceTypeMetadataFilter{}.Destroy(r.MetadataFilters)
	FfiDestroyerOptionalInt64{}.Destroy(r.FromTimestamp)
	FfiDestroyerOptionalInt64{}.Destroy(r.ToTimestamp)
	FfiDestroyerOptionalBool{}.Destroy(r.IncludeFailures)
	FfiDestroyerOptionalUint32{}.Destroy(r.Offset)
	FfiDestroyerOptionalUint32{}.Destroy(r.Limit)
}

type FfiConverterTypeListPaymentsRequest struct{}

var FfiConverterTypeListPaymentsRequestINSTANCE = FfiConverterTypeListPaymentsRequest{}

func (c FfiConverterTypeListPaymentsRequest) Lift(rb RustBufferI) ListPaymentsRequest {
	return LiftFromRustBuffer[ListPaymentsRequest](c, rb)
}

func (c FfiConverterTypeListPaymentsRequest) Read(reader io.Reader) ListPaymentsRequest {
	return ListPaymentsRequest{
		FfiConverterOptionalSequenceTypePaymentTypeFilterINSTANCE.Read(reader),
		FfiConverterOptionalSequenceTypeMetadataFilterINSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListPaymentsRequest) Lower(value ListPaymentsRequest) RustBuffer {
	return LowerIntoRustBuffer[ListPaymentsRequest](c, value)
}

func (c FfiConverterTypeListPaymentsRequest) Write(writer io.Writer, value ListPaymentsRequest) {
	FfiConverterOptionalSequenceTypePaymentTypeFilterINSTANCE.Write(writer, value.Filters)
	FfiConverterOptionalSequenceTypeMetadataFilterINSTANCE.Write(writer, value.MetadataFilters)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.FromTimestamp)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.ToTimestamp)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.IncludeFailures)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Offset)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Limit)
}

type FfiDestroyerTypeListPaymentsRequest struct{}

func (_ FfiDestroyerTypeListPaymentsRequest) Destroy(value ListPaymentsRequest) {
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
	FfiDestroyerOptionalSequenceTypeSwapStatus{}.Destroy(r.Status)
	FfiDestroyerOptionalInt64{}.Destroy(r.FromTimestamp)
	FfiDestroyerOptionalInt64{}.Destroy(r.ToTimestamp)
	FfiDestroyerOptionalUint32{}.Destroy(r.Offset)
	FfiDestroyerOptionalUint32{}.Destroy(r.Limit)
}

type FfiConverterTypeListSwapsRequest struct{}

var FfiConverterTypeListSwapsRequestINSTANCE = FfiConverterTypeListSwapsRequest{}

func (c FfiConverterTypeListSwapsRequest) Lift(rb RustBufferI) ListSwapsRequest {
	return LiftFromRustBuffer[ListSwapsRequest](c, rb)
}

func (c FfiConverterTypeListSwapsRequest) Read(reader io.Reader) ListSwapsRequest {
	return ListSwapsRequest{
		FfiConverterOptionalSequenceTypeSwapStatusINSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalInt64INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeListSwapsRequest) Lower(value ListSwapsRequest) RustBuffer {
	return LowerIntoRustBuffer[ListSwapsRequest](c, value)
}

func (c FfiConverterTypeListSwapsRequest) Write(writer io.Writer, value ListSwapsRequest) {
	FfiConverterOptionalSequenceTypeSwapStatusINSTANCE.Write(writer, value.Status)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.FromTimestamp)
	FfiConverterOptionalInt64INSTANCE.Write(writer, value.ToTimestamp)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Offset)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Limit)
}

type FfiDestroyerTypeListSwapsRequest struct{}

func (_ FfiDestroyerTypeListSwapsRequest) Destroy(value ListSwapsRequest) {
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
	FfiDestroyerOptionalTypeSuccessActionProcessed{}.Destroy(r.LnurlSuccessAction)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlPayDomain)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlPayComment)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlMetadata)
	FfiDestroyerOptionalString{}.Destroy(r.LnAddress)
	FfiDestroyerOptionalString{}.Destroy(r.LnurlWithdrawEndpoint)
	FfiDestroyerOptionalTypeSwapInfo{}.Destroy(r.SwapInfo)
	FfiDestroyerOptionalTypeReverseSwapInfo{}.Destroy(r.ReverseSwapInfo)
	FfiDestroyerOptionalUint32{}.Destroy(r.PendingExpirationBlock)
}

type FfiConverterTypeLnPaymentDetails struct{}

var FfiConverterTypeLnPaymentDetailsINSTANCE = FfiConverterTypeLnPaymentDetails{}

func (c FfiConverterTypeLnPaymentDetails) Lift(rb RustBufferI) LnPaymentDetails {
	return LiftFromRustBuffer[LnPaymentDetails](c, rb)
}

func (c FfiConverterTypeLnPaymentDetails) Read(reader io.Reader) LnPaymentDetails {
	return LnPaymentDetails{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalTypeSuccessActionProcessedINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalTypeSwapInfoINSTANCE.Read(reader),
		FfiConverterOptionalTypeReverseSwapInfoINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnPaymentDetails) Lower(value LnPaymentDetails) RustBuffer {
	return LowerIntoRustBuffer[LnPaymentDetails](c, value)
}

func (c FfiConverterTypeLnPaymentDetails) Write(writer io.Writer, value LnPaymentDetails) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterStringINSTANCE.Write(writer, value.Label)
	FfiConverterStringINSTANCE.Write(writer, value.DestinationPubkey)
	FfiConverterStringINSTANCE.Write(writer, value.PaymentPreimage)
	FfiConverterBoolINSTANCE.Write(writer, value.Keysend)
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.OpenChannelBolt11)
	FfiConverterOptionalTypeSuccessActionProcessedINSTANCE.Write(writer, value.LnurlSuccessAction)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlPayDomain)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlPayComment)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlMetadata)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnAddress)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LnurlWithdrawEndpoint)
	FfiConverterOptionalTypeSwapInfoINSTANCE.Write(writer, value.SwapInfo)
	FfiConverterOptionalTypeReverseSwapInfoINSTANCE.Write(writer, value.ReverseSwapInfo)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.PendingExpirationBlock)
}

type FfiDestroyerTypeLnPaymentDetails struct{}

func (_ FfiDestroyerTypeLnPaymentDetails) Destroy(value LnPaymentDetails) {
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

type FfiConverterTypeLnUrlAuthRequestData struct{}

var FfiConverterTypeLnUrlAuthRequestDataINSTANCE = FfiConverterTypeLnUrlAuthRequestData{}

func (c FfiConverterTypeLnUrlAuthRequestData) Lift(rb RustBufferI) LnUrlAuthRequestData {
	return LiftFromRustBuffer[LnUrlAuthRequestData](c, rb)
}

func (c FfiConverterTypeLnUrlAuthRequestData) Read(reader io.Reader) LnUrlAuthRequestData {
	return LnUrlAuthRequestData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlAuthRequestData) Lower(value LnUrlAuthRequestData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlAuthRequestData](c, value)
}

func (c FfiConverterTypeLnUrlAuthRequestData) Write(writer io.Writer, value LnUrlAuthRequestData) {
	FfiConverterStringINSTANCE.Write(writer, value.K1)
	FfiConverterStringINSTANCE.Write(writer, value.Domain)
	FfiConverterStringINSTANCE.Write(writer, value.Url)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Action)
}

type FfiDestroyerTypeLnUrlAuthRequestData struct{}

func (_ FfiDestroyerTypeLnUrlAuthRequestData) Destroy(value LnUrlAuthRequestData) {
	value.Destroy()
}

type LnUrlErrorData struct {
	Reason string
}

func (r *LnUrlErrorData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Reason)
}

type FfiConverterTypeLnUrlErrorData struct{}

var FfiConverterTypeLnUrlErrorDataINSTANCE = FfiConverterTypeLnUrlErrorData{}

func (c FfiConverterTypeLnUrlErrorData) Lift(rb RustBufferI) LnUrlErrorData {
	return LiftFromRustBuffer[LnUrlErrorData](c, rb)
}

func (c FfiConverterTypeLnUrlErrorData) Read(reader io.Reader) LnUrlErrorData {
	return LnUrlErrorData{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlErrorData) Lower(value LnUrlErrorData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlErrorData](c, value)
}

func (c FfiConverterTypeLnUrlErrorData) Write(writer io.Writer, value LnUrlErrorData) {
	FfiConverterStringINSTANCE.Write(writer, value.Reason)
}

type FfiDestroyerTypeLnUrlErrorData struct{}

func (_ FfiDestroyerTypeLnUrlErrorData) Destroy(value LnUrlErrorData) {
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

type FfiConverterTypeLnUrlPayErrorData struct{}

var FfiConverterTypeLnUrlPayErrorDataINSTANCE = FfiConverterTypeLnUrlPayErrorData{}

func (c FfiConverterTypeLnUrlPayErrorData) Lift(rb RustBufferI) LnUrlPayErrorData {
	return LiftFromRustBuffer[LnUrlPayErrorData](c, rb)
}

func (c FfiConverterTypeLnUrlPayErrorData) Read(reader io.Reader) LnUrlPayErrorData {
	return LnUrlPayErrorData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlPayErrorData) Lower(value LnUrlPayErrorData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayErrorData](c, value)
}

func (c FfiConverterTypeLnUrlPayErrorData) Write(writer io.Writer, value LnUrlPayErrorData) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterStringINSTANCE.Write(writer, value.Reason)
}

type FfiDestroyerTypeLnUrlPayErrorData struct{}

func (_ FfiDestroyerTypeLnUrlPayErrorData) Destroy(value LnUrlPayErrorData) {
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
	FfiDestroyerTypeLnUrlPayRequestData{}.Destroy(r.Data)
	FfiDestroyerUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerBool{}.Destroy(r.UseTrampoline)
	FfiDestroyerOptionalString{}.Destroy(r.Comment)
	FfiDestroyerOptionalString{}.Destroy(r.PaymentLabel)
	FfiDestroyerOptionalBool{}.Destroy(r.ValidateSuccessActionUrl)
}

type FfiConverterTypeLnUrlPayRequest struct{}

var FfiConverterTypeLnUrlPayRequestINSTANCE = FfiConverterTypeLnUrlPayRequest{}

func (c FfiConverterTypeLnUrlPayRequest) Lift(rb RustBufferI) LnUrlPayRequest {
	return LiftFromRustBuffer[LnUrlPayRequest](c, rb)
}

func (c FfiConverterTypeLnUrlPayRequest) Read(reader io.Reader) LnUrlPayRequest {
	return LnUrlPayRequest{
		FfiConverterTypeLnUrlPayRequestDataINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlPayRequest) Lower(value LnUrlPayRequest) RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayRequest](c, value)
}

func (c FfiConverterTypeLnUrlPayRequest) Write(writer io.Writer, value LnUrlPayRequest) {
	FfiConverterTypeLnUrlPayRequestDataINSTANCE.Write(writer, value.Data)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterBoolINSTANCE.Write(writer, value.UseTrampoline)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Comment)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.PaymentLabel)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.ValidateSuccessActionUrl)
}

type FfiDestroyerTypeLnUrlPayRequest struct{}

func (_ FfiDestroyerTypeLnUrlPayRequest) Destroy(value LnUrlPayRequest) {
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

type FfiConverterTypeLnUrlPayRequestData struct{}

var FfiConverterTypeLnUrlPayRequestDataINSTANCE = FfiConverterTypeLnUrlPayRequestData{}

func (c FfiConverterTypeLnUrlPayRequestData) Lift(rb RustBufferI) LnUrlPayRequestData {
	return LiftFromRustBuffer[LnUrlPayRequestData](c, rb)
}

func (c FfiConverterTypeLnUrlPayRequestData) Read(reader io.Reader) LnUrlPayRequestData {
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

func (c FfiConverterTypeLnUrlPayRequestData) Lower(value LnUrlPayRequestData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayRequestData](c, value)
}

func (c FfiConverterTypeLnUrlPayRequestData) Write(writer io.Writer, value LnUrlPayRequestData) {
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

type FfiDestroyerTypeLnUrlPayRequestData struct{}

func (_ FfiDestroyerTypeLnUrlPayRequestData) Destroy(value LnUrlPayRequestData) {
	value.Destroy()
}

type LnUrlPaySuccessData struct {
	SuccessAction *SuccessActionProcessed
	Payment       Payment
}

func (r *LnUrlPaySuccessData) Destroy() {
	FfiDestroyerOptionalTypeSuccessActionProcessed{}.Destroy(r.SuccessAction)
	FfiDestroyerTypePayment{}.Destroy(r.Payment)
}

type FfiConverterTypeLnUrlPaySuccessData struct{}

var FfiConverterTypeLnUrlPaySuccessDataINSTANCE = FfiConverterTypeLnUrlPaySuccessData{}

func (c FfiConverterTypeLnUrlPaySuccessData) Lift(rb RustBufferI) LnUrlPaySuccessData {
	return LiftFromRustBuffer[LnUrlPaySuccessData](c, rb)
}

func (c FfiConverterTypeLnUrlPaySuccessData) Read(reader io.Reader) LnUrlPaySuccessData {
	return LnUrlPaySuccessData{
		FfiConverterOptionalTypeSuccessActionProcessedINSTANCE.Read(reader),
		FfiConverterTypePaymentINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlPaySuccessData) Lower(value LnUrlPaySuccessData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlPaySuccessData](c, value)
}

func (c FfiConverterTypeLnUrlPaySuccessData) Write(writer io.Writer, value LnUrlPaySuccessData) {
	FfiConverterOptionalTypeSuccessActionProcessedINSTANCE.Write(writer, value.SuccessAction)
	FfiConverterTypePaymentINSTANCE.Write(writer, value.Payment)
}

type FfiDestroyerTypeLnUrlPaySuccessData struct{}

func (_ FfiDestroyerTypeLnUrlPaySuccessData) Destroy(value LnUrlPaySuccessData) {
	value.Destroy()
}

type LnUrlWithdrawRequest struct {
	Data        LnUrlWithdrawRequestData
	AmountMsat  uint64
	Description *string
}

func (r *LnUrlWithdrawRequest) Destroy() {
	FfiDestroyerTypeLnUrlWithdrawRequestData{}.Destroy(r.Data)
	FfiDestroyerUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
}

type FfiConverterTypeLnUrlWithdrawRequest struct{}

var FfiConverterTypeLnUrlWithdrawRequestINSTANCE = FfiConverterTypeLnUrlWithdrawRequest{}

func (c FfiConverterTypeLnUrlWithdrawRequest) Lift(rb RustBufferI) LnUrlWithdrawRequest {
	return LiftFromRustBuffer[LnUrlWithdrawRequest](c, rb)
}

func (c FfiConverterTypeLnUrlWithdrawRequest) Read(reader io.Reader) LnUrlWithdrawRequest {
	return LnUrlWithdrawRequest{
		FfiConverterTypeLnUrlWithdrawRequestDataINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlWithdrawRequest) Lower(value LnUrlWithdrawRequest) RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawRequest](c, value)
}

func (c FfiConverterTypeLnUrlWithdrawRequest) Write(writer io.Writer, value LnUrlWithdrawRequest) {
	FfiConverterTypeLnUrlWithdrawRequestDataINSTANCE.Write(writer, value.Data)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
}

type FfiDestroyerTypeLnUrlWithdrawRequest struct{}

func (_ FfiDestroyerTypeLnUrlWithdrawRequest) Destroy(value LnUrlWithdrawRequest) {
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

type FfiConverterTypeLnUrlWithdrawRequestData struct{}

var FfiConverterTypeLnUrlWithdrawRequestDataINSTANCE = FfiConverterTypeLnUrlWithdrawRequestData{}

func (c FfiConverterTypeLnUrlWithdrawRequestData) Lift(rb RustBufferI) LnUrlWithdrawRequestData {
	return LiftFromRustBuffer[LnUrlWithdrawRequestData](c, rb)
}

func (c FfiConverterTypeLnUrlWithdrawRequestData) Read(reader io.Reader) LnUrlWithdrawRequestData {
	return LnUrlWithdrawRequestData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlWithdrawRequestData) Lower(value LnUrlWithdrawRequestData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawRequestData](c, value)
}

func (c FfiConverterTypeLnUrlWithdrawRequestData) Write(writer io.Writer, value LnUrlWithdrawRequestData) {
	FfiConverterStringINSTANCE.Write(writer, value.Callback)
	FfiConverterStringINSTANCE.Write(writer, value.K1)
	FfiConverterStringINSTANCE.Write(writer, value.DefaultDescription)
	FfiConverterUint64INSTANCE.Write(writer, value.MinWithdrawable)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxWithdrawable)
}

type FfiDestroyerTypeLnUrlWithdrawRequestData struct{}

func (_ FfiDestroyerTypeLnUrlWithdrawRequestData) Destroy(value LnUrlWithdrawRequestData) {
	value.Destroy()
}

type LnUrlWithdrawSuccessData struct {
	Invoice LnInvoice
}

func (r *LnUrlWithdrawSuccessData) Destroy() {
	FfiDestroyerTypeLnInvoice{}.Destroy(r.Invoice)
}

type FfiConverterTypeLnUrlWithdrawSuccessData struct{}

var FfiConverterTypeLnUrlWithdrawSuccessDataINSTANCE = FfiConverterTypeLnUrlWithdrawSuccessData{}

func (c FfiConverterTypeLnUrlWithdrawSuccessData) Lift(rb RustBufferI) LnUrlWithdrawSuccessData {
	return LiftFromRustBuffer[LnUrlWithdrawSuccessData](c, rb)
}

func (c FfiConverterTypeLnUrlWithdrawSuccessData) Read(reader io.Reader) LnUrlWithdrawSuccessData {
	return LnUrlWithdrawSuccessData{
		FfiConverterTypeLNInvoiceINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLnUrlWithdrawSuccessData) Lower(value LnUrlWithdrawSuccessData) RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawSuccessData](c, value)
}

func (c FfiConverterTypeLnUrlWithdrawSuccessData) Write(writer io.Writer, value LnUrlWithdrawSuccessData) {
	FfiConverterTypeLNInvoiceINSTANCE.Write(writer, value.Invoice)
}

type FfiDestroyerTypeLnUrlWithdrawSuccessData struct{}

func (_ FfiDestroyerTypeLnUrlWithdrawSuccessData) Destroy(value LnUrlWithdrawSuccessData) {
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
	FfiDestroyerTypeSymbol{}.Destroy(r.Symbol)
}

type FfiConverterTypeLocaleOverrides struct{}

var FfiConverterTypeLocaleOverridesINSTANCE = FfiConverterTypeLocaleOverrides{}

func (c FfiConverterTypeLocaleOverrides) Lift(rb RustBufferI) LocaleOverrides {
	return LiftFromRustBuffer[LocaleOverrides](c, rb)
}

func (c FfiConverterTypeLocaleOverrides) Read(reader io.Reader) LocaleOverrides {
	return LocaleOverrides{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterTypeSymbolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLocaleOverrides) Lower(value LocaleOverrides) RustBuffer {
	return LowerIntoRustBuffer[LocaleOverrides](c, value)
}

func (c FfiConverterTypeLocaleOverrides) Write(writer io.Writer, value LocaleOverrides) {
	FfiConverterStringINSTANCE.Write(writer, value.Locale)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Spacing)
	FfiConverterTypeSymbolINSTANCE.Write(writer, value.Symbol)
}

type FfiDestroyerTypeLocaleOverrides struct{}

func (_ FfiDestroyerTypeLocaleOverrides) Destroy(value LocaleOverrides) {
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

type FfiConverterTypeLocalizedName struct{}

var FfiConverterTypeLocalizedNameINSTANCE = FfiConverterTypeLocalizedName{}

func (c FfiConverterTypeLocalizedName) Lift(rb RustBufferI) LocalizedName {
	return LiftFromRustBuffer[LocalizedName](c, rb)
}

func (c FfiConverterTypeLocalizedName) Read(reader io.Reader) LocalizedName {
	return LocalizedName{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLocalizedName) Lower(value LocalizedName) RustBuffer {
	return LowerIntoRustBuffer[LocalizedName](c, value)
}

func (c FfiConverterTypeLocalizedName) Write(writer io.Writer, value LocalizedName) {
	FfiConverterStringINSTANCE.Write(writer, value.Locale)
	FfiConverterStringINSTANCE.Write(writer, value.Name)
}

type FfiDestroyerTypeLocalizedName struct{}

func (_ FfiDestroyerTypeLocalizedName) Destroy(value LocalizedName) {
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

type FfiConverterTypeLogEntry struct{}

var FfiConverterTypeLogEntryINSTANCE = FfiConverterTypeLogEntry{}

func (c FfiConverterTypeLogEntry) Lift(rb RustBufferI) LogEntry {
	return LiftFromRustBuffer[LogEntry](c, rb)
}

func (c FfiConverterTypeLogEntry) Read(reader io.Reader) LogEntry {
	return LogEntry{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLogEntry) Lower(value LogEntry) RustBuffer {
	return LowerIntoRustBuffer[LogEntry](c, value)
}

func (c FfiConverterTypeLogEntry) Write(writer io.Writer, value LogEntry) {
	FfiConverterStringINSTANCE.Write(writer, value.Line)
	FfiConverterStringINSTANCE.Write(writer, value.Level)
}

type FfiDestroyerTypeLogEntry struct{}

func (_ FfiDestroyerTypeLogEntry) Destroy(value LogEntry) {
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
	FfiDestroyerTypeOpeningFeeParamsMenu{}.Destroy(r.OpeningFeeParamsList)
}

type FfiConverterTypeLspInformation struct{}

var FfiConverterTypeLspInformationINSTANCE = FfiConverterTypeLspInformation{}

func (c FfiConverterTypeLspInformation) Lift(rb RustBufferI) LspInformation {
	return LiftFromRustBuffer[LspInformation](c, rb)
}

func (c FfiConverterTypeLspInformation) Read(reader io.Reader) LspInformation {
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
		FfiConverterTypeOpeningFeeParamsMenuINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeLspInformation) Lower(value LspInformation) RustBuffer {
	return LowerIntoRustBuffer[LspInformation](c, value)
}

func (c FfiConverterTypeLspInformation) Write(writer io.Writer, value LspInformation) {
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
	FfiConverterTypeOpeningFeeParamsMenuINSTANCE.Write(writer, value.OpeningFeeParamsList)
}

type FfiDestroyerTypeLspInformation struct{}

func (_ FfiDestroyerTypeLspInformation) Destroy(value LspInformation) {
	value.Destroy()
}

type MessageSuccessActionData struct {
	Message string
}

func (r *MessageSuccessActionData) Destroy() {
	FfiDestroyerString{}.Destroy(r.Message)
}

type FfiConverterTypeMessageSuccessActionData struct{}

var FfiConverterTypeMessageSuccessActionDataINSTANCE = FfiConverterTypeMessageSuccessActionData{}

func (c FfiConverterTypeMessageSuccessActionData) Lift(rb RustBufferI) MessageSuccessActionData {
	return LiftFromRustBuffer[MessageSuccessActionData](c, rb)
}

func (c FfiConverterTypeMessageSuccessActionData) Read(reader io.Reader) MessageSuccessActionData {
	return MessageSuccessActionData{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeMessageSuccessActionData) Lower(value MessageSuccessActionData) RustBuffer {
	return LowerIntoRustBuffer[MessageSuccessActionData](c, value)
}

func (c FfiConverterTypeMessageSuccessActionData) Write(writer io.Writer, value MessageSuccessActionData) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerTypeMessageSuccessActionData struct{}

func (_ FfiDestroyerTypeMessageSuccessActionData) Destroy(value MessageSuccessActionData) {
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

type FfiConverterTypeMetadataFilter struct{}

var FfiConverterTypeMetadataFilterINSTANCE = FfiConverterTypeMetadataFilter{}

func (c FfiConverterTypeMetadataFilter) Lift(rb RustBufferI) MetadataFilter {
	return LiftFromRustBuffer[MetadataFilter](c, rb)
}

func (c FfiConverterTypeMetadataFilter) Read(reader io.Reader) MetadataFilter {
	return MetadataFilter{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeMetadataFilter) Lower(value MetadataFilter) RustBuffer {
	return LowerIntoRustBuffer[MetadataFilter](c, value)
}

func (c FfiConverterTypeMetadataFilter) Write(writer io.Writer, value MetadataFilter) {
	FfiConverterStringINSTANCE.Write(writer, value.JsonPath)
	FfiConverterStringINSTANCE.Write(writer, value.JsonValue)
}

type FfiDestroyerTypeMetadataFilter struct{}

func (_ FfiDestroyerTypeMetadataFilter) Destroy(value MetadataFilter) {
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

type FfiConverterTypeMetadataItem struct{}

var FfiConverterTypeMetadataItemINSTANCE = FfiConverterTypeMetadataItem{}

func (c FfiConverterTypeMetadataItem) Lift(rb RustBufferI) MetadataItem {
	return LiftFromRustBuffer[MetadataItem](c, rb)
}

func (c FfiConverterTypeMetadataItem) Read(reader io.Reader) MetadataItem {
	return MetadataItem{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeMetadataItem) Lower(value MetadataItem) RustBuffer {
	return LowerIntoRustBuffer[MetadataItem](c, value)
}

func (c FfiConverterTypeMetadataItem) Write(writer io.Writer, value MetadataItem) {
	FfiConverterStringINSTANCE.Write(writer, value.Key)
	FfiConverterStringINSTANCE.Write(writer, value.Value)
}

type FfiDestroyerTypeMetadataItem struct{}

func (_ FfiDestroyerTypeMetadataItem) Destroy(value MetadataItem) {
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
	FfiDestroyerSequenceTypeUnspentTransactionOutput{}.Destroy(r.Utxos)
	FfiDestroyerUint64{}.Destroy(r.MaxPayableMsat)
	FfiDestroyerUint64{}.Destroy(r.MaxReceivableMsat)
	FfiDestroyerUint64{}.Destroy(r.MaxSinglePaymentAmountMsat)
	FfiDestroyerUint64{}.Destroy(r.MaxChanReserveMsats)
	FfiDestroyerSequenceString{}.Destroy(r.ConnectedPeers)
	FfiDestroyerUint64{}.Destroy(r.MaxReceivableSinglePaymentAmountMsat)
	FfiDestroyerUint64{}.Destroy(r.TotalInboundLiquidityMsats)
}

type FfiConverterTypeNodeState struct{}

var FfiConverterTypeNodeStateINSTANCE = FfiConverterTypeNodeState{}

func (c FfiConverterTypeNodeState) Lift(rb RustBufferI) NodeState {
	return LiftFromRustBuffer[NodeState](c, rb)
}

func (c FfiConverterTypeNodeState) Read(reader io.Reader) NodeState {
	return NodeState{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSequenceTypeUnspentTransactionOutputINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSequenceStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeNodeState) Lower(value NodeState) RustBuffer {
	return LowerIntoRustBuffer[NodeState](c, value)
}

func (c FfiConverterTypeNodeState) Write(writer io.Writer, value NodeState) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterUint32INSTANCE.Write(writer, value.BlockHeight)
	FfiConverterUint64INSTANCE.Write(writer, value.ChannelsBalanceMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.OnchainBalanceMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.PendingOnchainBalanceMsat)
	FfiConverterSequenceTypeUnspentTransactionOutputINSTANCE.Write(writer, value.Utxos)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxPayableMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxReceivableMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxSinglePaymentAmountMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxChanReserveMsats)
	FfiConverterSequenceStringINSTANCE.Write(writer, value.ConnectedPeers)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxReceivableSinglePaymentAmountMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.TotalInboundLiquidityMsats)
}

type FfiDestroyerTypeNodeState struct{}

func (_ FfiDestroyerTypeNodeState) Destroy(value NodeState) {
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

type FfiConverterTypeOnchainPaymentLimitsResponse struct{}

var FfiConverterTypeOnchainPaymentLimitsResponseINSTANCE = FfiConverterTypeOnchainPaymentLimitsResponse{}

func (c FfiConverterTypeOnchainPaymentLimitsResponse) Lift(rb RustBufferI) OnchainPaymentLimitsResponse {
	return LiftFromRustBuffer[OnchainPaymentLimitsResponse](c, rb)
}

func (c FfiConverterTypeOnchainPaymentLimitsResponse) Read(reader io.Reader) OnchainPaymentLimitsResponse {
	return OnchainPaymentLimitsResponse{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeOnchainPaymentLimitsResponse) Lower(value OnchainPaymentLimitsResponse) RustBuffer {
	return LowerIntoRustBuffer[OnchainPaymentLimitsResponse](c, value)
}

func (c FfiConverterTypeOnchainPaymentLimitsResponse) Write(writer io.Writer, value OnchainPaymentLimitsResponse) {
	FfiConverterUint64INSTANCE.Write(writer, value.MinSat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxSat)
	FfiConverterUint64INSTANCE.Write(writer, value.MaxPayableSat)
}

type FfiDestroyerTypeOnchainPaymentLimitsResponse struct{}

func (_ FfiDestroyerTypeOnchainPaymentLimitsResponse) Destroy(value OnchainPaymentLimitsResponse) {
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

type FfiConverterTypeOpenChannelFeeRequest struct{}

var FfiConverterTypeOpenChannelFeeRequestINSTANCE = FfiConverterTypeOpenChannelFeeRequest{}

func (c FfiConverterTypeOpenChannelFeeRequest) Lift(rb RustBufferI) OpenChannelFeeRequest {
	return LiftFromRustBuffer[OpenChannelFeeRequest](c, rb)
}

func (c FfiConverterTypeOpenChannelFeeRequest) Read(reader io.Reader) OpenChannelFeeRequest {
	return OpenChannelFeeRequest{
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeOpenChannelFeeRequest) Lower(value OpenChannelFeeRequest) RustBuffer {
	return LowerIntoRustBuffer[OpenChannelFeeRequest](c, value)
}

func (c FfiConverterTypeOpenChannelFeeRequest) Write(writer io.Writer, value OpenChannelFeeRequest) {
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Expiry)
}

type FfiDestroyerTypeOpenChannelFeeRequest struct{}

func (_ FfiDestroyerTypeOpenChannelFeeRequest) Destroy(value OpenChannelFeeRequest) {
	value.Destroy()
}

type OpenChannelFeeResponse struct {
	FeeMsat   *uint64
	FeeParams OpeningFeeParams
}

func (r *OpenChannelFeeResponse) Destroy() {
	FfiDestroyerOptionalUint64{}.Destroy(r.FeeMsat)
	FfiDestroyerTypeOpeningFeeParams{}.Destroy(r.FeeParams)
}

type FfiConverterTypeOpenChannelFeeResponse struct{}

var FfiConverterTypeOpenChannelFeeResponseINSTANCE = FfiConverterTypeOpenChannelFeeResponse{}

func (c FfiConverterTypeOpenChannelFeeResponse) Lift(rb RustBufferI) OpenChannelFeeResponse {
	return LiftFromRustBuffer[OpenChannelFeeResponse](c, rb)
}

func (c FfiConverterTypeOpenChannelFeeResponse) Read(reader io.Reader) OpenChannelFeeResponse {
	return OpenChannelFeeResponse{
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterTypeOpeningFeeParamsINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeOpenChannelFeeResponse) Lower(value OpenChannelFeeResponse) RustBuffer {
	return LowerIntoRustBuffer[OpenChannelFeeResponse](c, value)
}

func (c FfiConverterTypeOpenChannelFeeResponse) Write(writer io.Writer, value OpenChannelFeeResponse) {
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.FeeMsat)
	FfiConverterTypeOpeningFeeParamsINSTANCE.Write(writer, value.FeeParams)
}

type FfiDestroyerTypeOpenChannelFeeResponse struct{}

func (_ FfiDestroyerTypeOpenChannelFeeResponse) Destroy(value OpenChannelFeeResponse) {
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

type FfiConverterTypeOpeningFeeParams struct{}

var FfiConverterTypeOpeningFeeParamsINSTANCE = FfiConverterTypeOpeningFeeParams{}

func (c FfiConverterTypeOpeningFeeParams) Lift(rb RustBufferI) OpeningFeeParams {
	return LiftFromRustBuffer[OpeningFeeParams](c, rb)
}

func (c FfiConverterTypeOpeningFeeParams) Read(reader io.Reader) OpeningFeeParams {
	return OpeningFeeParams{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeOpeningFeeParams) Lower(value OpeningFeeParams) RustBuffer {
	return LowerIntoRustBuffer[OpeningFeeParams](c, value)
}

func (c FfiConverterTypeOpeningFeeParams) Write(writer io.Writer, value OpeningFeeParams) {
	FfiConverterUint64INSTANCE.Write(writer, value.MinMsat)
	FfiConverterUint32INSTANCE.Write(writer, value.Proportional)
	FfiConverterStringINSTANCE.Write(writer, value.ValidUntil)
	FfiConverterUint32INSTANCE.Write(writer, value.MaxIdleTime)
	FfiConverterUint32INSTANCE.Write(writer, value.MaxClientToSelfDelay)
	FfiConverterStringINSTANCE.Write(writer, value.Promise)
}

type FfiDestroyerTypeOpeningFeeParams struct{}

func (_ FfiDestroyerTypeOpeningFeeParams) Destroy(value OpeningFeeParams) {
	value.Destroy()
}

type OpeningFeeParamsMenu struct {
	Values []OpeningFeeParams
}

func (r *OpeningFeeParamsMenu) Destroy() {
	FfiDestroyerSequenceTypeOpeningFeeParams{}.Destroy(r.Values)
}

type FfiConverterTypeOpeningFeeParamsMenu struct{}

var FfiConverterTypeOpeningFeeParamsMenuINSTANCE = FfiConverterTypeOpeningFeeParamsMenu{}

func (c FfiConverterTypeOpeningFeeParamsMenu) Lift(rb RustBufferI) OpeningFeeParamsMenu {
	return LiftFromRustBuffer[OpeningFeeParamsMenu](c, rb)
}

func (c FfiConverterTypeOpeningFeeParamsMenu) Read(reader io.Reader) OpeningFeeParamsMenu {
	return OpeningFeeParamsMenu{
		FfiConverterSequenceTypeOpeningFeeParamsINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeOpeningFeeParamsMenu) Lower(value OpeningFeeParamsMenu) RustBuffer {
	return LowerIntoRustBuffer[OpeningFeeParamsMenu](c, value)
}

func (c FfiConverterTypeOpeningFeeParamsMenu) Write(writer io.Writer, value OpeningFeeParamsMenu) {
	FfiConverterSequenceTypeOpeningFeeParamsINSTANCE.Write(writer, value.Values)
}

type FfiDestroyerTypeOpeningFeeParamsMenu struct{}

func (_ FfiDestroyerTypeOpeningFeeParamsMenu) Destroy(value OpeningFeeParamsMenu) {
	value.Destroy()
}

type PayOnchainRequest struct {
	RecipientAddress string
	PrepareRes       PrepareOnchainPaymentResponse
}

func (r *PayOnchainRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.RecipientAddress)
	FfiDestroyerTypePrepareOnchainPaymentResponse{}.Destroy(r.PrepareRes)
}

type FfiConverterTypePayOnchainRequest struct{}

var FfiConverterTypePayOnchainRequestINSTANCE = FfiConverterTypePayOnchainRequest{}

func (c FfiConverterTypePayOnchainRequest) Lift(rb RustBufferI) PayOnchainRequest {
	return LiftFromRustBuffer[PayOnchainRequest](c, rb)
}

func (c FfiConverterTypePayOnchainRequest) Read(reader io.Reader) PayOnchainRequest {
	return PayOnchainRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypePrepareOnchainPaymentResponseINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePayOnchainRequest) Lower(value PayOnchainRequest) RustBuffer {
	return LowerIntoRustBuffer[PayOnchainRequest](c, value)
}

func (c FfiConverterTypePayOnchainRequest) Write(writer io.Writer, value PayOnchainRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.RecipientAddress)
	FfiConverterTypePrepareOnchainPaymentResponseINSTANCE.Write(writer, value.PrepareRes)
}

type FfiDestroyerTypePayOnchainRequest struct{}

func (_ FfiDestroyerTypePayOnchainRequest) Destroy(value PayOnchainRequest) {
	value.Destroy()
}

type PayOnchainResponse struct {
	ReverseSwapInfo ReverseSwapInfo
}

func (r *PayOnchainResponse) Destroy() {
	FfiDestroyerTypeReverseSwapInfo{}.Destroy(r.ReverseSwapInfo)
}

type FfiConverterTypePayOnchainResponse struct{}

var FfiConverterTypePayOnchainResponseINSTANCE = FfiConverterTypePayOnchainResponse{}

func (c FfiConverterTypePayOnchainResponse) Lift(rb RustBufferI) PayOnchainResponse {
	return LiftFromRustBuffer[PayOnchainResponse](c, rb)
}

func (c FfiConverterTypePayOnchainResponse) Read(reader io.Reader) PayOnchainResponse {
	return PayOnchainResponse{
		FfiConverterTypeReverseSwapInfoINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePayOnchainResponse) Lower(value PayOnchainResponse) RustBuffer {
	return LowerIntoRustBuffer[PayOnchainResponse](c, value)
}

func (c FfiConverterTypePayOnchainResponse) Write(writer io.Writer, value PayOnchainResponse) {
	FfiConverterTypeReverseSwapInfoINSTANCE.Write(writer, value.ReverseSwapInfo)
}

type FfiDestroyerTypePayOnchainResponse struct{}

func (_ FfiDestroyerTypePayOnchainResponse) Destroy(value PayOnchainResponse) {
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
	FfiDestroyerTypePaymentType{}.Destroy(r.PaymentType)
	FfiDestroyerInt64{}.Destroy(r.PaymentTime)
	FfiDestroyerUint64{}.Destroy(r.AmountMsat)
	FfiDestroyerUint64{}.Destroy(r.FeeMsat)
	FfiDestroyerTypePaymentStatus{}.Destroy(r.Status)
	FfiDestroyerOptionalString{}.Destroy(r.Error)
	FfiDestroyerOptionalString{}.Destroy(r.Description)
	FfiDestroyerTypePaymentDetails{}.Destroy(r.Details)
	FfiDestroyerOptionalString{}.Destroy(r.Metadata)
}

type FfiConverterTypePayment struct{}

var FfiConverterTypePaymentINSTANCE = FfiConverterTypePayment{}

func (c FfiConverterTypePayment) Lift(rb RustBufferI) Payment {
	return LiftFromRustBuffer[Payment](c, rb)
}

func (c FfiConverterTypePayment) Read(reader io.Reader) Payment {
	return Payment{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterTypePaymentTypeINSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterTypePaymentStatusINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterTypePaymentDetailsINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePayment) Lower(value Payment) RustBuffer {
	return LowerIntoRustBuffer[Payment](c, value)
}

func (c FfiConverterTypePayment) Write(writer io.Writer, value Payment) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterTypePaymentTypeINSTANCE.Write(writer, value.PaymentType)
	FfiConverterInt64INSTANCE.Write(writer, value.PaymentTime)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterUint64INSTANCE.Write(writer, value.FeeMsat)
	FfiConverterTypePaymentStatusINSTANCE.Write(writer, value.Status)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Error)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Description)
	FfiConverterTypePaymentDetailsINSTANCE.Write(writer, value.Details)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Metadata)
}

type FfiDestroyerTypePayment struct{}

func (_ FfiDestroyerTypePayment) Destroy(value Payment) {
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
	FfiDestroyerOptionalTypeLnInvoice{}.Destroy(r.Invoice)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
}

type FfiConverterTypePaymentFailedData struct{}

var FfiConverterTypePaymentFailedDataINSTANCE = FfiConverterTypePaymentFailedData{}

func (c FfiConverterTypePaymentFailedData) Lift(rb RustBufferI) PaymentFailedData {
	return LiftFromRustBuffer[PaymentFailedData](c, rb)
}

func (c FfiConverterTypePaymentFailedData) Read(reader io.Reader) PaymentFailedData {
	return PaymentFailedData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalTypeLNInvoiceINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePaymentFailedData) Lower(value PaymentFailedData) RustBuffer {
	return LowerIntoRustBuffer[PaymentFailedData](c, value)
}

func (c FfiConverterTypePaymentFailedData) Write(writer io.Writer, value PaymentFailedData) {
	FfiConverterStringINSTANCE.Write(writer, value.Error)
	FfiConverterStringINSTANCE.Write(writer, value.NodeId)
	FfiConverterOptionalTypeLNInvoiceINSTANCE.Write(writer, value.Invoice)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
}

type FfiDestroyerTypePaymentFailedData struct{}

func (_ FfiDestroyerTypePaymentFailedData) Destroy(value PaymentFailedData) {
	value.Destroy()
}

type PrepareOnchainPaymentRequest struct {
	AmountSat      uint64
	AmountType     SwapAmountType
	ClaimTxFeerate uint32
}

func (r *PrepareOnchainPaymentRequest) Destroy() {
	FfiDestroyerUint64{}.Destroy(r.AmountSat)
	FfiDestroyerTypeSwapAmountType{}.Destroy(r.AmountType)
	FfiDestroyerUint32{}.Destroy(r.ClaimTxFeerate)
}

type FfiConverterTypePrepareOnchainPaymentRequest struct{}

var FfiConverterTypePrepareOnchainPaymentRequestINSTANCE = FfiConverterTypePrepareOnchainPaymentRequest{}

func (c FfiConverterTypePrepareOnchainPaymentRequest) Lift(rb RustBufferI) PrepareOnchainPaymentRequest {
	return LiftFromRustBuffer[PrepareOnchainPaymentRequest](c, rb)
}

func (c FfiConverterTypePrepareOnchainPaymentRequest) Read(reader io.Reader) PrepareOnchainPaymentRequest {
	return PrepareOnchainPaymentRequest{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterTypeSwapAmountTypeINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareOnchainPaymentRequest) Lower(value PrepareOnchainPaymentRequest) RustBuffer {
	return LowerIntoRustBuffer[PrepareOnchainPaymentRequest](c, value)
}

func (c FfiConverterTypePrepareOnchainPaymentRequest) Write(writer io.Writer, value PrepareOnchainPaymentRequest) {
	FfiConverterUint64INSTANCE.Write(writer, value.AmountSat)
	FfiConverterTypeSwapAmountTypeINSTANCE.Write(writer, value.AmountType)
	FfiConverterUint32INSTANCE.Write(writer, value.ClaimTxFeerate)
}

type FfiDestroyerTypePrepareOnchainPaymentRequest struct{}

func (_ FfiDestroyerTypePrepareOnchainPaymentRequest) Destroy(value PrepareOnchainPaymentRequest) {
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

type FfiConverterTypePrepareOnchainPaymentResponse struct{}

var FfiConverterTypePrepareOnchainPaymentResponseINSTANCE = FfiConverterTypePrepareOnchainPaymentResponse{}

func (c FfiConverterTypePrepareOnchainPaymentResponse) Lift(rb RustBufferI) PrepareOnchainPaymentResponse {
	return LiftFromRustBuffer[PrepareOnchainPaymentResponse](c, rb)
}

func (c FfiConverterTypePrepareOnchainPaymentResponse) Read(reader io.Reader) PrepareOnchainPaymentResponse {
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

func (c FfiConverterTypePrepareOnchainPaymentResponse) Lower(value PrepareOnchainPaymentResponse) RustBuffer {
	return LowerIntoRustBuffer[PrepareOnchainPaymentResponse](c, value)
}

func (c FfiConverterTypePrepareOnchainPaymentResponse) Write(writer io.Writer, value PrepareOnchainPaymentResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.FeesHash)
	FfiConverterFloat64INSTANCE.Write(writer, value.FeesPercentage)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesLockup)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesClaim)
	FfiConverterUint64INSTANCE.Write(writer, value.SenderAmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.RecipientAmountSat)
	FfiConverterUint64INSTANCE.Write(writer, value.TotalFees)
}

type FfiDestroyerTypePrepareOnchainPaymentResponse struct{}

func (_ FfiDestroyerTypePrepareOnchainPaymentResponse) Destroy(value PrepareOnchainPaymentResponse) {
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

type FfiConverterTypePrepareRedeemOnchainFundsRequest struct{}

var FfiConverterTypePrepareRedeemOnchainFundsRequestINSTANCE = FfiConverterTypePrepareRedeemOnchainFundsRequest{}

func (c FfiConverterTypePrepareRedeemOnchainFundsRequest) Lift(rb RustBufferI) PrepareRedeemOnchainFundsRequest {
	return LiftFromRustBuffer[PrepareRedeemOnchainFundsRequest](c, rb)
}

func (c FfiConverterTypePrepareRedeemOnchainFundsRequest) Read(reader io.Reader) PrepareRedeemOnchainFundsRequest {
	return PrepareRedeemOnchainFundsRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareRedeemOnchainFundsRequest) Lower(value PrepareRedeemOnchainFundsRequest) RustBuffer {
	return LowerIntoRustBuffer[PrepareRedeemOnchainFundsRequest](c, value)
}

func (c FfiConverterTypePrepareRedeemOnchainFundsRequest) Write(writer io.Writer, value PrepareRedeemOnchainFundsRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.ToAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.SatPerVbyte)
}

type FfiDestroyerTypePrepareRedeemOnchainFundsRequest struct{}

func (_ FfiDestroyerTypePrepareRedeemOnchainFundsRequest) Destroy(value PrepareRedeemOnchainFundsRequest) {
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

type FfiConverterTypePrepareRedeemOnchainFundsResponse struct{}

var FfiConverterTypePrepareRedeemOnchainFundsResponseINSTANCE = FfiConverterTypePrepareRedeemOnchainFundsResponse{}

func (c FfiConverterTypePrepareRedeemOnchainFundsResponse) Lift(rb RustBufferI) PrepareRedeemOnchainFundsResponse {
	return LiftFromRustBuffer[PrepareRedeemOnchainFundsResponse](c, rb)
}

func (c FfiConverterTypePrepareRedeemOnchainFundsResponse) Read(reader io.Reader) PrepareRedeemOnchainFundsResponse {
	return PrepareRedeemOnchainFundsResponse{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareRedeemOnchainFundsResponse) Lower(value PrepareRedeemOnchainFundsResponse) RustBuffer {
	return LowerIntoRustBuffer[PrepareRedeemOnchainFundsResponse](c, value)
}

func (c FfiConverterTypePrepareRedeemOnchainFundsResponse) Write(writer io.Writer, value PrepareRedeemOnchainFundsResponse) {
	FfiConverterUint64INSTANCE.Write(writer, value.TxWeight)
	FfiConverterUint64INSTANCE.Write(writer, value.TxFeeSat)
}

type FfiDestroyerTypePrepareRedeemOnchainFundsResponse struct{}

func (_ FfiDestroyerTypePrepareRedeemOnchainFundsResponse) Destroy(value PrepareRedeemOnchainFundsResponse) {
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

type FfiConverterTypePrepareRefundRequest struct{}

var FfiConverterTypePrepareRefundRequestINSTANCE = FfiConverterTypePrepareRefundRequest{}

func (c FfiConverterTypePrepareRefundRequest) Lift(rb RustBufferI) PrepareRefundRequest {
	return LiftFromRustBuffer[PrepareRefundRequest](c, rb)
}

func (c FfiConverterTypePrepareRefundRequest) Read(reader io.Reader) PrepareRefundRequest {
	return PrepareRefundRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareRefundRequest) Lower(value PrepareRefundRequest) RustBuffer {
	return LowerIntoRustBuffer[PrepareRefundRequest](c, value)
}

func (c FfiConverterTypePrepareRefundRequest) Write(writer io.Writer, value PrepareRefundRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapAddress)
	FfiConverterStringINSTANCE.Write(writer, value.ToAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.SatPerVbyte)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.Unilateral)
}

type FfiDestroyerTypePrepareRefundRequest struct{}

func (_ FfiDestroyerTypePrepareRefundRequest) Destroy(value PrepareRefundRequest) {
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

type FfiConverterTypePrepareRefundResponse struct{}

var FfiConverterTypePrepareRefundResponseINSTANCE = FfiConverterTypePrepareRefundResponse{}

func (c FfiConverterTypePrepareRefundResponse) Lift(rb RustBufferI) PrepareRefundResponse {
	return LiftFromRustBuffer[PrepareRefundResponse](c, rb)
}

func (c FfiConverterTypePrepareRefundResponse) Read(reader io.Reader) PrepareRefundResponse {
	return PrepareRefundResponse{
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypePrepareRefundResponse) Lower(value PrepareRefundResponse) RustBuffer {
	return LowerIntoRustBuffer[PrepareRefundResponse](c, value)
}

func (c FfiConverterTypePrepareRefundResponse) Write(writer io.Writer, value PrepareRefundResponse) {
	FfiConverterUint32INSTANCE.Write(writer, value.RefundTxWeight)
	FfiConverterUint64INSTANCE.Write(writer, value.RefundTxFeeSat)
}

type FfiDestroyerTypePrepareRefundResponse struct{}

func (_ FfiDestroyerTypePrepareRefundResponse) Destroy(value PrepareRefundResponse) {
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

type FfiConverterTypeRate struct{}

var FfiConverterTypeRateINSTANCE = FfiConverterTypeRate{}

func (c FfiConverterTypeRate) Lift(rb RustBufferI) Rate {
	return LiftFromRustBuffer[Rate](c, rb)
}

func (c FfiConverterTypeRate) Read(reader io.Reader) Rate {
	return Rate{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterFloat64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRate) Lower(value Rate) RustBuffer {
	return LowerIntoRustBuffer[Rate](c, value)
}

func (c FfiConverterTypeRate) Write(writer io.Writer, value Rate) {
	FfiConverterStringINSTANCE.Write(writer, value.Coin)
	FfiConverterFloat64INSTANCE.Write(writer, value.Value)
}

type FfiDestroyerTypeRate struct{}

func (_ FfiDestroyerTypeRate) Destroy(value Rate) {
	value.Destroy()
}

type ReceiveOnchainRequest struct {
	OpeningFeeParams *OpeningFeeParams
}

func (r *ReceiveOnchainRequest) Destroy() {
	FfiDestroyerOptionalTypeOpeningFeeParams{}.Destroy(r.OpeningFeeParams)
}

type FfiConverterTypeReceiveOnchainRequest struct{}

var FfiConverterTypeReceiveOnchainRequestINSTANCE = FfiConverterTypeReceiveOnchainRequest{}

func (c FfiConverterTypeReceiveOnchainRequest) Lift(rb RustBufferI) ReceiveOnchainRequest {
	return LiftFromRustBuffer[ReceiveOnchainRequest](c, rb)
}

func (c FfiConverterTypeReceiveOnchainRequest) Read(reader io.Reader) ReceiveOnchainRequest {
	return ReceiveOnchainRequest{
		FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeReceiveOnchainRequest) Lower(value ReceiveOnchainRequest) RustBuffer {
	return LowerIntoRustBuffer[ReceiveOnchainRequest](c, value)
}

func (c FfiConverterTypeReceiveOnchainRequest) Write(writer io.Writer, value ReceiveOnchainRequest) {
	FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Write(writer, value.OpeningFeeParams)
}

type FfiDestroyerTypeReceiveOnchainRequest struct{}

func (_ FfiDestroyerTypeReceiveOnchainRequest) Destroy(value ReceiveOnchainRequest) {
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
	FfiDestroyerOptionalTypeOpeningFeeParams{}.Destroy(r.OpeningFeeParams)
	FfiDestroyerOptionalBool{}.Destroy(r.UseDescriptionHash)
	FfiDestroyerOptionalUint32{}.Destroy(r.Expiry)
	FfiDestroyerOptionalUint32{}.Destroy(r.Cltv)
}

type FfiConverterTypeReceivePaymentRequest struct{}

var FfiConverterTypeReceivePaymentRequestINSTANCE = FfiConverterTypeReceivePaymentRequest{}

func (c FfiConverterTypeReceivePaymentRequest) Lift(rb RustBufferI) ReceivePaymentRequest {
	return LiftFromRustBuffer[ReceivePaymentRequest](c, rb)
}

func (c FfiConverterTypeReceivePaymentRequest) Read(reader io.Reader) ReceivePaymentRequest {
	return ReceivePaymentRequest{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalSequenceUint8INSTANCE.Read(reader),
		FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeReceivePaymentRequest) Lower(value ReceivePaymentRequest) RustBuffer {
	return LowerIntoRustBuffer[ReceivePaymentRequest](c, value)
}

func (c FfiConverterTypeReceivePaymentRequest) Write(writer io.Writer, value ReceivePaymentRequest) {
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterOptionalSequenceUint8INSTANCE.Write(writer, value.Preimage)
	FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Write(writer, value.OpeningFeeParams)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.UseDescriptionHash)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Expiry)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Cltv)
}

type FfiDestroyerTypeReceivePaymentRequest struct{}

func (_ FfiDestroyerTypeReceivePaymentRequest) Destroy(value ReceivePaymentRequest) {
	value.Destroy()
}

type ReceivePaymentResponse struct {
	LnInvoice        LnInvoice
	OpeningFeeParams *OpeningFeeParams
	OpeningFeeMsat   *uint64
}

func (r *ReceivePaymentResponse) Destroy() {
	FfiDestroyerTypeLnInvoice{}.Destroy(r.LnInvoice)
	FfiDestroyerOptionalTypeOpeningFeeParams{}.Destroy(r.OpeningFeeParams)
	FfiDestroyerOptionalUint64{}.Destroy(r.OpeningFeeMsat)
}

type FfiConverterTypeReceivePaymentResponse struct{}

var FfiConverterTypeReceivePaymentResponseINSTANCE = FfiConverterTypeReceivePaymentResponse{}

func (c FfiConverterTypeReceivePaymentResponse) Lift(rb RustBufferI) ReceivePaymentResponse {
	return LiftFromRustBuffer[ReceivePaymentResponse](c, rb)
}

func (c FfiConverterTypeReceivePaymentResponse) Read(reader io.Reader) ReceivePaymentResponse {
	return ReceivePaymentResponse{
		FfiConverterTypeLNInvoiceINSTANCE.Read(reader),
		FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeReceivePaymentResponse) Lower(value ReceivePaymentResponse) RustBuffer {
	return LowerIntoRustBuffer[ReceivePaymentResponse](c, value)
}

func (c FfiConverterTypeReceivePaymentResponse) Write(writer io.Writer, value ReceivePaymentResponse) {
	FfiConverterTypeLNInvoiceINSTANCE.Write(writer, value.LnInvoice)
	FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Write(writer, value.OpeningFeeParams)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.OpeningFeeMsat)
}

type FfiDestroyerTypeReceivePaymentResponse struct{}

func (_ FfiDestroyerTypeReceivePaymentResponse) Destroy(value ReceivePaymentResponse) {
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

type FfiConverterTypeRecommendedFees struct{}

var FfiConverterTypeRecommendedFeesINSTANCE = FfiConverterTypeRecommendedFees{}

func (c FfiConverterTypeRecommendedFees) Lift(rb RustBufferI) RecommendedFees {
	return LiftFromRustBuffer[RecommendedFees](c, rb)
}

func (c FfiConverterTypeRecommendedFees) Read(reader io.Reader) RecommendedFees {
	return RecommendedFees{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRecommendedFees) Lower(value RecommendedFees) RustBuffer {
	return LowerIntoRustBuffer[RecommendedFees](c, value)
}

func (c FfiConverterTypeRecommendedFees) Write(writer io.Writer, value RecommendedFees) {
	FfiConverterUint64INSTANCE.Write(writer, value.FastestFee)
	FfiConverterUint64INSTANCE.Write(writer, value.HalfHourFee)
	FfiConverterUint64INSTANCE.Write(writer, value.HourFee)
	FfiConverterUint64INSTANCE.Write(writer, value.EconomyFee)
	FfiConverterUint64INSTANCE.Write(writer, value.MinimumFee)
}

type FfiDestroyerTypeRecommendedFees struct{}

func (_ FfiDestroyerTypeRecommendedFees) Destroy(value RecommendedFees) {
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

type FfiConverterTypeRedeemOnchainFundsRequest struct{}

var FfiConverterTypeRedeemOnchainFundsRequestINSTANCE = FfiConverterTypeRedeemOnchainFundsRequest{}

func (c FfiConverterTypeRedeemOnchainFundsRequest) Lift(rb RustBufferI) RedeemOnchainFundsRequest {
	return LiftFromRustBuffer[RedeemOnchainFundsRequest](c, rb)
}

func (c FfiConverterTypeRedeemOnchainFundsRequest) Read(reader io.Reader) RedeemOnchainFundsRequest {
	return RedeemOnchainFundsRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRedeemOnchainFundsRequest) Lower(value RedeemOnchainFundsRequest) RustBuffer {
	return LowerIntoRustBuffer[RedeemOnchainFundsRequest](c, value)
}

func (c FfiConverterTypeRedeemOnchainFundsRequest) Write(writer io.Writer, value RedeemOnchainFundsRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.ToAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.SatPerVbyte)
}

type FfiDestroyerTypeRedeemOnchainFundsRequest struct{}

func (_ FfiDestroyerTypeRedeemOnchainFundsRequest) Destroy(value RedeemOnchainFundsRequest) {
	value.Destroy()
}

type RedeemOnchainFundsResponse struct {
	Txid []uint8
}

func (r *RedeemOnchainFundsResponse) Destroy() {
	FfiDestroyerSequenceUint8{}.Destroy(r.Txid)
}

type FfiConverterTypeRedeemOnchainFundsResponse struct{}

var FfiConverterTypeRedeemOnchainFundsResponseINSTANCE = FfiConverterTypeRedeemOnchainFundsResponse{}

func (c FfiConverterTypeRedeemOnchainFundsResponse) Lift(rb RustBufferI) RedeemOnchainFundsResponse {
	return LiftFromRustBuffer[RedeemOnchainFundsResponse](c, rb)
}

func (c FfiConverterTypeRedeemOnchainFundsResponse) Read(reader io.Reader) RedeemOnchainFundsResponse {
	return RedeemOnchainFundsResponse{
		FfiConverterSequenceUint8INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRedeemOnchainFundsResponse) Lower(value RedeemOnchainFundsResponse) RustBuffer {
	return LowerIntoRustBuffer[RedeemOnchainFundsResponse](c, value)
}

func (c FfiConverterTypeRedeemOnchainFundsResponse) Write(writer io.Writer, value RedeemOnchainFundsResponse) {
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Txid)
}

type FfiDestroyerTypeRedeemOnchainFundsResponse struct{}

func (_ FfiDestroyerTypeRedeemOnchainFundsResponse) Destroy(value RedeemOnchainFundsResponse) {
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

type FfiConverterTypeRefundRequest struct{}

var FfiConverterTypeRefundRequestINSTANCE = FfiConverterTypeRefundRequest{}

func (c FfiConverterTypeRefundRequest) Lift(rb RustBufferI) RefundRequest {
	return LiftFromRustBuffer[RefundRequest](c, rb)
}

func (c FfiConverterTypeRefundRequest) Read(reader io.Reader) RefundRequest {
	return RefundRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRefundRequest) Lower(value RefundRequest) RustBuffer {
	return LowerIntoRustBuffer[RefundRequest](c, value)
}

func (c FfiConverterTypeRefundRequest) Write(writer io.Writer, value RefundRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.SwapAddress)
	FfiConverterStringINSTANCE.Write(writer, value.ToAddress)
	FfiConverterUint32INSTANCE.Write(writer, value.SatPerVbyte)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.Unilateral)
}

type FfiDestroyerTypeRefundRequest struct{}

func (_ FfiDestroyerTypeRefundRequest) Destroy(value RefundRequest) {
	value.Destroy()
}

type RefundResponse struct {
	RefundTxId string
}

func (r *RefundResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.RefundTxId)
}

type FfiConverterTypeRefundResponse struct{}

var FfiConverterTypeRefundResponseINSTANCE = FfiConverterTypeRefundResponse{}

func (c FfiConverterTypeRefundResponse) Lift(rb RustBufferI) RefundResponse {
	return LiftFromRustBuffer[RefundResponse](c, rb)
}

func (c FfiConverterTypeRefundResponse) Read(reader io.Reader) RefundResponse {
	return RefundResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRefundResponse) Lower(value RefundResponse) RustBuffer {
	return LowerIntoRustBuffer[RefundResponse](c, value)
}

func (c FfiConverterTypeRefundResponse) Write(writer io.Writer, value RefundResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.RefundTxId)
}

type FfiDestroyerTypeRefundResponse struct{}

func (_ FfiDestroyerTypeRefundResponse) Destroy(value RefundResponse) {
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

type FfiConverterTypeReportPaymentFailureDetails struct{}

var FfiConverterTypeReportPaymentFailureDetailsINSTANCE = FfiConverterTypeReportPaymentFailureDetails{}

func (c FfiConverterTypeReportPaymentFailureDetails) Lift(rb RustBufferI) ReportPaymentFailureDetails {
	return LiftFromRustBuffer[ReportPaymentFailureDetails](c, rb)
}

func (c FfiConverterTypeReportPaymentFailureDetails) Read(reader io.Reader) ReportPaymentFailureDetails {
	return ReportPaymentFailureDetails{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeReportPaymentFailureDetails) Lower(value ReportPaymentFailureDetails) RustBuffer {
	return LowerIntoRustBuffer[ReportPaymentFailureDetails](c, value)
}

func (c FfiConverterTypeReportPaymentFailureDetails) Write(writer io.Writer, value ReportPaymentFailureDetails) {
	FfiConverterStringINSTANCE.Write(writer, value.PaymentHash)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Comment)
}

type FfiDestroyerTypeReportPaymentFailureDetails struct{}

func (_ FfiDestroyerTypeReportPaymentFailureDetails) Destroy(value ReportPaymentFailureDetails) {
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

type FfiConverterTypeReverseSwapFeesRequest struct{}

var FfiConverterTypeReverseSwapFeesRequestINSTANCE = FfiConverterTypeReverseSwapFeesRequest{}

func (c FfiConverterTypeReverseSwapFeesRequest) Lift(rb RustBufferI) ReverseSwapFeesRequest {
	return LiftFromRustBuffer[ReverseSwapFeesRequest](c, rb)
}

func (c FfiConverterTypeReverseSwapFeesRequest) Read(reader io.Reader) ReverseSwapFeesRequest {
	return ReverseSwapFeesRequest{
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeReverseSwapFeesRequest) Lower(value ReverseSwapFeesRequest) RustBuffer {
	return LowerIntoRustBuffer[ReverseSwapFeesRequest](c, value)
}

func (c FfiConverterTypeReverseSwapFeesRequest) Write(writer io.Writer, value ReverseSwapFeesRequest) {
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.SendAmountSat)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.ClaimTxFeerate)
}

type FfiDestroyerTypeReverseSwapFeesRequest struct{}

func (_ FfiDestroyerTypeReverseSwapFeesRequest) Destroy(value ReverseSwapFeesRequest) {
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
	FfiDestroyerTypeReverseSwapStatus{}.Destroy(r.Status)
}

type FfiConverterTypeReverseSwapInfo struct{}

var FfiConverterTypeReverseSwapInfoINSTANCE = FfiConverterTypeReverseSwapInfo{}

func (c FfiConverterTypeReverseSwapInfo) Lift(rb RustBufferI) ReverseSwapInfo {
	return LiftFromRustBuffer[ReverseSwapInfo](c, rb)
}

func (c FfiConverterTypeReverseSwapInfo) Read(reader io.Reader) ReverseSwapInfo {
	return ReverseSwapInfo{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterTypeReverseSwapStatusINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeReverseSwapInfo) Lower(value ReverseSwapInfo) RustBuffer {
	return LowerIntoRustBuffer[ReverseSwapInfo](c, value)
}

func (c FfiConverterTypeReverseSwapInfo) Write(writer io.Writer, value ReverseSwapInfo) {
	FfiConverterStringINSTANCE.Write(writer, value.Id)
	FfiConverterStringINSTANCE.Write(writer, value.ClaimPubkey)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LockupTxid)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.ClaimTxid)
	FfiConverterUint64INSTANCE.Write(writer, value.OnchainAmountSat)
	FfiConverterTypeReverseSwapStatusINSTANCE.Write(writer, value.Status)
}

type FfiDestroyerTypeReverseSwapInfo struct{}

func (_ FfiDestroyerTypeReverseSwapInfo) Destroy(value ReverseSwapInfo) {
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

type FfiConverterTypeReverseSwapPairInfo struct{}

var FfiConverterTypeReverseSwapPairInfoINSTANCE = FfiConverterTypeReverseSwapPairInfo{}

func (c FfiConverterTypeReverseSwapPairInfo) Lift(rb RustBufferI) ReverseSwapPairInfo {
	return LiftFromRustBuffer[ReverseSwapPairInfo](c, rb)
}

func (c FfiConverterTypeReverseSwapPairInfo) Read(reader io.Reader) ReverseSwapPairInfo {
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

func (c FfiConverterTypeReverseSwapPairInfo) Lower(value ReverseSwapPairInfo) RustBuffer {
	return LowerIntoRustBuffer[ReverseSwapPairInfo](c, value)
}

func (c FfiConverterTypeReverseSwapPairInfo) Write(writer io.Writer, value ReverseSwapPairInfo) {
	FfiConverterUint64INSTANCE.Write(writer, value.Min)
	FfiConverterUint64INSTANCE.Write(writer, value.Max)
	FfiConverterStringINSTANCE.Write(writer, value.FeesHash)
	FfiConverterFloat64INSTANCE.Write(writer, value.FeesPercentage)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesLockup)
	FfiConverterUint64INSTANCE.Write(writer, value.FeesClaim)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.TotalFees)
}

type FfiDestroyerTypeReverseSwapPairInfo struct{}

func (_ FfiDestroyerTypeReverseSwapPairInfo) Destroy(value ReverseSwapPairInfo) {
	value.Destroy()
}

type RouteHint struct {
	Hops []RouteHintHop
}

func (r *RouteHint) Destroy() {
	FfiDestroyerSequenceTypeRouteHintHop{}.Destroy(r.Hops)
}

type FfiConverterTypeRouteHint struct{}

var FfiConverterTypeRouteHintINSTANCE = FfiConverterTypeRouteHint{}

func (c FfiConverterTypeRouteHint) Lift(rb RustBufferI) RouteHint {
	return LiftFromRustBuffer[RouteHint](c, rb)
}

func (c FfiConverterTypeRouteHint) Read(reader io.Reader) RouteHint {
	return RouteHint{
		FfiConverterSequenceTypeRouteHintHopINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeRouteHint) Lower(value RouteHint) RustBuffer {
	return LowerIntoRustBuffer[RouteHint](c, value)
}

func (c FfiConverterTypeRouteHint) Write(writer io.Writer, value RouteHint) {
	FfiConverterSequenceTypeRouteHintHopINSTANCE.Write(writer, value.Hops)
}

type FfiDestroyerTypeRouteHint struct{}

func (_ FfiDestroyerTypeRouteHint) Destroy(value RouteHint) {
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

type FfiConverterTypeRouteHintHop struct{}

var FfiConverterTypeRouteHintHopINSTANCE = FfiConverterTypeRouteHintHop{}

func (c FfiConverterTypeRouteHintHop) Lift(rb RustBufferI) RouteHintHop {
	return LiftFromRustBuffer[RouteHintHop](c, rb)
}

func (c FfiConverterTypeRouteHintHop) Read(reader io.Reader) RouteHintHop {
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

func (c FfiConverterTypeRouteHintHop) Lower(value RouteHintHop) RustBuffer {
	return LowerIntoRustBuffer[RouteHintHop](c, value)
}

func (c FfiConverterTypeRouteHintHop) Write(writer io.Writer, value RouteHintHop) {
	FfiConverterStringINSTANCE.Write(writer, value.SrcNodeId)
	FfiConverterStringINSTANCE.Write(writer, value.ShortChannelId)
	FfiConverterUint32INSTANCE.Write(writer, value.FeesBaseMsat)
	FfiConverterUint32INSTANCE.Write(writer, value.FeesProportionalMillionths)
	FfiConverterUint64INSTANCE.Write(writer, value.CltvExpiryDelta)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.HtlcMinimumMsat)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.HtlcMaximumMsat)
}

type FfiDestroyerTypeRouteHintHop struct{}

func (_ FfiDestroyerTypeRouteHintHop) Destroy(value RouteHintHop) {
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

type FfiConverterTypeSendPaymentRequest struct{}

var FfiConverterTypeSendPaymentRequestINSTANCE = FfiConverterTypeSendPaymentRequest{}

func (c FfiConverterTypeSendPaymentRequest) Lift(rb RustBufferI) SendPaymentRequest {
	return LiftFromRustBuffer[SendPaymentRequest](c, rb)
}

func (c FfiConverterTypeSendPaymentRequest) Read(reader io.Reader) SendPaymentRequest {
	return SendPaymentRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSendPaymentRequest) Lower(value SendPaymentRequest) RustBuffer {
	return LowerIntoRustBuffer[SendPaymentRequest](c, value)
}

func (c FfiConverterTypeSendPaymentRequest) Write(writer io.Writer, value SendPaymentRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Bolt11)
	FfiConverterBoolINSTANCE.Write(writer, value.UseTrampoline)
	FfiConverterOptionalUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
}

type FfiDestroyerTypeSendPaymentRequest struct{}

func (_ FfiDestroyerTypeSendPaymentRequest) Destroy(value SendPaymentRequest) {
	value.Destroy()
}

type SendPaymentResponse struct {
	Payment Payment
}

func (r *SendPaymentResponse) Destroy() {
	FfiDestroyerTypePayment{}.Destroy(r.Payment)
}

type FfiConverterTypeSendPaymentResponse struct{}

var FfiConverterTypeSendPaymentResponseINSTANCE = FfiConverterTypeSendPaymentResponse{}

func (c FfiConverterTypeSendPaymentResponse) Lift(rb RustBufferI) SendPaymentResponse {
	return LiftFromRustBuffer[SendPaymentResponse](c, rb)
}

func (c FfiConverterTypeSendPaymentResponse) Read(reader io.Reader) SendPaymentResponse {
	return SendPaymentResponse{
		FfiConverterTypePaymentINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSendPaymentResponse) Lower(value SendPaymentResponse) RustBuffer {
	return LowerIntoRustBuffer[SendPaymentResponse](c, value)
}

func (c FfiConverterTypeSendPaymentResponse) Write(writer io.Writer, value SendPaymentResponse) {
	FfiConverterTypePaymentINSTANCE.Write(writer, value.Payment)
}

type FfiDestroyerTypeSendPaymentResponse struct{}

func (_ FfiDestroyerTypeSendPaymentResponse) Destroy(value SendPaymentResponse) {
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
	FfiDestroyerOptionalSequenceTypeTlvEntry{}.Destroy(r.ExtraTlvs)
	FfiDestroyerOptionalString{}.Destroy(r.Label)
}

type FfiConverterTypeSendSpontaneousPaymentRequest struct{}

var FfiConverterTypeSendSpontaneousPaymentRequestINSTANCE = FfiConverterTypeSendSpontaneousPaymentRequest{}

func (c FfiConverterTypeSendSpontaneousPaymentRequest) Lift(rb RustBufferI) SendSpontaneousPaymentRequest {
	return LiftFromRustBuffer[SendSpontaneousPaymentRequest](c, rb)
}

func (c FfiConverterTypeSendSpontaneousPaymentRequest) Read(reader io.Reader) SendSpontaneousPaymentRequest {
	return SendSpontaneousPaymentRequest{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterOptionalSequenceTypeTlvEntryINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSendSpontaneousPaymentRequest) Lower(value SendSpontaneousPaymentRequest) RustBuffer {
	return LowerIntoRustBuffer[SendSpontaneousPaymentRequest](c, value)
}

func (c FfiConverterTypeSendSpontaneousPaymentRequest) Write(writer io.Writer, value SendSpontaneousPaymentRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.NodeId)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMsat)
	FfiConverterOptionalSequenceTypeTlvEntryINSTANCE.Write(writer, value.ExtraTlvs)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Label)
}

type FfiDestroyerTypeSendSpontaneousPaymentRequest struct{}

func (_ FfiDestroyerTypeSendSpontaneousPaymentRequest) Destroy(value SendSpontaneousPaymentRequest) {
	value.Destroy()
}

type ServiceHealthCheckResponse struct {
	Status HealthCheckStatus
}

func (r *ServiceHealthCheckResponse) Destroy() {
	FfiDestroyerTypeHealthCheckStatus{}.Destroy(r.Status)
}

type FfiConverterTypeServiceHealthCheckResponse struct{}

var FfiConverterTypeServiceHealthCheckResponseINSTANCE = FfiConverterTypeServiceHealthCheckResponse{}

func (c FfiConverterTypeServiceHealthCheckResponse) Lift(rb RustBufferI) ServiceHealthCheckResponse {
	return LiftFromRustBuffer[ServiceHealthCheckResponse](c, rb)
}

func (c FfiConverterTypeServiceHealthCheckResponse) Read(reader io.Reader) ServiceHealthCheckResponse {
	return ServiceHealthCheckResponse{
		FfiConverterTypeHealthCheckStatusINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeServiceHealthCheckResponse) Lower(value ServiceHealthCheckResponse) RustBuffer {
	return LowerIntoRustBuffer[ServiceHealthCheckResponse](c, value)
}

func (c FfiConverterTypeServiceHealthCheckResponse) Write(writer io.Writer, value ServiceHealthCheckResponse) {
	FfiConverterTypeHealthCheckStatusINSTANCE.Write(writer, value.Status)
}

type FfiDestroyerTypeServiceHealthCheckResponse struct{}

func (_ FfiDestroyerTypeServiceHealthCheckResponse) Destroy(value ServiceHealthCheckResponse) {
	value.Destroy()
}

type SignMessageRequest struct {
	Message string
}

func (r *SignMessageRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.Message)
}

type FfiConverterTypeSignMessageRequest struct{}

var FfiConverterTypeSignMessageRequestINSTANCE = FfiConverterTypeSignMessageRequest{}

func (c FfiConverterTypeSignMessageRequest) Lift(rb RustBufferI) SignMessageRequest {
	return LiftFromRustBuffer[SignMessageRequest](c, rb)
}

func (c FfiConverterTypeSignMessageRequest) Read(reader io.Reader) SignMessageRequest {
	return SignMessageRequest{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSignMessageRequest) Lower(value SignMessageRequest) RustBuffer {
	return LowerIntoRustBuffer[SignMessageRequest](c, value)
}

func (c FfiConverterTypeSignMessageRequest) Write(writer io.Writer, value SignMessageRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.Message)
}

type FfiDestroyerTypeSignMessageRequest struct{}

func (_ FfiDestroyerTypeSignMessageRequest) Destroy(value SignMessageRequest) {
	value.Destroy()
}

type SignMessageResponse struct {
	Signature string
}

func (r *SignMessageResponse) Destroy() {
	FfiDestroyerString{}.Destroy(r.Signature)
}

type FfiConverterTypeSignMessageResponse struct{}

var FfiConverterTypeSignMessageResponseINSTANCE = FfiConverterTypeSignMessageResponse{}

func (c FfiConverterTypeSignMessageResponse) Lift(rb RustBufferI) SignMessageResponse {
	return LiftFromRustBuffer[SignMessageResponse](c, rb)
}

func (c FfiConverterTypeSignMessageResponse) Read(reader io.Reader) SignMessageResponse {
	return SignMessageResponse{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSignMessageResponse) Lower(value SignMessageResponse) RustBuffer {
	return LowerIntoRustBuffer[SignMessageResponse](c, value)
}

func (c FfiConverterTypeSignMessageResponse) Write(writer io.Writer, value SignMessageResponse) {
	FfiConverterStringINSTANCE.Write(writer, value.Signature)
}

type FfiDestroyerTypeSignMessageResponse struct{}

func (_ FfiDestroyerTypeSignMessageResponse) Destroy(value SignMessageResponse) {
	value.Destroy()
}

type StaticBackupRequest struct {
	WorkingDir string
}

func (r *StaticBackupRequest) Destroy() {
	FfiDestroyerString{}.Destroy(r.WorkingDir)
}

type FfiConverterTypeStaticBackupRequest struct{}

var FfiConverterTypeStaticBackupRequestINSTANCE = FfiConverterTypeStaticBackupRequest{}

func (c FfiConverterTypeStaticBackupRequest) Lift(rb RustBufferI) StaticBackupRequest {
	return LiftFromRustBuffer[StaticBackupRequest](c, rb)
}

func (c FfiConverterTypeStaticBackupRequest) Read(reader io.Reader) StaticBackupRequest {
	return StaticBackupRequest{
		FfiConverterStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeStaticBackupRequest) Lower(value StaticBackupRequest) RustBuffer {
	return LowerIntoRustBuffer[StaticBackupRequest](c, value)
}

func (c FfiConverterTypeStaticBackupRequest) Write(writer io.Writer, value StaticBackupRequest) {
	FfiConverterStringINSTANCE.Write(writer, value.WorkingDir)
}

type FfiDestroyerTypeStaticBackupRequest struct{}

func (_ FfiDestroyerTypeStaticBackupRequest) Destroy(value StaticBackupRequest) {
	value.Destroy()
}

type StaticBackupResponse struct {
	Backup *[]string
}

func (r *StaticBackupResponse) Destroy() {
	FfiDestroyerOptionalSequenceString{}.Destroy(r.Backup)
}

type FfiConverterTypeStaticBackupResponse struct{}

var FfiConverterTypeStaticBackupResponseINSTANCE = FfiConverterTypeStaticBackupResponse{}

func (c FfiConverterTypeStaticBackupResponse) Lift(rb RustBufferI) StaticBackupResponse {
	return LiftFromRustBuffer[StaticBackupResponse](c, rb)
}

func (c FfiConverterTypeStaticBackupResponse) Read(reader io.Reader) StaticBackupResponse {
	return StaticBackupResponse{
		FfiConverterOptionalSequenceStringINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeStaticBackupResponse) Lower(value StaticBackupResponse) RustBuffer {
	return LowerIntoRustBuffer[StaticBackupResponse](c, value)
}

func (c FfiConverterTypeStaticBackupResponse) Write(writer io.Writer, value StaticBackupResponse) {
	FfiConverterOptionalSequenceStringINSTANCE.Write(writer, value.Backup)
}

type FfiDestroyerTypeStaticBackupResponse struct{}

func (_ FfiDestroyerTypeStaticBackupResponse) Destroy(value StaticBackupResponse) {
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
	FfiDestroyerTypeSwapStatus{}.Destroy(r.Status)
	FfiDestroyerSequenceString{}.Destroy(r.RefundTxIds)
	FfiDestroyerSequenceString{}.Destroy(r.UnconfirmedTxIds)
	FfiDestroyerSequenceString{}.Destroy(r.ConfirmedTxIds)
	FfiDestroyerInt64{}.Destroy(r.MinAllowedDeposit)
	FfiDestroyerInt64{}.Destroy(r.MaxAllowedDeposit)
	FfiDestroyerInt64{}.Destroy(r.MaxSwapperPayable)
	FfiDestroyerOptionalString{}.Destroy(r.LastRedeemError)
	FfiDestroyerOptionalTypeOpeningFeeParams{}.Destroy(r.ChannelOpeningFees)
	FfiDestroyerOptionalUint32{}.Destroy(r.ConfirmedAt)
}

type FfiConverterTypeSwapInfo struct{}

var FfiConverterTypeSwapInfoINSTANCE = FfiConverterTypeSwapInfo{}

func (c FfiConverterTypeSwapInfo) Lift(rb RustBufferI) SwapInfo {
	return LiftFromRustBuffer[SwapInfo](c, rb)
}

func (c FfiConverterTypeSwapInfo) Read(reader io.Reader) SwapInfo {
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
		FfiConverterTypeSwapStatusINSTANCE.Read(reader),
		FfiConverterSequenceStringINSTANCE.Read(reader),
		FfiConverterSequenceStringINSTANCE.Read(reader),
		FfiConverterSequenceStringINSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterInt64INSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSwapInfo) Lower(value SwapInfo) RustBuffer {
	return LowerIntoRustBuffer[SwapInfo](c, value)
}

func (c FfiConverterTypeSwapInfo) Write(writer io.Writer, value SwapInfo) {
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
	FfiConverterTypeSwapStatusINSTANCE.Write(writer, value.Status)
	FfiConverterSequenceStringINSTANCE.Write(writer, value.RefundTxIds)
	FfiConverterSequenceStringINSTANCE.Write(writer, value.UnconfirmedTxIds)
	FfiConverterSequenceStringINSTANCE.Write(writer, value.ConfirmedTxIds)
	FfiConverterInt64INSTANCE.Write(writer, value.MinAllowedDeposit)
	FfiConverterInt64INSTANCE.Write(writer, value.MaxAllowedDeposit)
	FfiConverterInt64INSTANCE.Write(writer, value.MaxSwapperPayable)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.LastRedeemError)
	FfiConverterOptionalTypeOpeningFeeParamsINSTANCE.Write(writer, value.ChannelOpeningFees)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.ConfirmedAt)
}

type FfiDestroyerTypeSwapInfo struct{}

func (_ FfiDestroyerTypeSwapInfo) Destroy(value SwapInfo) {
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

type FfiConverterTypeSymbol struct{}

var FfiConverterTypeSymbolINSTANCE = FfiConverterTypeSymbol{}

func (c FfiConverterTypeSymbol) Lift(rb RustBufferI) Symbol {
	return LiftFromRustBuffer[Symbol](c, rb)
}

func (c FfiConverterTypeSymbol) Read(reader io.Reader) Symbol {
	return Symbol{
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalStringINSTANCE.Read(reader),
		FfiConverterOptionalBoolINSTANCE.Read(reader),
		FfiConverterOptionalUint32INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeSymbol) Lower(value Symbol) RustBuffer {
	return LowerIntoRustBuffer[Symbol](c, value)
}

func (c FfiConverterTypeSymbol) Write(writer io.Writer, value Symbol) {
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Grapheme)
	FfiConverterOptionalStringINSTANCE.Write(writer, value.Template)
	FfiConverterOptionalBoolINSTANCE.Write(writer, value.Rtl)
	FfiConverterOptionalUint32INSTANCE.Write(writer, value.Position)
}

type FfiDestroyerTypeSymbol struct{}

func (_ FfiDestroyerTypeSymbol) Destroy(value Symbol) {
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

type FfiConverterTypeTlvEntry struct{}

var FfiConverterTypeTlvEntryINSTANCE = FfiConverterTypeTlvEntry{}

func (c FfiConverterTypeTlvEntry) Lift(rb RustBufferI) TlvEntry {
	return LiftFromRustBuffer[TlvEntry](c, rb)
}

func (c FfiConverterTypeTlvEntry) Read(reader io.Reader) TlvEntry {
	return TlvEntry{
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterSequenceUint8INSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeTlvEntry) Lower(value TlvEntry) RustBuffer {
	return LowerIntoRustBuffer[TlvEntry](c, value)
}

func (c FfiConverterTypeTlvEntry) Write(writer io.Writer, value TlvEntry) {
	FfiConverterUint64INSTANCE.Write(writer, value.FieldNumber)
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Value)
}

type FfiDestroyerTypeTlvEntry struct{}

func (_ FfiDestroyerTypeTlvEntry) Destroy(value TlvEntry) {
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

type FfiConverterTypeUnspentTransactionOutput struct{}

var FfiConverterTypeUnspentTransactionOutputINSTANCE = FfiConverterTypeUnspentTransactionOutput{}

func (c FfiConverterTypeUnspentTransactionOutput) Lift(rb RustBufferI) UnspentTransactionOutput {
	return LiftFromRustBuffer[UnspentTransactionOutput](c, rb)
}

func (c FfiConverterTypeUnspentTransactionOutput) Read(reader io.Reader) UnspentTransactionOutput {
	return UnspentTransactionOutput{
		FfiConverterSequenceUint8INSTANCE.Read(reader),
		FfiConverterUint32INSTANCE.Read(reader),
		FfiConverterUint64INSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeUnspentTransactionOutput) Lower(value UnspentTransactionOutput) RustBuffer {
	return LowerIntoRustBuffer[UnspentTransactionOutput](c, value)
}

func (c FfiConverterTypeUnspentTransactionOutput) Write(writer io.Writer, value UnspentTransactionOutput) {
	FfiConverterSequenceUint8INSTANCE.Write(writer, value.Txid)
	FfiConverterUint32INSTANCE.Write(writer, value.Outnum)
	FfiConverterUint64INSTANCE.Write(writer, value.AmountMillisatoshi)
	FfiConverterStringINSTANCE.Write(writer, value.Address)
	FfiConverterBoolINSTANCE.Write(writer, value.Reserved)
}

type FfiDestroyerTypeUnspentTransactionOutput struct{}

func (_ FfiDestroyerTypeUnspentTransactionOutput) Destroy(value UnspentTransactionOutput) {
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

type FfiConverterTypeUrlSuccessActionData struct{}

var FfiConverterTypeUrlSuccessActionDataINSTANCE = FfiConverterTypeUrlSuccessActionData{}

func (c FfiConverterTypeUrlSuccessActionData) Lift(rb RustBufferI) UrlSuccessActionData {
	return LiftFromRustBuffer[UrlSuccessActionData](c, rb)
}

func (c FfiConverterTypeUrlSuccessActionData) Read(reader io.Reader) UrlSuccessActionData {
	return UrlSuccessActionData{
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterStringINSTANCE.Read(reader),
		FfiConverterBoolINSTANCE.Read(reader),
	}
}

func (c FfiConverterTypeUrlSuccessActionData) Lower(value UrlSuccessActionData) RustBuffer {
	return LowerIntoRustBuffer[UrlSuccessActionData](c, value)
}

func (c FfiConverterTypeUrlSuccessActionData) Write(writer io.Writer, value UrlSuccessActionData) {
	FfiConverterStringINSTANCE.Write(writer, value.Description)
	FfiConverterStringINSTANCE.Write(writer, value.Url)
	FfiConverterBoolINSTANCE.Write(writer, value.MatchesCallbackDomain)
}

type FfiDestroyerTypeUrlSuccessActionData struct{}

func (_ FfiDestroyerTypeUrlSuccessActionData) Destroy(value UrlSuccessActionData) {
	value.Destroy()
}

type AesSuccessActionDataResult interface {
	Destroy()
}
type AesSuccessActionDataResultDecrypted struct {
	Data AesSuccessActionDataDecrypted
}

func (e AesSuccessActionDataResultDecrypted) Destroy() {
	FfiDestroyerTypeAesSuccessActionDataDecrypted{}.Destroy(e.Data)
}

type AesSuccessActionDataResultErrorStatus struct {
	Reason string
}

func (e AesSuccessActionDataResultErrorStatus) Destroy() {
	FfiDestroyerString{}.Destroy(e.Reason)
}

type FfiConverterTypeAesSuccessActionDataResult struct{}

var FfiConverterTypeAesSuccessActionDataResultINSTANCE = FfiConverterTypeAesSuccessActionDataResult{}

func (c FfiConverterTypeAesSuccessActionDataResult) Lift(rb RustBufferI) AesSuccessActionDataResult {
	return LiftFromRustBuffer[AesSuccessActionDataResult](c, rb)
}

func (c FfiConverterTypeAesSuccessActionDataResult) Lower(value AesSuccessActionDataResult) RustBuffer {
	return LowerIntoRustBuffer[AesSuccessActionDataResult](c, value)
}
func (FfiConverterTypeAesSuccessActionDataResult) Read(reader io.Reader) AesSuccessActionDataResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return AesSuccessActionDataResultDecrypted{
			FfiConverterTypeAesSuccessActionDataDecryptedINSTANCE.Read(reader),
		}
	case 2:
		return AesSuccessActionDataResultErrorStatus{
			FfiConverterStringINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeAesSuccessActionDataResult.Read()", id))
	}
}

func (FfiConverterTypeAesSuccessActionDataResult) Write(writer io.Writer, value AesSuccessActionDataResult) {
	switch variant_value := value.(type) {
	case AesSuccessActionDataResultDecrypted:
		writeInt32(writer, 1)
		FfiConverterTypeAesSuccessActionDataDecryptedINSTANCE.Write(writer, variant_value.Data)
	case AesSuccessActionDataResultErrorStatus:
		writeInt32(writer, 2)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Reason)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeAesSuccessActionDataResult.Write", value))
	}
}

type FfiDestroyerTypeAesSuccessActionDataResult struct{}

func (_ FfiDestroyerTypeAesSuccessActionDataResult) Destroy(value AesSuccessActionDataResult) {
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
	FfiDestroyerTypeInvoicePaidDetails{}.Destroy(e.Details)
}

type BreezEventSynced struct {
}

func (e BreezEventSynced) Destroy() {
}

type BreezEventPaymentSucceed struct {
	Details Payment
}

func (e BreezEventPaymentSucceed) Destroy() {
	FfiDestroyerTypePayment{}.Destroy(e.Details)
}

type BreezEventPaymentFailed struct {
	Details PaymentFailedData
}

func (e BreezEventPaymentFailed) Destroy() {
	FfiDestroyerTypePaymentFailedData{}.Destroy(e.Details)
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
	FfiDestroyerTypeBackupFailedData{}.Destroy(e.Details)
}

type BreezEventReverseSwapUpdated struct {
	Details ReverseSwapInfo
}

func (e BreezEventReverseSwapUpdated) Destroy() {
	FfiDestroyerTypeReverseSwapInfo{}.Destroy(e.Details)
}

type BreezEventSwapUpdated struct {
	Details SwapInfo
}

func (e BreezEventSwapUpdated) Destroy() {
	FfiDestroyerTypeSwapInfo{}.Destroy(e.Details)
}

type FfiConverterTypeBreezEvent struct{}

var FfiConverterTypeBreezEventINSTANCE = FfiConverterTypeBreezEvent{}

func (c FfiConverterTypeBreezEvent) Lift(rb RustBufferI) BreezEvent {
	return LiftFromRustBuffer[BreezEvent](c, rb)
}

func (c FfiConverterTypeBreezEvent) Lower(value BreezEvent) RustBuffer {
	return LowerIntoRustBuffer[BreezEvent](c, value)
}
func (FfiConverterTypeBreezEvent) Read(reader io.Reader) BreezEvent {
	id := readInt32(reader)
	switch id {
	case 1:
		return BreezEventNewBlock{
			FfiConverterUint32INSTANCE.Read(reader),
		}
	case 2:
		return BreezEventInvoicePaid{
			FfiConverterTypeInvoicePaidDetailsINSTANCE.Read(reader),
		}
	case 3:
		return BreezEventSynced{}
	case 4:
		return BreezEventPaymentSucceed{
			FfiConverterTypePaymentINSTANCE.Read(reader),
		}
	case 5:
		return BreezEventPaymentFailed{
			FfiConverterTypePaymentFailedDataINSTANCE.Read(reader),
		}
	case 6:
		return BreezEventBackupStarted{}
	case 7:
		return BreezEventBackupSucceeded{}
	case 8:
		return BreezEventBackupFailed{
			FfiConverterTypeBackupFailedDataINSTANCE.Read(reader),
		}
	case 9:
		return BreezEventReverseSwapUpdated{
			FfiConverterTypeReverseSwapInfoINSTANCE.Read(reader),
		}
	case 10:
		return BreezEventSwapUpdated{
			FfiConverterTypeSwapInfoINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeBreezEvent.Read()", id))
	}
}

func (FfiConverterTypeBreezEvent) Write(writer io.Writer, value BreezEvent) {
	switch variant_value := value.(type) {
	case BreezEventNewBlock:
		writeInt32(writer, 1)
		FfiConverterUint32INSTANCE.Write(writer, variant_value.Block)
	case BreezEventInvoicePaid:
		writeInt32(writer, 2)
		FfiConverterTypeInvoicePaidDetailsINSTANCE.Write(writer, variant_value.Details)
	case BreezEventSynced:
		writeInt32(writer, 3)
	case BreezEventPaymentSucceed:
		writeInt32(writer, 4)
		FfiConverterTypePaymentINSTANCE.Write(writer, variant_value.Details)
	case BreezEventPaymentFailed:
		writeInt32(writer, 5)
		FfiConverterTypePaymentFailedDataINSTANCE.Write(writer, variant_value.Details)
	case BreezEventBackupStarted:
		writeInt32(writer, 6)
	case BreezEventBackupSucceeded:
		writeInt32(writer, 7)
	case BreezEventBackupFailed:
		writeInt32(writer, 8)
		FfiConverterTypeBackupFailedDataINSTANCE.Write(writer, variant_value.Details)
	case BreezEventReverseSwapUpdated:
		writeInt32(writer, 9)
		FfiConverterTypeReverseSwapInfoINSTANCE.Write(writer, variant_value.Details)
	case BreezEventSwapUpdated:
		writeInt32(writer, 10)
		FfiConverterTypeSwapInfoINSTANCE.Write(writer, variant_value.Details)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeBreezEvent.Write", value))
	}
}

type FfiDestroyerTypeBreezEvent struct{}

func (_ FfiDestroyerTypeBreezEvent) Destroy(value BreezEvent) {
	value.Destroy()
}

type BuyBitcoinProvider uint

const (
	BuyBitcoinProviderMoonpay BuyBitcoinProvider = 1
)

type FfiConverterTypeBuyBitcoinProvider struct{}

var FfiConverterTypeBuyBitcoinProviderINSTANCE = FfiConverterTypeBuyBitcoinProvider{}

func (c FfiConverterTypeBuyBitcoinProvider) Lift(rb RustBufferI) BuyBitcoinProvider {
	return LiftFromRustBuffer[BuyBitcoinProvider](c, rb)
}

func (c FfiConverterTypeBuyBitcoinProvider) Lower(value BuyBitcoinProvider) RustBuffer {
	return LowerIntoRustBuffer[BuyBitcoinProvider](c, value)
}
func (FfiConverterTypeBuyBitcoinProvider) Read(reader io.Reader) BuyBitcoinProvider {
	id := readInt32(reader)
	return BuyBitcoinProvider(id)
}

func (FfiConverterTypeBuyBitcoinProvider) Write(writer io.Writer, value BuyBitcoinProvider) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeBuyBitcoinProvider struct{}

func (_ FfiDestroyerTypeBuyBitcoinProvider) Destroy(value BuyBitcoinProvider) {
}

type ChannelState uint

const (
	ChannelStatePendingOpen  ChannelState = 1
	ChannelStateOpened       ChannelState = 2
	ChannelStatePendingClose ChannelState = 3
	ChannelStateClosed       ChannelState = 4
)

type FfiConverterTypeChannelState struct{}

var FfiConverterTypeChannelStateINSTANCE = FfiConverterTypeChannelState{}

func (c FfiConverterTypeChannelState) Lift(rb RustBufferI) ChannelState {
	return LiftFromRustBuffer[ChannelState](c, rb)
}

func (c FfiConverterTypeChannelState) Lower(value ChannelState) RustBuffer {
	return LowerIntoRustBuffer[ChannelState](c, value)
}
func (FfiConverterTypeChannelState) Read(reader io.Reader) ChannelState {
	id := readInt32(reader)
	return ChannelState(id)
}

func (FfiConverterTypeChannelState) Write(writer io.Writer, value ChannelState) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeChannelState struct{}

func (_ FfiDestroyerTypeChannelState) Destroy(value ChannelState) {
}

type ConnectError struct {
	err error
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
	return &ConnectError{
		err: &ConnectErrorGeneric{},
	}
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
	return &ConnectError{
		err: &ConnectErrorRestoreOnly{},
	}
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
	return &ConnectError{
		err: &ConnectErrorServiceConnectivity{},
	}
}

func (err ConnectErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self ConnectErrorServiceConnectivity) Is(target error) bool {
	return target == ErrConnectErrorServiceConnectivity
}

type FfiConverterTypeConnectError struct{}

var FfiConverterTypeConnectErrorINSTANCE = FfiConverterTypeConnectError{}

func (c FfiConverterTypeConnectError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeConnectError) Lower(value *ConnectError) RustBuffer {
	return LowerIntoRustBuffer[*ConnectError](c, value)
}

func (c FfiConverterTypeConnectError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeConnectError.Read()", errorID))
	}

}

func (c FfiConverterTypeConnectError) Write(writer io.Writer, value *ConnectError) {
	switch variantValue := value.err.(type) {
	case *ConnectErrorGeneric:
		writeInt32(writer, 1)
	case *ConnectErrorRestoreOnly:
		writeInt32(writer, 2)
	case *ConnectErrorServiceConnectivity:
		writeInt32(writer, 3)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeConnectError.Write", value))
	}
}

type EnvironmentType uint

const (
	EnvironmentTypeProduction EnvironmentType = 1
	EnvironmentTypeStaging    EnvironmentType = 2
)

type FfiConverterTypeEnvironmentType struct{}

var FfiConverterTypeEnvironmentTypeINSTANCE = FfiConverterTypeEnvironmentType{}

func (c FfiConverterTypeEnvironmentType) Lift(rb RustBufferI) EnvironmentType {
	return LiftFromRustBuffer[EnvironmentType](c, rb)
}

func (c FfiConverterTypeEnvironmentType) Lower(value EnvironmentType) RustBuffer {
	return LowerIntoRustBuffer[EnvironmentType](c, value)
}
func (FfiConverterTypeEnvironmentType) Read(reader io.Reader) EnvironmentType {
	id := readInt32(reader)
	return EnvironmentType(id)
}

func (FfiConverterTypeEnvironmentType) Write(writer io.Writer, value EnvironmentType) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeEnvironmentType struct{}

func (_ FfiDestroyerTypeEnvironmentType) Destroy(value EnvironmentType) {
}

type FeeratePreset uint

const (
	FeeratePresetRegular  FeeratePreset = 1
	FeeratePresetEconomy  FeeratePreset = 2
	FeeratePresetPriority FeeratePreset = 3
)

type FfiConverterTypeFeeratePreset struct{}

var FfiConverterTypeFeeratePresetINSTANCE = FfiConverterTypeFeeratePreset{}

func (c FfiConverterTypeFeeratePreset) Lift(rb RustBufferI) FeeratePreset {
	return LiftFromRustBuffer[FeeratePreset](c, rb)
}

func (c FfiConverterTypeFeeratePreset) Lower(value FeeratePreset) RustBuffer {
	return LowerIntoRustBuffer[FeeratePreset](c, value)
}
func (FfiConverterTypeFeeratePreset) Read(reader io.Reader) FeeratePreset {
	id := readInt32(reader)
	return FeeratePreset(id)
}

func (FfiConverterTypeFeeratePreset) Write(writer io.Writer, value FeeratePreset) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeFeeratePreset struct{}

func (_ FfiDestroyerTypeFeeratePreset) Destroy(value FeeratePreset) {
}

type HealthCheckStatus uint

const (
	HealthCheckStatusOperational       HealthCheckStatus = 1
	HealthCheckStatusMaintenance       HealthCheckStatus = 2
	HealthCheckStatusServiceDisruption HealthCheckStatus = 3
)

type FfiConverterTypeHealthCheckStatus struct{}

var FfiConverterTypeHealthCheckStatusINSTANCE = FfiConverterTypeHealthCheckStatus{}

func (c FfiConverterTypeHealthCheckStatus) Lift(rb RustBufferI) HealthCheckStatus {
	return LiftFromRustBuffer[HealthCheckStatus](c, rb)
}

func (c FfiConverterTypeHealthCheckStatus) Lower(value HealthCheckStatus) RustBuffer {
	return LowerIntoRustBuffer[HealthCheckStatus](c, value)
}
func (FfiConverterTypeHealthCheckStatus) Read(reader io.Reader) HealthCheckStatus {
	id := readInt32(reader)
	return HealthCheckStatus(id)
}

func (FfiConverterTypeHealthCheckStatus) Write(writer io.Writer, value HealthCheckStatus) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeHealthCheckStatus struct{}

func (_ FfiDestroyerTypeHealthCheckStatus) Destroy(value HealthCheckStatus) {
}

type InputType interface {
	Destroy()
}
type InputTypeBitcoinAddress struct {
	Address BitcoinAddressData
}

func (e InputTypeBitcoinAddress) Destroy() {
	FfiDestroyerTypeBitcoinAddressData{}.Destroy(e.Address)
}

type InputTypeBolt11 struct {
	Invoice LnInvoice
}

func (e InputTypeBolt11) Destroy() {
	FfiDestroyerTypeLnInvoice{}.Destroy(e.Invoice)
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
	FfiDestroyerTypeLnUrlPayRequestData{}.Destroy(e.Data)
	FfiDestroyerOptionalString{}.Destroy(e.Bip353Address)
}

type InputTypeLnUrlWithdraw struct {
	Data LnUrlWithdrawRequestData
}

func (e InputTypeLnUrlWithdraw) Destroy() {
	FfiDestroyerTypeLnUrlWithdrawRequestData{}.Destroy(e.Data)
}

type InputTypeLnUrlAuth struct {
	Data LnUrlAuthRequestData
}

func (e InputTypeLnUrlAuth) Destroy() {
	FfiDestroyerTypeLnUrlAuthRequestData{}.Destroy(e.Data)
}

type InputTypeLnUrlError struct {
	Data LnUrlErrorData
}

func (e InputTypeLnUrlError) Destroy() {
	FfiDestroyerTypeLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterTypeInputType struct{}

var FfiConverterTypeInputTypeINSTANCE = FfiConverterTypeInputType{}

func (c FfiConverterTypeInputType) Lift(rb RustBufferI) InputType {
	return LiftFromRustBuffer[InputType](c, rb)
}

func (c FfiConverterTypeInputType) Lower(value InputType) RustBuffer {
	return LowerIntoRustBuffer[InputType](c, value)
}
func (FfiConverterTypeInputType) Read(reader io.Reader) InputType {
	id := readInt32(reader)
	switch id {
	case 1:
		return InputTypeBitcoinAddress{
			FfiConverterTypeBitcoinAddressDataINSTANCE.Read(reader),
		}
	case 2:
		return InputTypeBolt11{
			FfiConverterTypeLNInvoiceINSTANCE.Read(reader),
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
			FfiConverterTypeLnUrlPayRequestDataINSTANCE.Read(reader),
			FfiConverterOptionalStringINSTANCE.Read(reader),
		}
	case 6:
		return InputTypeLnUrlWithdraw{
			FfiConverterTypeLnUrlWithdrawRequestDataINSTANCE.Read(reader),
		}
	case 7:
		return InputTypeLnUrlAuth{
			FfiConverterTypeLnUrlAuthRequestDataINSTANCE.Read(reader),
		}
	case 8:
		return InputTypeLnUrlError{
			FfiConverterTypeLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeInputType.Read()", id))
	}
}

func (FfiConverterTypeInputType) Write(writer io.Writer, value InputType) {
	switch variant_value := value.(type) {
	case InputTypeBitcoinAddress:
		writeInt32(writer, 1)
		FfiConverterTypeBitcoinAddressDataINSTANCE.Write(writer, variant_value.Address)
	case InputTypeBolt11:
		writeInt32(writer, 2)
		FfiConverterTypeLNInvoiceINSTANCE.Write(writer, variant_value.Invoice)
	case InputTypeNodeId:
		writeInt32(writer, 3)
		FfiConverterStringINSTANCE.Write(writer, variant_value.NodeId)
	case InputTypeUrl:
		writeInt32(writer, 4)
		FfiConverterStringINSTANCE.Write(writer, variant_value.Url)
	case InputTypeLnUrlPay:
		writeInt32(writer, 5)
		FfiConverterTypeLnUrlPayRequestDataINSTANCE.Write(writer, variant_value.Data)
		FfiConverterOptionalStringINSTANCE.Write(writer, variant_value.Bip353Address)
	case InputTypeLnUrlWithdraw:
		writeInt32(writer, 6)
		FfiConverterTypeLnUrlWithdrawRequestDataINSTANCE.Write(writer, variant_value.Data)
	case InputTypeLnUrlAuth:
		writeInt32(writer, 7)
		FfiConverterTypeLnUrlAuthRequestDataINSTANCE.Write(writer, variant_value.Data)
	case InputTypeLnUrlError:
		writeInt32(writer, 8)
		FfiConverterTypeLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeInputType.Write", value))
	}
}

type FfiDestroyerTypeInputType struct{}

func (_ FfiDestroyerTypeInputType) Destroy(value InputType) {
	value.Destroy()
}

type LnUrlAuthError struct {
	err error
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
	return &LnUrlAuthError{
		err: &LnUrlAuthErrorGeneric{},
	}
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
	return &LnUrlAuthError{
		err: &LnUrlAuthErrorInvalidUri{},
	}
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
	return &LnUrlAuthError{
		err: &LnUrlAuthErrorServiceConnectivity{},
	}
}

func (err LnUrlAuthErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self LnUrlAuthErrorServiceConnectivity) Is(target error) bool {
	return target == ErrLnUrlAuthErrorServiceConnectivity
}

type FfiConverterTypeLnUrlAuthError struct{}

var FfiConverterTypeLnUrlAuthErrorINSTANCE = FfiConverterTypeLnUrlAuthError{}

func (c FfiConverterTypeLnUrlAuthError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeLnUrlAuthError) Lower(value *LnUrlAuthError) RustBuffer {
	return LowerIntoRustBuffer[*LnUrlAuthError](c, value)
}

func (c FfiConverterTypeLnUrlAuthError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeLnUrlAuthError.Read()", errorID))
	}

}

func (c FfiConverterTypeLnUrlAuthError) Write(writer io.Writer, value *LnUrlAuthError) {
	switch variantValue := value.err.(type) {
	case *LnUrlAuthErrorGeneric:
		writeInt32(writer, 1)
	case *LnUrlAuthErrorInvalidUri:
		writeInt32(writer, 2)
	case *LnUrlAuthErrorServiceConnectivity:
		writeInt32(writer, 3)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeLnUrlAuthError.Write", value))
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
	FfiDestroyerTypeLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterTypeLnUrlCallbackStatus struct{}

var FfiConverterTypeLnUrlCallbackStatusINSTANCE = FfiConverterTypeLnUrlCallbackStatus{}

func (c FfiConverterTypeLnUrlCallbackStatus) Lift(rb RustBufferI) LnUrlCallbackStatus {
	return LiftFromRustBuffer[LnUrlCallbackStatus](c, rb)
}

func (c FfiConverterTypeLnUrlCallbackStatus) Lower(value LnUrlCallbackStatus) RustBuffer {
	return LowerIntoRustBuffer[LnUrlCallbackStatus](c, value)
}
func (FfiConverterTypeLnUrlCallbackStatus) Read(reader io.Reader) LnUrlCallbackStatus {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlCallbackStatusOk{}
	case 2:
		return LnUrlCallbackStatusErrorStatus{
			FfiConverterTypeLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeLnUrlCallbackStatus.Read()", id))
	}
}

func (FfiConverterTypeLnUrlCallbackStatus) Write(writer io.Writer, value LnUrlCallbackStatus) {
	switch variant_value := value.(type) {
	case LnUrlCallbackStatusOk:
		writeInt32(writer, 1)
	case LnUrlCallbackStatusErrorStatus:
		writeInt32(writer, 2)
		FfiConverterTypeLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeLnUrlCallbackStatus.Write", value))
	}
}

type FfiDestroyerTypeLnUrlCallbackStatus struct{}

func (_ FfiDestroyerTypeLnUrlCallbackStatus) Destroy(value LnUrlCallbackStatus) {
	value.Destroy()
}

type LnUrlPayError struct {
	err error
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorAlreadyPaid{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorGeneric{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorInvalidAmount{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorInvalidInvoice{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorInvalidNetwork{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorInvalidUri{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorInvoiceExpired{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorPaymentFailed{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorPaymentTimeout{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorRouteNotFound{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorRouteTooExpensive{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorServiceConnectivity{},
	}
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
	return &LnUrlPayError{
		err: &LnUrlPayErrorInsufficientBalance{},
	}
}

func (err LnUrlPayErrorInsufficientBalance) Error() string {
	return fmt.Sprintf("InsufficientBalance: %s", err.message)
}

func (self LnUrlPayErrorInsufficientBalance) Is(target error) bool {
	return target == ErrLnUrlPayErrorInsufficientBalance
}

type FfiConverterTypeLnUrlPayError struct{}

var FfiConverterTypeLnUrlPayErrorINSTANCE = FfiConverterTypeLnUrlPayError{}

func (c FfiConverterTypeLnUrlPayError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeLnUrlPayError) Lower(value *LnUrlPayError) RustBuffer {
	return LowerIntoRustBuffer[*LnUrlPayError](c, value)
}

func (c FfiConverterTypeLnUrlPayError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeLnUrlPayError.Read()", errorID))
	}

}

func (c FfiConverterTypeLnUrlPayError) Write(writer io.Writer, value *LnUrlPayError) {
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
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeLnUrlPayError.Write", value))
	}
}

type LnUrlPayResult interface {
	Destroy()
}
type LnUrlPayResultEndpointSuccess struct {
	Data LnUrlPaySuccessData
}

func (e LnUrlPayResultEndpointSuccess) Destroy() {
	FfiDestroyerTypeLnUrlPaySuccessData{}.Destroy(e.Data)
}

type LnUrlPayResultEndpointError struct {
	Data LnUrlErrorData
}

func (e LnUrlPayResultEndpointError) Destroy() {
	FfiDestroyerTypeLnUrlErrorData{}.Destroy(e.Data)
}

type LnUrlPayResultPayError struct {
	Data LnUrlPayErrorData
}

func (e LnUrlPayResultPayError) Destroy() {
	FfiDestroyerTypeLnUrlPayErrorData{}.Destroy(e.Data)
}

type FfiConverterTypeLnUrlPayResult struct{}

var FfiConverterTypeLnUrlPayResultINSTANCE = FfiConverterTypeLnUrlPayResult{}

func (c FfiConverterTypeLnUrlPayResult) Lift(rb RustBufferI) LnUrlPayResult {
	return LiftFromRustBuffer[LnUrlPayResult](c, rb)
}

func (c FfiConverterTypeLnUrlPayResult) Lower(value LnUrlPayResult) RustBuffer {
	return LowerIntoRustBuffer[LnUrlPayResult](c, value)
}
func (FfiConverterTypeLnUrlPayResult) Read(reader io.Reader) LnUrlPayResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlPayResultEndpointSuccess{
			FfiConverterTypeLnUrlPaySuccessDataINSTANCE.Read(reader),
		}
	case 2:
		return LnUrlPayResultEndpointError{
			FfiConverterTypeLnUrlErrorDataINSTANCE.Read(reader),
		}
	case 3:
		return LnUrlPayResultPayError{
			FfiConverterTypeLnUrlPayErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeLnUrlPayResult.Read()", id))
	}
}

func (FfiConverterTypeLnUrlPayResult) Write(writer io.Writer, value LnUrlPayResult) {
	switch variant_value := value.(type) {
	case LnUrlPayResultEndpointSuccess:
		writeInt32(writer, 1)
		FfiConverterTypeLnUrlPaySuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlPayResultEndpointError:
		writeInt32(writer, 2)
		FfiConverterTypeLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlPayResultPayError:
		writeInt32(writer, 3)
		FfiConverterTypeLnUrlPayErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeLnUrlPayResult.Write", value))
	}
}

type FfiDestroyerTypeLnUrlPayResult struct{}

func (_ FfiDestroyerTypeLnUrlPayResult) Destroy(value LnUrlPayResult) {
	value.Destroy()
}

type LnUrlWithdrawError struct {
	err error
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorGeneric{},
	}
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorInvalidAmount{},
	}
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorInvalidInvoice{},
	}
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorInvalidUri{},
	}
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorServiceConnectivity{},
	}
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
	return &LnUrlWithdrawError{
		err: &LnUrlWithdrawErrorInvoiceNoRoutingHints{},
	}
}

func (err LnUrlWithdrawErrorInvoiceNoRoutingHints) Error() string {
	return fmt.Sprintf("InvoiceNoRoutingHints: %s", err.message)
}

func (self LnUrlWithdrawErrorInvoiceNoRoutingHints) Is(target error) bool {
	return target == ErrLnUrlWithdrawErrorInvoiceNoRoutingHints
}

type FfiConverterTypeLnUrlWithdrawError struct{}

var FfiConverterTypeLnUrlWithdrawErrorINSTANCE = FfiConverterTypeLnUrlWithdrawError{}

func (c FfiConverterTypeLnUrlWithdrawError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeLnUrlWithdrawError) Lower(value *LnUrlWithdrawError) RustBuffer {
	return LowerIntoRustBuffer[*LnUrlWithdrawError](c, value)
}

func (c FfiConverterTypeLnUrlWithdrawError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeLnUrlWithdrawError.Read()", errorID))
	}

}

func (c FfiConverterTypeLnUrlWithdrawError) Write(writer io.Writer, value *LnUrlWithdrawError) {
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
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeLnUrlWithdrawError.Write", value))
	}
}

type LnUrlWithdrawResult interface {
	Destroy()
}
type LnUrlWithdrawResultOk struct {
	Data LnUrlWithdrawSuccessData
}

func (e LnUrlWithdrawResultOk) Destroy() {
	FfiDestroyerTypeLnUrlWithdrawSuccessData{}.Destroy(e.Data)
}

type LnUrlWithdrawResultTimeout struct {
	Data LnUrlWithdrawSuccessData
}

func (e LnUrlWithdrawResultTimeout) Destroy() {
	FfiDestroyerTypeLnUrlWithdrawSuccessData{}.Destroy(e.Data)
}

type LnUrlWithdrawResultErrorStatus struct {
	Data LnUrlErrorData
}

func (e LnUrlWithdrawResultErrorStatus) Destroy() {
	FfiDestroyerTypeLnUrlErrorData{}.Destroy(e.Data)
}

type FfiConverterTypeLnUrlWithdrawResult struct{}

var FfiConverterTypeLnUrlWithdrawResultINSTANCE = FfiConverterTypeLnUrlWithdrawResult{}

func (c FfiConverterTypeLnUrlWithdrawResult) Lift(rb RustBufferI) LnUrlWithdrawResult {
	return LiftFromRustBuffer[LnUrlWithdrawResult](c, rb)
}

func (c FfiConverterTypeLnUrlWithdrawResult) Lower(value LnUrlWithdrawResult) RustBuffer {
	return LowerIntoRustBuffer[LnUrlWithdrawResult](c, value)
}
func (FfiConverterTypeLnUrlWithdrawResult) Read(reader io.Reader) LnUrlWithdrawResult {
	id := readInt32(reader)
	switch id {
	case 1:
		return LnUrlWithdrawResultOk{
			FfiConverterTypeLnUrlWithdrawSuccessDataINSTANCE.Read(reader),
		}
	case 2:
		return LnUrlWithdrawResultTimeout{
			FfiConverterTypeLnUrlWithdrawSuccessDataINSTANCE.Read(reader),
		}
	case 3:
		return LnUrlWithdrawResultErrorStatus{
			FfiConverterTypeLnUrlErrorDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeLnUrlWithdrawResult.Read()", id))
	}
}

func (FfiConverterTypeLnUrlWithdrawResult) Write(writer io.Writer, value LnUrlWithdrawResult) {
	switch variant_value := value.(type) {
	case LnUrlWithdrawResultOk:
		writeInt32(writer, 1)
		FfiConverterTypeLnUrlWithdrawSuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlWithdrawResultTimeout:
		writeInt32(writer, 2)
		FfiConverterTypeLnUrlWithdrawSuccessDataINSTANCE.Write(writer, variant_value.Data)
	case LnUrlWithdrawResultErrorStatus:
		writeInt32(writer, 3)
		FfiConverterTypeLnUrlErrorDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeLnUrlWithdrawResult.Write", value))
	}
}

type FfiDestroyerTypeLnUrlWithdrawResult struct{}

func (_ FfiDestroyerTypeLnUrlWithdrawResult) Destroy(value LnUrlWithdrawResult) {
	value.Destroy()
}

type Network uint

const (
	NetworkBitcoin Network = 1
	NetworkTestnet Network = 2
	NetworkSignet  Network = 3
	NetworkRegtest Network = 4
)

type FfiConverterTypeNetwork struct{}

var FfiConverterTypeNetworkINSTANCE = FfiConverterTypeNetwork{}

func (c FfiConverterTypeNetwork) Lift(rb RustBufferI) Network {
	return LiftFromRustBuffer[Network](c, rb)
}

func (c FfiConverterTypeNetwork) Lower(value Network) RustBuffer {
	return LowerIntoRustBuffer[Network](c, value)
}
func (FfiConverterTypeNetwork) Read(reader io.Reader) Network {
	id := readInt32(reader)
	return Network(id)
}

func (FfiConverterTypeNetwork) Write(writer io.Writer, value Network) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeNetwork struct{}

func (_ FfiDestroyerTypeNetwork) Destroy(value Network) {
}

type NodeConfig interface {
	Destroy()
}
type NodeConfigGreenlight struct {
	Config GreenlightNodeConfig
}

func (e NodeConfigGreenlight) Destroy() {
	FfiDestroyerTypeGreenlightNodeConfig{}.Destroy(e.Config)
}

type FfiConverterTypeNodeConfig struct{}

var FfiConverterTypeNodeConfigINSTANCE = FfiConverterTypeNodeConfig{}

func (c FfiConverterTypeNodeConfig) Lift(rb RustBufferI) NodeConfig {
	return LiftFromRustBuffer[NodeConfig](c, rb)
}

func (c FfiConverterTypeNodeConfig) Lower(value NodeConfig) RustBuffer {
	return LowerIntoRustBuffer[NodeConfig](c, value)
}
func (FfiConverterTypeNodeConfig) Read(reader io.Reader) NodeConfig {
	id := readInt32(reader)
	switch id {
	case 1:
		return NodeConfigGreenlight{
			FfiConverterTypeGreenlightNodeConfigINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeNodeConfig.Read()", id))
	}
}

func (FfiConverterTypeNodeConfig) Write(writer io.Writer, value NodeConfig) {
	switch variant_value := value.(type) {
	case NodeConfigGreenlight:
		writeInt32(writer, 1)
		FfiConverterTypeGreenlightNodeConfigINSTANCE.Write(writer, variant_value.Config)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeNodeConfig.Write", value))
	}
}

type FfiDestroyerTypeNodeConfig struct{}

func (_ FfiDestroyerTypeNodeConfig) Destroy(value NodeConfig) {
	value.Destroy()
}

type NodeCredentials interface {
	Destroy()
}
type NodeCredentialsGreenlight struct {
	Credentials GreenlightDeviceCredentials
}

func (e NodeCredentialsGreenlight) Destroy() {
	FfiDestroyerTypeGreenlightDeviceCredentials{}.Destroy(e.Credentials)
}

type FfiConverterTypeNodeCredentials struct{}

var FfiConverterTypeNodeCredentialsINSTANCE = FfiConverterTypeNodeCredentials{}

func (c FfiConverterTypeNodeCredentials) Lift(rb RustBufferI) NodeCredentials {
	return LiftFromRustBuffer[NodeCredentials](c, rb)
}

func (c FfiConverterTypeNodeCredentials) Lower(value NodeCredentials) RustBuffer {
	return LowerIntoRustBuffer[NodeCredentials](c, value)
}
func (FfiConverterTypeNodeCredentials) Read(reader io.Reader) NodeCredentials {
	id := readInt32(reader)
	switch id {
	case 1:
		return NodeCredentialsGreenlight{
			FfiConverterTypeGreenlightDeviceCredentialsINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeNodeCredentials.Read()", id))
	}
}

func (FfiConverterTypeNodeCredentials) Write(writer io.Writer, value NodeCredentials) {
	switch variant_value := value.(type) {
	case NodeCredentialsGreenlight:
		writeInt32(writer, 1)
		FfiConverterTypeGreenlightDeviceCredentialsINSTANCE.Write(writer, variant_value.Credentials)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeNodeCredentials.Write", value))
	}
}

type FfiDestroyerTypeNodeCredentials struct{}

func (_ FfiDestroyerTypeNodeCredentials) Destroy(value NodeCredentials) {
	value.Destroy()
}

type PaymentDetails interface {
	Destroy()
}
type PaymentDetailsLn struct {
	Data LnPaymentDetails
}

func (e PaymentDetailsLn) Destroy() {
	FfiDestroyerTypeLnPaymentDetails{}.Destroy(e.Data)
}

type PaymentDetailsClosedChannel struct {
	Data ClosedChannelPaymentDetails
}

func (e PaymentDetailsClosedChannel) Destroy() {
	FfiDestroyerTypeClosedChannelPaymentDetails{}.Destroy(e.Data)
}

type FfiConverterTypePaymentDetails struct{}

var FfiConverterTypePaymentDetailsINSTANCE = FfiConverterTypePaymentDetails{}

func (c FfiConverterTypePaymentDetails) Lift(rb RustBufferI) PaymentDetails {
	return LiftFromRustBuffer[PaymentDetails](c, rb)
}

func (c FfiConverterTypePaymentDetails) Lower(value PaymentDetails) RustBuffer {
	return LowerIntoRustBuffer[PaymentDetails](c, value)
}
func (FfiConverterTypePaymentDetails) Read(reader io.Reader) PaymentDetails {
	id := readInt32(reader)
	switch id {
	case 1:
		return PaymentDetailsLn{
			FfiConverterTypeLnPaymentDetailsINSTANCE.Read(reader),
		}
	case 2:
		return PaymentDetailsClosedChannel{
			FfiConverterTypeClosedChannelPaymentDetailsINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypePaymentDetails.Read()", id))
	}
}

func (FfiConverterTypePaymentDetails) Write(writer io.Writer, value PaymentDetails) {
	switch variant_value := value.(type) {
	case PaymentDetailsLn:
		writeInt32(writer, 1)
		FfiConverterTypeLnPaymentDetailsINSTANCE.Write(writer, variant_value.Data)
	case PaymentDetailsClosedChannel:
		writeInt32(writer, 2)
		FfiConverterTypeClosedChannelPaymentDetailsINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypePaymentDetails.Write", value))
	}
}

type FfiDestroyerTypePaymentDetails struct{}

func (_ FfiDestroyerTypePaymentDetails) Destroy(value PaymentDetails) {
	value.Destroy()
}

type PaymentStatus uint

const (
	PaymentStatusPending  PaymentStatus = 1
	PaymentStatusComplete PaymentStatus = 2
	PaymentStatusFailed   PaymentStatus = 3
)

type FfiConverterTypePaymentStatus struct{}

var FfiConverterTypePaymentStatusINSTANCE = FfiConverterTypePaymentStatus{}

func (c FfiConverterTypePaymentStatus) Lift(rb RustBufferI) PaymentStatus {
	return LiftFromRustBuffer[PaymentStatus](c, rb)
}

func (c FfiConverterTypePaymentStatus) Lower(value PaymentStatus) RustBuffer {
	return LowerIntoRustBuffer[PaymentStatus](c, value)
}
func (FfiConverterTypePaymentStatus) Read(reader io.Reader) PaymentStatus {
	id := readInt32(reader)
	return PaymentStatus(id)
}

func (FfiConverterTypePaymentStatus) Write(writer io.Writer, value PaymentStatus) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypePaymentStatus struct{}

func (_ FfiDestroyerTypePaymentStatus) Destroy(value PaymentStatus) {
}

type PaymentType uint

const (
	PaymentTypeSent          PaymentType = 1
	PaymentTypeReceived      PaymentType = 2
	PaymentTypeClosedChannel PaymentType = 3
)

type FfiConverterTypePaymentType struct{}

var FfiConverterTypePaymentTypeINSTANCE = FfiConverterTypePaymentType{}

func (c FfiConverterTypePaymentType) Lift(rb RustBufferI) PaymentType {
	return LiftFromRustBuffer[PaymentType](c, rb)
}

func (c FfiConverterTypePaymentType) Lower(value PaymentType) RustBuffer {
	return LowerIntoRustBuffer[PaymentType](c, value)
}
func (FfiConverterTypePaymentType) Read(reader io.Reader) PaymentType {
	id := readInt32(reader)
	return PaymentType(id)
}

func (FfiConverterTypePaymentType) Write(writer io.Writer, value PaymentType) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypePaymentType struct{}

func (_ FfiDestroyerTypePaymentType) Destroy(value PaymentType) {
}

type PaymentTypeFilter uint

const (
	PaymentTypeFilterSent          PaymentTypeFilter = 1
	PaymentTypeFilterReceived      PaymentTypeFilter = 2
	PaymentTypeFilterClosedChannel PaymentTypeFilter = 3
)

type FfiConverterTypePaymentTypeFilter struct{}

var FfiConverterTypePaymentTypeFilterINSTANCE = FfiConverterTypePaymentTypeFilter{}

func (c FfiConverterTypePaymentTypeFilter) Lift(rb RustBufferI) PaymentTypeFilter {
	return LiftFromRustBuffer[PaymentTypeFilter](c, rb)
}

func (c FfiConverterTypePaymentTypeFilter) Lower(value PaymentTypeFilter) RustBuffer {
	return LowerIntoRustBuffer[PaymentTypeFilter](c, value)
}
func (FfiConverterTypePaymentTypeFilter) Read(reader io.Reader) PaymentTypeFilter {
	id := readInt32(reader)
	return PaymentTypeFilter(id)
}

func (FfiConverterTypePaymentTypeFilter) Write(writer io.Writer, value PaymentTypeFilter) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypePaymentTypeFilter struct{}

func (_ FfiDestroyerTypePaymentTypeFilter) Destroy(value PaymentTypeFilter) {
}

type ReceiveOnchainError struct {
	err error
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
	return &ReceiveOnchainError{
		err: &ReceiveOnchainErrorGeneric{},
	}
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
	return &ReceiveOnchainError{
		err: &ReceiveOnchainErrorServiceConnectivity{},
	}
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
	return &ReceiveOnchainError{
		err: &ReceiveOnchainErrorSwapInProgress{},
	}
}

func (err ReceiveOnchainErrorSwapInProgress) Error() string {
	return fmt.Sprintf("SwapInProgress: %s", err.message)
}

func (self ReceiveOnchainErrorSwapInProgress) Is(target error) bool {
	return target == ErrReceiveOnchainErrorSwapInProgress
}

type FfiConverterTypeReceiveOnchainError struct{}

var FfiConverterTypeReceiveOnchainErrorINSTANCE = FfiConverterTypeReceiveOnchainError{}

func (c FfiConverterTypeReceiveOnchainError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeReceiveOnchainError) Lower(value *ReceiveOnchainError) RustBuffer {
	return LowerIntoRustBuffer[*ReceiveOnchainError](c, value)
}

func (c FfiConverterTypeReceiveOnchainError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeReceiveOnchainError.Read()", errorID))
	}

}

func (c FfiConverterTypeReceiveOnchainError) Write(writer io.Writer, value *ReceiveOnchainError) {
	switch variantValue := value.err.(type) {
	case *ReceiveOnchainErrorGeneric:
		writeInt32(writer, 1)
	case *ReceiveOnchainErrorServiceConnectivity:
		writeInt32(writer, 2)
	case *ReceiveOnchainErrorSwapInProgress:
		writeInt32(writer, 3)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeReceiveOnchainError.Write", value))
	}
}

type ReceivePaymentError struct {
	err error
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
	return &ReceivePaymentError{
		err: &ReceivePaymentErrorGeneric{},
	}
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
	return &ReceivePaymentError{
		err: &ReceivePaymentErrorInvalidAmount{},
	}
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
	return &ReceivePaymentError{
		err: &ReceivePaymentErrorInvalidInvoice{},
	}
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
	return &ReceivePaymentError{
		err: &ReceivePaymentErrorInvoiceExpired{},
	}
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
	return &ReceivePaymentError{
		err: &ReceivePaymentErrorInvoiceNoDescription{},
	}
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
	return &ReceivePaymentError{
		err: &ReceivePaymentErrorInvoicePreimageAlreadyExists{},
	}
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
	return &ReceivePaymentError{
		err: &ReceivePaymentErrorServiceConnectivity{},
	}
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
	return &ReceivePaymentError{
		err: &ReceivePaymentErrorInvoiceNoRoutingHints{},
	}
}

func (err ReceivePaymentErrorInvoiceNoRoutingHints) Error() string {
	return fmt.Sprintf("InvoiceNoRoutingHints: %s", err.message)
}

func (self ReceivePaymentErrorInvoiceNoRoutingHints) Is(target error) bool {
	return target == ErrReceivePaymentErrorInvoiceNoRoutingHints
}

type FfiConverterTypeReceivePaymentError struct{}

var FfiConverterTypeReceivePaymentErrorINSTANCE = FfiConverterTypeReceivePaymentError{}

func (c FfiConverterTypeReceivePaymentError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeReceivePaymentError) Lower(value *ReceivePaymentError) RustBuffer {
	return LowerIntoRustBuffer[*ReceivePaymentError](c, value)
}

func (c FfiConverterTypeReceivePaymentError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeReceivePaymentError.Read()", errorID))
	}

}

func (c FfiConverterTypeReceivePaymentError) Write(writer io.Writer, value *ReceivePaymentError) {
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
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeReceivePaymentError.Write", value))
	}
}

type RedeemOnchainError struct {
	err error
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
	return &RedeemOnchainError{
		err: &RedeemOnchainErrorGeneric{},
	}
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
	return &RedeemOnchainError{
		err: &RedeemOnchainErrorServiceConnectivity{},
	}
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
	return &RedeemOnchainError{
		err: &RedeemOnchainErrorInsufficientFunds{},
	}
}

func (err RedeemOnchainErrorInsufficientFunds) Error() string {
	return fmt.Sprintf("InsufficientFunds: %s", err.message)
}

func (self RedeemOnchainErrorInsufficientFunds) Is(target error) bool {
	return target == ErrRedeemOnchainErrorInsufficientFunds
}

type FfiConverterTypeRedeemOnchainError struct{}

var FfiConverterTypeRedeemOnchainErrorINSTANCE = FfiConverterTypeRedeemOnchainError{}

func (c FfiConverterTypeRedeemOnchainError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeRedeemOnchainError) Lower(value *RedeemOnchainError) RustBuffer {
	return LowerIntoRustBuffer[*RedeemOnchainError](c, value)
}

func (c FfiConverterTypeRedeemOnchainError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeRedeemOnchainError.Read()", errorID))
	}

}

func (c FfiConverterTypeRedeemOnchainError) Write(writer io.Writer, value *RedeemOnchainError) {
	switch variantValue := value.err.(type) {
	case *RedeemOnchainErrorGeneric:
		writeInt32(writer, 1)
	case *RedeemOnchainErrorServiceConnectivity:
		writeInt32(writer, 2)
	case *RedeemOnchainErrorInsufficientFunds:
		writeInt32(writer, 3)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeRedeemOnchainError.Write", value))
	}
}

type ReportIssueRequest interface {
	Destroy()
}
type ReportIssueRequestPaymentFailure struct {
	Data ReportPaymentFailureDetails
}

func (e ReportIssueRequestPaymentFailure) Destroy() {
	FfiDestroyerTypeReportPaymentFailureDetails{}.Destroy(e.Data)
}

type FfiConverterTypeReportIssueRequest struct{}

var FfiConverterTypeReportIssueRequestINSTANCE = FfiConverterTypeReportIssueRequest{}

func (c FfiConverterTypeReportIssueRequest) Lift(rb RustBufferI) ReportIssueRequest {
	return LiftFromRustBuffer[ReportIssueRequest](c, rb)
}

func (c FfiConverterTypeReportIssueRequest) Lower(value ReportIssueRequest) RustBuffer {
	return LowerIntoRustBuffer[ReportIssueRequest](c, value)
}
func (FfiConverterTypeReportIssueRequest) Read(reader io.Reader) ReportIssueRequest {
	id := readInt32(reader)
	switch id {
	case 1:
		return ReportIssueRequestPaymentFailure{
			FfiConverterTypeReportPaymentFailureDetailsINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeReportIssueRequest.Read()", id))
	}
}

func (FfiConverterTypeReportIssueRequest) Write(writer io.Writer, value ReportIssueRequest) {
	switch variant_value := value.(type) {
	case ReportIssueRequestPaymentFailure:
		writeInt32(writer, 1)
		FfiConverterTypeReportPaymentFailureDetailsINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeReportIssueRequest.Write", value))
	}
}

type FfiDestroyerTypeReportIssueRequest struct{}

func (_ FfiDestroyerTypeReportIssueRequest) Destroy(value ReportIssueRequest) {
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

type FfiConverterTypeReverseSwapStatus struct{}

var FfiConverterTypeReverseSwapStatusINSTANCE = FfiConverterTypeReverseSwapStatus{}

func (c FfiConverterTypeReverseSwapStatus) Lift(rb RustBufferI) ReverseSwapStatus {
	return LiftFromRustBuffer[ReverseSwapStatus](c, rb)
}

func (c FfiConverterTypeReverseSwapStatus) Lower(value ReverseSwapStatus) RustBuffer {
	return LowerIntoRustBuffer[ReverseSwapStatus](c, value)
}
func (FfiConverterTypeReverseSwapStatus) Read(reader io.Reader) ReverseSwapStatus {
	id := readInt32(reader)
	return ReverseSwapStatus(id)
}

func (FfiConverterTypeReverseSwapStatus) Write(writer io.Writer, value ReverseSwapStatus) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeReverseSwapStatus struct{}

func (_ FfiDestroyerTypeReverseSwapStatus) Destroy(value ReverseSwapStatus) {
}

type SdkError struct {
	err error
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
	return &SdkError{
		err: &SdkErrorGeneric{},
	}
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
	return &SdkError{
		err: &SdkErrorServiceConnectivity{},
	}
}

func (err SdkErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self SdkErrorServiceConnectivity) Is(target error) bool {
	return target == ErrSdkErrorServiceConnectivity
}

type FfiConverterTypeSdkError struct{}

var FfiConverterTypeSdkErrorINSTANCE = FfiConverterTypeSdkError{}

func (c FfiConverterTypeSdkError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeSdkError) Lower(value *SdkError) RustBuffer {
	return LowerIntoRustBuffer[*SdkError](c, value)
}

func (c FfiConverterTypeSdkError) Read(reader io.Reader) error {
	errorID := readUint32(reader)

	message := FfiConverterStringINSTANCE.Read(reader)
	switch errorID {
	case 1:
		return &SdkError{&SdkErrorGeneric{message}}
	case 2:
		return &SdkError{&SdkErrorServiceConnectivity{message}}
	default:
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeSdkError.Read()", errorID))
	}

}

func (c FfiConverterTypeSdkError) Write(writer io.Writer, value *SdkError) {
	switch variantValue := value.err.(type) {
	case *SdkErrorGeneric:
		writeInt32(writer, 1)
	case *SdkErrorServiceConnectivity:
		writeInt32(writer, 2)
	default:
		_ = variantValue
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeSdkError.Write", value))
	}
}

type SendOnchainError struct {
	err error
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
	return &SendOnchainError{
		err: &SendOnchainErrorGeneric{},
	}
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
	return &SendOnchainError{
		err: &SendOnchainErrorInvalidDestinationAddress{},
	}
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
	return &SendOnchainError{
		err: &SendOnchainErrorOutOfRange{},
	}
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
	return &SendOnchainError{
		err: &SendOnchainErrorPaymentFailed{},
	}
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
	return &SendOnchainError{
		err: &SendOnchainErrorPaymentTimeout{},
	}
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
	return &SendOnchainError{
		err: &SendOnchainErrorServiceConnectivity{},
	}
}

func (err SendOnchainErrorServiceConnectivity) Error() string {
	return fmt.Sprintf("ServiceConnectivity: %s", err.message)
}

func (self SendOnchainErrorServiceConnectivity) Is(target error) bool {
	return target == ErrSendOnchainErrorServiceConnectivity
}

type FfiConverterTypeSendOnchainError struct{}

var FfiConverterTypeSendOnchainErrorINSTANCE = FfiConverterTypeSendOnchainError{}

func (c FfiConverterTypeSendOnchainError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeSendOnchainError) Lower(value *SendOnchainError) RustBuffer {
	return LowerIntoRustBuffer[*SendOnchainError](c, value)
}

func (c FfiConverterTypeSendOnchainError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeSendOnchainError.Read()", errorID))
	}

}

func (c FfiConverterTypeSendOnchainError) Write(writer io.Writer, value *SendOnchainError) {
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
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeSendOnchainError.Write", value))
	}
}

type SendPaymentError struct {
	err error
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
	return &SendPaymentError{
		err: &SendPaymentErrorAlreadyPaid{},
	}
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
	return &SendPaymentError{
		err: &SendPaymentErrorGeneric{},
	}
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
	return &SendPaymentError{
		err: &SendPaymentErrorInvalidAmount{},
	}
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
	return &SendPaymentError{
		err: &SendPaymentErrorInvalidInvoice{},
	}
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
	return &SendPaymentError{
		err: &SendPaymentErrorInvoiceExpired{},
	}
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
	return &SendPaymentError{
		err: &SendPaymentErrorInvalidNetwork{},
	}
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
	return &SendPaymentError{
		err: &SendPaymentErrorPaymentFailed{},
	}
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
	return &SendPaymentError{
		err: &SendPaymentErrorPaymentTimeout{},
	}
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
	return &SendPaymentError{
		err: &SendPaymentErrorRouteNotFound{},
	}
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
	return &SendPaymentError{
		err: &SendPaymentErrorRouteTooExpensive{},
	}
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
	return &SendPaymentError{
		err: &SendPaymentErrorServiceConnectivity{},
	}
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
	return &SendPaymentError{
		err: &SendPaymentErrorInsufficientBalance{},
	}
}

func (err SendPaymentErrorInsufficientBalance) Error() string {
	return fmt.Sprintf("InsufficientBalance: %s", err.message)
}

func (self SendPaymentErrorInsufficientBalance) Is(target error) bool {
	return target == ErrSendPaymentErrorInsufficientBalance
}

type FfiConverterTypeSendPaymentError struct{}

var FfiConverterTypeSendPaymentErrorINSTANCE = FfiConverterTypeSendPaymentError{}

func (c FfiConverterTypeSendPaymentError) Lift(eb RustBufferI) error {
	return LiftFromRustBuffer[error](c, eb)
}

func (c FfiConverterTypeSendPaymentError) Lower(value *SendPaymentError) RustBuffer {
	return LowerIntoRustBuffer[*SendPaymentError](c, value)
}

func (c FfiConverterTypeSendPaymentError) Read(reader io.Reader) error {
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
		panic(fmt.Sprintf("Unknown error code %d in FfiConverterTypeSendPaymentError.Read()", errorID))
	}

}

func (c FfiConverterTypeSendPaymentError) Write(writer io.Writer, value *SendPaymentError) {
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
		panic(fmt.Sprintf("invalid error value `%v` in FfiConverterTypeSendPaymentError.Write", value))
	}
}

type SuccessActionProcessed interface {
	Destroy()
}
type SuccessActionProcessedAes struct {
	Result AesSuccessActionDataResult
}

func (e SuccessActionProcessedAes) Destroy() {
	FfiDestroyerTypeAesSuccessActionDataResult{}.Destroy(e.Result)
}

type SuccessActionProcessedMessage struct {
	Data MessageSuccessActionData
}

func (e SuccessActionProcessedMessage) Destroy() {
	FfiDestroyerTypeMessageSuccessActionData{}.Destroy(e.Data)
}

type SuccessActionProcessedUrl struct {
	Data UrlSuccessActionData
}

func (e SuccessActionProcessedUrl) Destroy() {
	FfiDestroyerTypeUrlSuccessActionData{}.Destroy(e.Data)
}

type FfiConverterTypeSuccessActionProcessed struct{}

var FfiConverterTypeSuccessActionProcessedINSTANCE = FfiConverterTypeSuccessActionProcessed{}

func (c FfiConverterTypeSuccessActionProcessed) Lift(rb RustBufferI) SuccessActionProcessed {
	return LiftFromRustBuffer[SuccessActionProcessed](c, rb)
}

func (c FfiConverterTypeSuccessActionProcessed) Lower(value SuccessActionProcessed) RustBuffer {
	return LowerIntoRustBuffer[SuccessActionProcessed](c, value)
}
func (FfiConverterTypeSuccessActionProcessed) Read(reader io.Reader) SuccessActionProcessed {
	id := readInt32(reader)
	switch id {
	case 1:
		return SuccessActionProcessedAes{
			FfiConverterTypeAesSuccessActionDataResultINSTANCE.Read(reader),
		}
	case 2:
		return SuccessActionProcessedMessage{
			FfiConverterTypeMessageSuccessActionDataINSTANCE.Read(reader),
		}
	case 3:
		return SuccessActionProcessedUrl{
			FfiConverterTypeUrlSuccessActionDataINSTANCE.Read(reader),
		}
	default:
		panic(fmt.Sprintf("invalid enum value %v in FfiConverterTypeSuccessActionProcessed.Read()", id))
	}
}

func (FfiConverterTypeSuccessActionProcessed) Write(writer io.Writer, value SuccessActionProcessed) {
	switch variant_value := value.(type) {
	case SuccessActionProcessedAes:
		writeInt32(writer, 1)
		FfiConverterTypeAesSuccessActionDataResultINSTANCE.Write(writer, variant_value.Result)
	case SuccessActionProcessedMessage:
		writeInt32(writer, 2)
		FfiConverterTypeMessageSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	case SuccessActionProcessedUrl:
		writeInt32(writer, 3)
		FfiConverterTypeUrlSuccessActionDataINSTANCE.Write(writer, variant_value.Data)
	default:
		_ = variant_value
		panic(fmt.Sprintf("invalid enum value `%v` in FfiConverterTypeSuccessActionProcessed.Write", value))
	}
}

type FfiDestroyerTypeSuccessActionProcessed struct{}

func (_ FfiDestroyerTypeSuccessActionProcessed) Destroy(value SuccessActionProcessed) {
	value.Destroy()
}

type SwapAmountType uint

const (
	SwapAmountTypeSend    SwapAmountType = 1
	SwapAmountTypeReceive SwapAmountType = 2
)

type FfiConverterTypeSwapAmountType struct{}

var FfiConverterTypeSwapAmountTypeINSTANCE = FfiConverterTypeSwapAmountType{}

func (c FfiConverterTypeSwapAmountType) Lift(rb RustBufferI) SwapAmountType {
	return LiftFromRustBuffer[SwapAmountType](c, rb)
}

func (c FfiConverterTypeSwapAmountType) Lower(value SwapAmountType) RustBuffer {
	return LowerIntoRustBuffer[SwapAmountType](c, value)
}
func (FfiConverterTypeSwapAmountType) Read(reader io.Reader) SwapAmountType {
	id := readInt32(reader)
	return SwapAmountType(id)
}

func (FfiConverterTypeSwapAmountType) Write(writer io.Writer, value SwapAmountType) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeSwapAmountType struct{}

func (_ FfiDestroyerTypeSwapAmountType) Destroy(value SwapAmountType) {
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

type FfiConverterTypeSwapStatus struct{}

var FfiConverterTypeSwapStatusINSTANCE = FfiConverterTypeSwapStatus{}

func (c FfiConverterTypeSwapStatus) Lift(rb RustBufferI) SwapStatus {
	return LiftFromRustBuffer[SwapStatus](c, rb)
}

func (c FfiConverterTypeSwapStatus) Lower(value SwapStatus) RustBuffer {
	return LowerIntoRustBuffer[SwapStatus](c, value)
}
func (FfiConverterTypeSwapStatus) Read(reader io.Reader) SwapStatus {
	id := readInt32(reader)
	return SwapStatus(id)
}

func (FfiConverterTypeSwapStatus) Write(writer io.Writer, value SwapStatus) {
	writeInt32(writer, int32(value))
}

type FfiDestroyerTypeSwapStatus struct{}

func (_ FfiDestroyerTypeSwapStatus) Destroy(value SwapStatus) {
}

type uniffiCallbackResult C.int32_t

const (
	uniffiIdxCallbackFree               uniffiCallbackResult = 0
	uniffiCallbackResultSuccess         uniffiCallbackResult = 0
	uniffiCallbackResultError           uniffiCallbackResult = 1
	uniffiCallbackUnexpectedResultError uniffiCallbackResult = 2
	uniffiCallbackCancelled             uniffiCallbackResult = 3
)

type concurrentHandleMap[T any] struct {
	leftMap       map[uint64]*T
	rightMap      map[*T]uint64
	currentHandle uint64
	lock          sync.RWMutex
}

func newConcurrentHandleMap[T any]() *concurrentHandleMap[T] {
	return &concurrentHandleMap[T]{
		leftMap:  map[uint64]*T{},
		rightMap: map[*T]uint64{},
	}
}

func (cm *concurrentHandleMap[T]) insert(obj *T) uint64 {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if existingHandle, ok := cm.rightMap[obj]; ok {
		return existingHandle
	}
	cm.currentHandle = cm.currentHandle + 1
	cm.leftMap[cm.currentHandle] = obj
	cm.rightMap[obj] = cm.currentHandle
	return cm.currentHandle
}

func (cm *concurrentHandleMap[T]) remove(handle uint64) bool {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if val, ok := cm.leftMap[handle]; ok {
		delete(cm.leftMap, handle)
		delete(cm.rightMap, val)
	}
	return false
}

func (cm *concurrentHandleMap[T]) tryGet(handle uint64) (*T, bool) {
	cm.lock.RLock()
	defer cm.lock.RUnlock()

	val, ok := cm.leftMap[handle]
	return val, ok
}

type FfiConverterCallbackInterface[CallbackInterface any] struct {
	handleMap *concurrentHandleMap[CallbackInterface]
}

func (c *FfiConverterCallbackInterface[CallbackInterface]) drop(handle uint64) RustBuffer {
	c.handleMap.remove(handle)
	return RustBuffer{}
}

func (c *FfiConverterCallbackInterface[CallbackInterface]) Lift(handle uint64) CallbackInterface {
	val, ok := c.handleMap.tryGet(handle)
	if !ok {
		panic(fmt.Errorf("no callback in handle map: %d", handle))
	}
	return *val
}

func (c *FfiConverterCallbackInterface[CallbackInterface]) Read(reader io.Reader) CallbackInterface {
	return c.Lift(readUint64(reader))
}

func (c *FfiConverterCallbackInterface[CallbackInterface]) Lower(value CallbackInterface) C.uint64_t {
	return C.uint64_t(c.handleMap.insert(&value))
}

func (c *FfiConverterCallbackInterface[CallbackInterface]) Write(writer io.Writer, value CallbackInterface) {
	writeUint64(writer, uint64(c.Lower(value)))
}

type EventListener interface {
	OnEvent(e BreezEvent)
}

// foreignCallbackCallbackInterfaceEventListener cannot be callable be a compiled function at a same time
type foreignCallbackCallbackInterfaceEventListener struct{}

//export breez_sdk_bindings_cgo_EventListener
func breez_sdk_bindings_cgo_EventListener(handle C.uint64_t, method C.int32_t, argsPtr *C.uint8_t, argsLen C.int32_t, outBuf *C.RustBuffer) C.int32_t {
	cb := FfiConverterCallbackInterfaceEventListenerINSTANCE.Lift(uint64(handle))
	switch method {
	case 0:
		// 0 means Rust is done with the callback, and the callback
		// can be dropped by the foreign language.
		*outBuf = FfiConverterCallbackInterfaceEventListenerINSTANCE.drop(uint64(handle))
		// See docs of ForeignCallback in `uniffi/src/ffi/foreigncallbacks.rs`
		return C.int32_t(uniffiIdxCallbackFree)

	case 1:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceEventListener{}.InvokeOnEvent(cb, args, outBuf)
		return C.int32_t(result)

	default:
		// This should never happen, because an out of bounds method index won't
		// ever be used. Once we can catch errors, we should return an InternalException.
		// https://github.com/mozilla/uniffi-rs/issues/351
		return C.int32_t(uniffiCallbackUnexpectedResultError)
	}
}

func (foreignCallbackCallbackInterfaceEventListener) InvokeOnEvent(callback EventListener, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	reader := bytes.NewReader(args)
	callback.OnEvent(FfiConverterTypeBreezEventINSTANCE.Read(reader))

	return uniffiCallbackResultSuccess
}

type FfiConverterCallbackInterfaceEventListener struct {
	FfiConverterCallbackInterface[EventListener]
}

var FfiConverterCallbackInterfaceEventListenerINSTANCE = &FfiConverterCallbackInterfaceEventListener{
	FfiConverterCallbackInterface: FfiConverterCallbackInterface[EventListener]{
		handleMap: newConcurrentHandleMap[EventListener](),
	},
}

// This is a static function because only 1 instance is supported for registering
func (c *FfiConverterCallbackInterfaceEventListener) register() {
	rustCall(func(status *C.RustCallStatus) int32 {
		C.uniffi_breez_sdk_bindings_fn_init_callback_eventlistener(C.ForeignCallback(C.breez_sdk_bindings_cgo_EventListener), status)
		return 0
	})
}

type FfiDestroyerCallbackInterfaceEventListener struct{}

func (FfiDestroyerCallbackInterfaceEventListener) Destroy(value EventListener) {
}

type LogStream interface {
	Log(l LogEntry)
}

// foreignCallbackCallbackInterfaceLogStream cannot be callable be a compiled function at a same time
type foreignCallbackCallbackInterfaceLogStream struct{}

//export breez_sdk_bindings_cgo_LogStream
func breez_sdk_bindings_cgo_LogStream(handle C.uint64_t, method C.int32_t, argsPtr *C.uint8_t, argsLen C.int32_t, outBuf *C.RustBuffer) C.int32_t {
	cb := FfiConverterCallbackInterfaceLogStreamINSTANCE.Lift(uint64(handle))
	switch method {
	case 0:
		// 0 means Rust is done with the callback, and the callback
		// can be dropped by the foreign language.
		*outBuf = FfiConverterCallbackInterfaceLogStreamINSTANCE.drop(uint64(handle))
		// See docs of ForeignCallback in `uniffi/src/ffi/foreigncallbacks.rs`
		return C.int32_t(uniffiIdxCallbackFree)

	case 1:
		var result uniffiCallbackResult
		args := unsafe.Slice((*byte)(argsPtr), argsLen)
		result = foreignCallbackCallbackInterfaceLogStream{}.InvokeLog(cb, args, outBuf)
		return C.int32_t(result)

	default:
		// This should never happen, because an out of bounds method index won't
		// ever be used. Once we can catch errors, we should return an InternalException.
		// https://github.com/mozilla/uniffi-rs/issues/351
		return C.int32_t(uniffiCallbackUnexpectedResultError)
	}
}

func (foreignCallbackCallbackInterfaceLogStream) InvokeLog(callback LogStream, args []byte, outBuf *C.RustBuffer) uniffiCallbackResult {
	reader := bytes.NewReader(args)
	callback.Log(FfiConverterTypeLogEntryINSTANCE.Read(reader))

	return uniffiCallbackResultSuccess
}

type FfiConverterCallbackInterfaceLogStream struct {
	FfiConverterCallbackInterface[LogStream]
}

var FfiConverterCallbackInterfaceLogStreamINSTANCE = &FfiConverterCallbackInterfaceLogStream{
	FfiConverterCallbackInterface: FfiConverterCallbackInterface[LogStream]{
		handleMap: newConcurrentHandleMap[LogStream](),
	},
}

// This is a static function because only 1 instance is supported for registering
func (c *FfiConverterCallbackInterfaceLogStream) register() {
	rustCall(func(status *C.RustCallStatus) int32 {
		C.uniffi_breez_sdk_bindings_fn_init_callback_logstream(C.ForeignCallback(C.breez_sdk_bindings_cgo_LogStream), status)
		return 0
	})
}

type FfiDestroyerCallbackInterfaceLogStream struct{}

func (FfiDestroyerCallbackInterfaceLogStream) Destroy(value LogStream) {
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

func (c FfiConverterOptionalUint32) Lower(value *uint32) RustBuffer {
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

func (c FfiConverterOptionalUint64) Lower(value *uint64) RustBuffer {
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

func (c FfiConverterOptionalInt64) Lower(value *int64) RustBuffer {
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

func (c FfiConverterOptionalBool) Lower(value *bool) RustBuffer {
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

func (c FfiConverterOptionalString) Lower(value *string) RustBuffer {
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

type FfiConverterOptionalTypeGreenlightCredentials struct{}

var FfiConverterOptionalTypeGreenlightCredentialsINSTANCE = FfiConverterOptionalTypeGreenlightCredentials{}

func (c FfiConverterOptionalTypeGreenlightCredentials) Lift(rb RustBufferI) *GreenlightCredentials {
	return LiftFromRustBuffer[*GreenlightCredentials](c, rb)
}

func (_ FfiConverterOptionalTypeGreenlightCredentials) Read(reader io.Reader) *GreenlightCredentials {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeGreenlightCredentialsINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeGreenlightCredentials) Lower(value *GreenlightCredentials) RustBuffer {
	return LowerIntoRustBuffer[*GreenlightCredentials](c, value)
}

func (_ FfiConverterOptionalTypeGreenlightCredentials) Write(writer io.Writer, value *GreenlightCredentials) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeGreenlightCredentialsINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeGreenlightCredentials struct{}

func (_ FfiDestroyerOptionalTypeGreenlightCredentials) Destroy(value *GreenlightCredentials) {
	if value != nil {
		FfiDestroyerTypeGreenlightCredentials{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeLNInvoice struct{}

var FfiConverterOptionalTypeLNInvoiceINSTANCE = FfiConverterOptionalTypeLNInvoice{}

func (c FfiConverterOptionalTypeLNInvoice) Lift(rb RustBufferI) *LnInvoice {
	return LiftFromRustBuffer[*LnInvoice](c, rb)
}

func (_ FfiConverterOptionalTypeLNInvoice) Read(reader io.Reader) *LnInvoice {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeLNInvoiceINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeLNInvoice) Lower(value *LnInvoice) RustBuffer {
	return LowerIntoRustBuffer[*LnInvoice](c, value)
}

func (_ FfiConverterOptionalTypeLNInvoice) Write(writer io.Writer, value *LnInvoice) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeLNInvoiceINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeLnInvoice struct{}

func (_ FfiDestroyerOptionalTypeLnInvoice) Destroy(value *LnInvoice) {
	if value != nil {
		FfiDestroyerTypeLnInvoice{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeLspInformation struct{}

var FfiConverterOptionalTypeLspInformationINSTANCE = FfiConverterOptionalTypeLspInformation{}

func (c FfiConverterOptionalTypeLspInformation) Lift(rb RustBufferI) *LspInformation {
	return LiftFromRustBuffer[*LspInformation](c, rb)
}

func (_ FfiConverterOptionalTypeLspInformation) Read(reader io.Reader) *LspInformation {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeLspInformationINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeLspInformation) Lower(value *LspInformation) RustBuffer {
	return LowerIntoRustBuffer[*LspInformation](c, value)
}

func (_ FfiConverterOptionalTypeLspInformation) Write(writer io.Writer, value *LspInformation) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeLspInformationINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeLspInformation struct{}

func (_ FfiDestroyerOptionalTypeLspInformation) Destroy(value *LspInformation) {
	if value != nil {
		FfiDestroyerTypeLspInformation{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeOpeningFeeParams struct{}

var FfiConverterOptionalTypeOpeningFeeParamsINSTANCE = FfiConverterOptionalTypeOpeningFeeParams{}

func (c FfiConverterOptionalTypeOpeningFeeParams) Lift(rb RustBufferI) *OpeningFeeParams {
	return LiftFromRustBuffer[*OpeningFeeParams](c, rb)
}

func (_ FfiConverterOptionalTypeOpeningFeeParams) Read(reader io.Reader) *OpeningFeeParams {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeOpeningFeeParamsINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeOpeningFeeParams) Lower(value *OpeningFeeParams) RustBuffer {
	return LowerIntoRustBuffer[*OpeningFeeParams](c, value)
}

func (_ FfiConverterOptionalTypeOpeningFeeParams) Write(writer io.Writer, value *OpeningFeeParams) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeOpeningFeeParamsINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeOpeningFeeParams struct{}

func (_ FfiDestroyerOptionalTypeOpeningFeeParams) Destroy(value *OpeningFeeParams) {
	if value != nil {
		FfiDestroyerTypeOpeningFeeParams{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypePayment struct{}

var FfiConverterOptionalTypePaymentINSTANCE = FfiConverterOptionalTypePayment{}

func (c FfiConverterOptionalTypePayment) Lift(rb RustBufferI) *Payment {
	return LiftFromRustBuffer[*Payment](c, rb)
}

func (_ FfiConverterOptionalTypePayment) Read(reader io.Reader) *Payment {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypePaymentINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypePayment) Lower(value *Payment) RustBuffer {
	return LowerIntoRustBuffer[*Payment](c, value)
}

func (_ FfiConverterOptionalTypePayment) Write(writer io.Writer, value *Payment) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypePaymentINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypePayment struct{}

func (_ FfiDestroyerOptionalTypePayment) Destroy(value *Payment) {
	if value != nil {
		FfiDestroyerTypePayment{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeReverseSwapInfo struct{}

var FfiConverterOptionalTypeReverseSwapInfoINSTANCE = FfiConverterOptionalTypeReverseSwapInfo{}

func (c FfiConverterOptionalTypeReverseSwapInfo) Lift(rb RustBufferI) *ReverseSwapInfo {
	return LiftFromRustBuffer[*ReverseSwapInfo](c, rb)
}

func (_ FfiConverterOptionalTypeReverseSwapInfo) Read(reader io.Reader) *ReverseSwapInfo {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeReverseSwapInfoINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeReverseSwapInfo) Lower(value *ReverseSwapInfo) RustBuffer {
	return LowerIntoRustBuffer[*ReverseSwapInfo](c, value)
}

func (_ FfiConverterOptionalTypeReverseSwapInfo) Write(writer io.Writer, value *ReverseSwapInfo) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeReverseSwapInfoINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeReverseSwapInfo struct{}

func (_ FfiDestroyerOptionalTypeReverseSwapInfo) Destroy(value *ReverseSwapInfo) {
	if value != nil {
		FfiDestroyerTypeReverseSwapInfo{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeSwapInfo struct{}

var FfiConverterOptionalTypeSwapInfoINSTANCE = FfiConverterOptionalTypeSwapInfo{}

func (c FfiConverterOptionalTypeSwapInfo) Lift(rb RustBufferI) *SwapInfo {
	return LiftFromRustBuffer[*SwapInfo](c, rb)
}

func (_ FfiConverterOptionalTypeSwapInfo) Read(reader io.Reader) *SwapInfo {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeSwapInfoINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeSwapInfo) Lower(value *SwapInfo) RustBuffer {
	return LowerIntoRustBuffer[*SwapInfo](c, value)
}

func (_ FfiConverterOptionalTypeSwapInfo) Write(writer io.Writer, value *SwapInfo) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeSwapInfoINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeSwapInfo struct{}

func (_ FfiDestroyerOptionalTypeSwapInfo) Destroy(value *SwapInfo) {
	if value != nil {
		FfiDestroyerTypeSwapInfo{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeSymbol struct{}

var FfiConverterOptionalTypeSymbolINSTANCE = FfiConverterOptionalTypeSymbol{}

func (c FfiConverterOptionalTypeSymbol) Lift(rb RustBufferI) *Symbol {
	return LiftFromRustBuffer[*Symbol](c, rb)
}

func (_ FfiConverterOptionalTypeSymbol) Read(reader io.Reader) *Symbol {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeSymbolINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeSymbol) Lower(value *Symbol) RustBuffer {
	return LowerIntoRustBuffer[*Symbol](c, value)
}

func (_ FfiConverterOptionalTypeSymbol) Write(writer io.Writer, value *Symbol) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeSymbolINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeSymbol struct{}

func (_ FfiDestroyerOptionalTypeSymbol) Destroy(value *Symbol) {
	if value != nil {
		FfiDestroyerTypeSymbol{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeNodeCredentials struct{}

var FfiConverterOptionalTypeNodeCredentialsINSTANCE = FfiConverterOptionalTypeNodeCredentials{}

func (c FfiConverterOptionalTypeNodeCredentials) Lift(rb RustBufferI) *NodeCredentials {
	return LiftFromRustBuffer[*NodeCredentials](c, rb)
}

func (_ FfiConverterOptionalTypeNodeCredentials) Read(reader io.Reader) *NodeCredentials {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeNodeCredentialsINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeNodeCredentials) Lower(value *NodeCredentials) RustBuffer {
	return LowerIntoRustBuffer[*NodeCredentials](c, value)
}

func (_ FfiConverterOptionalTypeNodeCredentials) Write(writer io.Writer, value *NodeCredentials) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeNodeCredentialsINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeNodeCredentials struct{}

func (_ FfiDestroyerOptionalTypeNodeCredentials) Destroy(value *NodeCredentials) {
	if value != nil {
		FfiDestroyerTypeNodeCredentials{}.Destroy(*value)
	}
}

type FfiConverterOptionalTypeSuccessActionProcessed struct{}

var FfiConverterOptionalTypeSuccessActionProcessedINSTANCE = FfiConverterOptionalTypeSuccessActionProcessed{}

func (c FfiConverterOptionalTypeSuccessActionProcessed) Lift(rb RustBufferI) *SuccessActionProcessed {
	return LiftFromRustBuffer[*SuccessActionProcessed](c, rb)
}

func (_ FfiConverterOptionalTypeSuccessActionProcessed) Read(reader io.Reader) *SuccessActionProcessed {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterTypeSuccessActionProcessedINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalTypeSuccessActionProcessed) Lower(value *SuccessActionProcessed) RustBuffer {
	return LowerIntoRustBuffer[*SuccessActionProcessed](c, value)
}

func (_ FfiConverterOptionalTypeSuccessActionProcessed) Write(writer io.Writer, value *SuccessActionProcessed) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterTypeSuccessActionProcessedINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalTypeSuccessActionProcessed struct{}

func (_ FfiDestroyerOptionalTypeSuccessActionProcessed) Destroy(value *SuccessActionProcessed) {
	if value != nil {
		FfiDestroyerTypeSuccessActionProcessed{}.Destroy(*value)
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

func (c FfiConverterOptionalSequenceUint8) Lower(value *[]uint8) RustBuffer {
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

func (c FfiConverterOptionalSequenceString) Lower(value *[]string) RustBuffer {
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

type FfiConverterOptionalSequenceTypeMetadataFilter struct{}

var FfiConverterOptionalSequenceTypeMetadataFilterINSTANCE = FfiConverterOptionalSequenceTypeMetadataFilter{}

func (c FfiConverterOptionalSequenceTypeMetadataFilter) Lift(rb RustBufferI) *[]MetadataFilter {
	return LiftFromRustBuffer[*[]MetadataFilter](c, rb)
}

func (_ FfiConverterOptionalSequenceTypeMetadataFilter) Read(reader io.Reader) *[]MetadataFilter {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceTypeMetadataFilterINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceTypeMetadataFilter) Lower(value *[]MetadataFilter) RustBuffer {
	return LowerIntoRustBuffer[*[]MetadataFilter](c, value)
}

func (_ FfiConverterOptionalSequenceTypeMetadataFilter) Write(writer io.Writer, value *[]MetadataFilter) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceTypeMetadataFilterINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceTypeMetadataFilter struct{}

func (_ FfiDestroyerOptionalSequenceTypeMetadataFilter) Destroy(value *[]MetadataFilter) {
	if value != nil {
		FfiDestroyerSequenceTypeMetadataFilter{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceTypeTlvEntry struct{}

var FfiConverterOptionalSequenceTypeTlvEntryINSTANCE = FfiConverterOptionalSequenceTypeTlvEntry{}

func (c FfiConverterOptionalSequenceTypeTlvEntry) Lift(rb RustBufferI) *[]TlvEntry {
	return LiftFromRustBuffer[*[]TlvEntry](c, rb)
}

func (_ FfiConverterOptionalSequenceTypeTlvEntry) Read(reader io.Reader) *[]TlvEntry {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceTypeTlvEntryINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceTypeTlvEntry) Lower(value *[]TlvEntry) RustBuffer {
	return LowerIntoRustBuffer[*[]TlvEntry](c, value)
}

func (_ FfiConverterOptionalSequenceTypeTlvEntry) Write(writer io.Writer, value *[]TlvEntry) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceTypeTlvEntryINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceTypeTlvEntry struct{}

func (_ FfiDestroyerOptionalSequenceTypeTlvEntry) Destroy(value *[]TlvEntry) {
	if value != nil {
		FfiDestroyerSequenceTypeTlvEntry{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceTypePaymentTypeFilter struct{}

var FfiConverterOptionalSequenceTypePaymentTypeFilterINSTANCE = FfiConverterOptionalSequenceTypePaymentTypeFilter{}

func (c FfiConverterOptionalSequenceTypePaymentTypeFilter) Lift(rb RustBufferI) *[]PaymentTypeFilter {
	return LiftFromRustBuffer[*[]PaymentTypeFilter](c, rb)
}

func (_ FfiConverterOptionalSequenceTypePaymentTypeFilter) Read(reader io.Reader) *[]PaymentTypeFilter {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceTypePaymentTypeFilterINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceTypePaymentTypeFilter) Lower(value *[]PaymentTypeFilter) RustBuffer {
	return LowerIntoRustBuffer[*[]PaymentTypeFilter](c, value)
}

func (_ FfiConverterOptionalSequenceTypePaymentTypeFilter) Write(writer io.Writer, value *[]PaymentTypeFilter) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceTypePaymentTypeFilterINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceTypePaymentTypeFilter struct{}

func (_ FfiDestroyerOptionalSequenceTypePaymentTypeFilter) Destroy(value *[]PaymentTypeFilter) {
	if value != nil {
		FfiDestroyerSequenceTypePaymentTypeFilter{}.Destroy(*value)
	}
}

type FfiConverterOptionalSequenceTypeSwapStatus struct{}

var FfiConverterOptionalSequenceTypeSwapStatusINSTANCE = FfiConverterOptionalSequenceTypeSwapStatus{}

func (c FfiConverterOptionalSequenceTypeSwapStatus) Lift(rb RustBufferI) *[]SwapStatus {
	return LiftFromRustBuffer[*[]SwapStatus](c, rb)
}

func (_ FfiConverterOptionalSequenceTypeSwapStatus) Read(reader io.Reader) *[]SwapStatus {
	if readInt8(reader) == 0 {
		return nil
	}
	temp := FfiConverterSequenceTypeSwapStatusINSTANCE.Read(reader)
	return &temp
}

func (c FfiConverterOptionalSequenceTypeSwapStatus) Lower(value *[]SwapStatus) RustBuffer {
	return LowerIntoRustBuffer[*[]SwapStatus](c, value)
}

func (_ FfiConverterOptionalSequenceTypeSwapStatus) Write(writer io.Writer, value *[]SwapStatus) {
	if value == nil {
		writeInt8(writer, 0)
	} else {
		writeInt8(writer, 1)
		FfiConverterSequenceTypeSwapStatusINSTANCE.Write(writer, *value)
	}
}

type FfiDestroyerOptionalSequenceTypeSwapStatus struct{}

func (_ FfiDestroyerOptionalSequenceTypeSwapStatus) Destroy(value *[]SwapStatus) {
	if value != nil {
		FfiDestroyerSequenceTypeSwapStatus{}.Destroy(*value)
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

func (c FfiConverterSequenceUint8) Lower(value []uint8) RustBuffer {
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

func (c FfiConverterSequenceString) Lower(value []string) RustBuffer {
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

type FfiConverterSequenceTypeFiatCurrency struct{}

var FfiConverterSequenceTypeFiatCurrencyINSTANCE = FfiConverterSequenceTypeFiatCurrency{}

func (c FfiConverterSequenceTypeFiatCurrency) Lift(rb RustBufferI) []FiatCurrency {
	return LiftFromRustBuffer[[]FiatCurrency](c, rb)
}

func (c FfiConverterSequenceTypeFiatCurrency) Read(reader io.Reader) []FiatCurrency {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]FiatCurrency, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeFiatCurrencyINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeFiatCurrency) Lower(value []FiatCurrency) RustBuffer {
	return LowerIntoRustBuffer[[]FiatCurrency](c, value)
}

func (c FfiConverterSequenceTypeFiatCurrency) Write(writer io.Writer, value []FiatCurrency) {
	if len(value) > math.MaxInt32 {
		panic("[]FiatCurrency is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeFiatCurrencyINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeFiatCurrency struct{}

func (FfiDestroyerSequenceTypeFiatCurrency) Destroy(sequence []FiatCurrency) {
	for _, value := range sequence {
		FfiDestroyerTypeFiatCurrency{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeLocaleOverrides struct{}

var FfiConverterSequenceTypeLocaleOverridesINSTANCE = FfiConverterSequenceTypeLocaleOverrides{}

func (c FfiConverterSequenceTypeLocaleOverrides) Lift(rb RustBufferI) []LocaleOverrides {
	return LiftFromRustBuffer[[]LocaleOverrides](c, rb)
}

func (c FfiConverterSequenceTypeLocaleOverrides) Read(reader io.Reader) []LocaleOverrides {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LocaleOverrides, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeLocaleOverridesINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeLocaleOverrides) Lower(value []LocaleOverrides) RustBuffer {
	return LowerIntoRustBuffer[[]LocaleOverrides](c, value)
}

func (c FfiConverterSequenceTypeLocaleOverrides) Write(writer io.Writer, value []LocaleOverrides) {
	if len(value) > math.MaxInt32 {
		panic("[]LocaleOverrides is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeLocaleOverridesINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeLocaleOverrides struct{}

func (FfiDestroyerSequenceTypeLocaleOverrides) Destroy(sequence []LocaleOverrides) {
	for _, value := range sequence {
		FfiDestroyerTypeLocaleOverrides{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeLocalizedName struct{}

var FfiConverterSequenceTypeLocalizedNameINSTANCE = FfiConverterSequenceTypeLocalizedName{}

func (c FfiConverterSequenceTypeLocalizedName) Lift(rb RustBufferI) []LocalizedName {
	return LiftFromRustBuffer[[]LocalizedName](c, rb)
}

func (c FfiConverterSequenceTypeLocalizedName) Read(reader io.Reader) []LocalizedName {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LocalizedName, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeLocalizedNameINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeLocalizedName) Lower(value []LocalizedName) RustBuffer {
	return LowerIntoRustBuffer[[]LocalizedName](c, value)
}

func (c FfiConverterSequenceTypeLocalizedName) Write(writer io.Writer, value []LocalizedName) {
	if len(value) > math.MaxInt32 {
		panic("[]LocalizedName is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeLocalizedNameINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeLocalizedName struct{}

func (FfiDestroyerSequenceTypeLocalizedName) Destroy(sequence []LocalizedName) {
	for _, value := range sequence {
		FfiDestroyerTypeLocalizedName{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeLspInformation struct{}

var FfiConverterSequenceTypeLspInformationINSTANCE = FfiConverterSequenceTypeLspInformation{}

func (c FfiConverterSequenceTypeLspInformation) Lift(rb RustBufferI) []LspInformation {
	return LiftFromRustBuffer[[]LspInformation](c, rb)
}

func (c FfiConverterSequenceTypeLspInformation) Read(reader io.Reader) []LspInformation {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]LspInformation, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeLspInformationINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeLspInformation) Lower(value []LspInformation) RustBuffer {
	return LowerIntoRustBuffer[[]LspInformation](c, value)
}

func (c FfiConverterSequenceTypeLspInformation) Write(writer io.Writer, value []LspInformation) {
	if len(value) > math.MaxInt32 {
		panic("[]LspInformation is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeLspInformationINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeLspInformation struct{}

func (FfiDestroyerSequenceTypeLspInformation) Destroy(sequence []LspInformation) {
	for _, value := range sequence {
		FfiDestroyerTypeLspInformation{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeMetadataFilter struct{}

var FfiConverterSequenceTypeMetadataFilterINSTANCE = FfiConverterSequenceTypeMetadataFilter{}

func (c FfiConverterSequenceTypeMetadataFilter) Lift(rb RustBufferI) []MetadataFilter {
	return LiftFromRustBuffer[[]MetadataFilter](c, rb)
}

func (c FfiConverterSequenceTypeMetadataFilter) Read(reader io.Reader) []MetadataFilter {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]MetadataFilter, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeMetadataFilterINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeMetadataFilter) Lower(value []MetadataFilter) RustBuffer {
	return LowerIntoRustBuffer[[]MetadataFilter](c, value)
}

func (c FfiConverterSequenceTypeMetadataFilter) Write(writer io.Writer, value []MetadataFilter) {
	if len(value) > math.MaxInt32 {
		panic("[]MetadataFilter is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeMetadataFilterINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeMetadataFilter struct{}

func (FfiDestroyerSequenceTypeMetadataFilter) Destroy(sequence []MetadataFilter) {
	for _, value := range sequence {
		FfiDestroyerTypeMetadataFilter{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeOpeningFeeParams struct{}

var FfiConverterSequenceTypeOpeningFeeParamsINSTANCE = FfiConverterSequenceTypeOpeningFeeParams{}

func (c FfiConverterSequenceTypeOpeningFeeParams) Lift(rb RustBufferI) []OpeningFeeParams {
	return LiftFromRustBuffer[[]OpeningFeeParams](c, rb)
}

func (c FfiConverterSequenceTypeOpeningFeeParams) Read(reader io.Reader) []OpeningFeeParams {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]OpeningFeeParams, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeOpeningFeeParamsINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeOpeningFeeParams) Lower(value []OpeningFeeParams) RustBuffer {
	return LowerIntoRustBuffer[[]OpeningFeeParams](c, value)
}

func (c FfiConverterSequenceTypeOpeningFeeParams) Write(writer io.Writer, value []OpeningFeeParams) {
	if len(value) > math.MaxInt32 {
		panic("[]OpeningFeeParams is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeOpeningFeeParamsINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeOpeningFeeParams struct{}

func (FfiDestroyerSequenceTypeOpeningFeeParams) Destroy(sequence []OpeningFeeParams) {
	for _, value := range sequence {
		FfiDestroyerTypeOpeningFeeParams{}.Destroy(value)
	}
}

type FfiConverterSequenceTypePayment struct{}

var FfiConverterSequenceTypePaymentINSTANCE = FfiConverterSequenceTypePayment{}

func (c FfiConverterSequenceTypePayment) Lift(rb RustBufferI) []Payment {
	return LiftFromRustBuffer[[]Payment](c, rb)
}

func (c FfiConverterSequenceTypePayment) Read(reader io.Reader) []Payment {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]Payment, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypePaymentINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypePayment) Lower(value []Payment) RustBuffer {
	return LowerIntoRustBuffer[[]Payment](c, value)
}

func (c FfiConverterSequenceTypePayment) Write(writer io.Writer, value []Payment) {
	if len(value) > math.MaxInt32 {
		panic("[]Payment is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypePaymentINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypePayment struct{}

func (FfiDestroyerSequenceTypePayment) Destroy(sequence []Payment) {
	for _, value := range sequence {
		FfiDestroyerTypePayment{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeRate struct{}

var FfiConverterSequenceTypeRateINSTANCE = FfiConverterSequenceTypeRate{}

func (c FfiConverterSequenceTypeRate) Lift(rb RustBufferI) []Rate {
	return LiftFromRustBuffer[[]Rate](c, rb)
}

func (c FfiConverterSequenceTypeRate) Read(reader io.Reader) []Rate {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]Rate, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeRateINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeRate) Lower(value []Rate) RustBuffer {
	return LowerIntoRustBuffer[[]Rate](c, value)
}

func (c FfiConverterSequenceTypeRate) Write(writer io.Writer, value []Rate) {
	if len(value) > math.MaxInt32 {
		panic("[]Rate is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeRateINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeRate struct{}

func (FfiDestroyerSequenceTypeRate) Destroy(sequence []Rate) {
	for _, value := range sequence {
		FfiDestroyerTypeRate{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeReverseSwapInfo struct{}

var FfiConverterSequenceTypeReverseSwapInfoINSTANCE = FfiConverterSequenceTypeReverseSwapInfo{}

func (c FfiConverterSequenceTypeReverseSwapInfo) Lift(rb RustBufferI) []ReverseSwapInfo {
	return LiftFromRustBuffer[[]ReverseSwapInfo](c, rb)
}

func (c FfiConverterSequenceTypeReverseSwapInfo) Read(reader io.Reader) []ReverseSwapInfo {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]ReverseSwapInfo, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeReverseSwapInfoINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeReverseSwapInfo) Lower(value []ReverseSwapInfo) RustBuffer {
	return LowerIntoRustBuffer[[]ReverseSwapInfo](c, value)
}

func (c FfiConverterSequenceTypeReverseSwapInfo) Write(writer io.Writer, value []ReverseSwapInfo) {
	if len(value) > math.MaxInt32 {
		panic("[]ReverseSwapInfo is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeReverseSwapInfoINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeReverseSwapInfo struct{}

func (FfiDestroyerSequenceTypeReverseSwapInfo) Destroy(sequence []ReverseSwapInfo) {
	for _, value := range sequence {
		FfiDestroyerTypeReverseSwapInfo{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeRouteHint struct{}

var FfiConverterSequenceTypeRouteHintINSTANCE = FfiConverterSequenceTypeRouteHint{}

func (c FfiConverterSequenceTypeRouteHint) Lift(rb RustBufferI) []RouteHint {
	return LiftFromRustBuffer[[]RouteHint](c, rb)
}

func (c FfiConverterSequenceTypeRouteHint) Read(reader io.Reader) []RouteHint {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]RouteHint, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeRouteHintINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeRouteHint) Lower(value []RouteHint) RustBuffer {
	return LowerIntoRustBuffer[[]RouteHint](c, value)
}

func (c FfiConverterSequenceTypeRouteHint) Write(writer io.Writer, value []RouteHint) {
	if len(value) > math.MaxInt32 {
		panic("[]RouteHint is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeRouteHintINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeRouteHint struct{}

func (FfiDestroyerSequenceTypeRouteHint) Destroy(sequence []RouteHint) {
	for _, value := range sequence {
		FfiDestroyerTypeRouteHint{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeRouteHintHop struct{}

var FfiConverterSequenceTypeRouteHintHopINSTANCE = FfiConverterSequenceTypeRouteHintHop{}

func (c FfiConverterSequenceTypeRouteHintHop) Lift(rb RustBufferI) []RouteHintHop {
	return LiftFromRustBuffer[[]RouteHintHop](c, rb)
}

func (c FfiConverterSequenceTypeRouteHintHop) Read(reader io.Reader) []RouteHintHop {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]RouteHintHop, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeRouteHintHopINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeRouteHintHop) Lower(value []RouteHintHop) RustBuffer {
	return LowerIntoRustBuffer[[]RouteHintHop](c, value)
}

func (c FfiConverterSequenceTypeRouteHintHop) Write(writer io.Writer, value []RouteHintHop) {
	if len(value) > math.MaxInt32 {
		panic("[]RouteHintHop is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeRouteHintHopINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeRouteHintHop struct{}

func (FfiDestroyerSequenceTypeRouteHintHop) Destroy(sequence []RouteHintHop) {
	for _, value := range sequence {
		FfiDestroyerTypeRouteHintHop{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeSwapInfo struct{}

var FfiConverterSequenceTypeSwapInfoINSTANCE = FfiConverterSequenceTypeSwapInfo{}

func (c FfiConverterSequenceTypeSwapInfo) Lift(rb RustBufferI) []SwapInfo {
	return LiftFromRustBuffer[[]SwapInfo](c, rb)
}

func (c FfiConverterSequenceTypeSwapInfo) Read(reader io.Reader) []SwapInfo {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]SwapInfo, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeSwapInfoINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeSwapInfo) Lower(value []SwapInfo) RustBuffer {
	return LowerIntoRustBuffer[[]SwapInfo](c, value)
}

func (c FfiConverterSequenceTypeSwapInfo) Write(writer io.Writer, value []SwapInfo) {
	if len(value) > math.MaxInt32 {
		panic("[]SwapInfo is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeSwapInfoINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeSwapInfo struct{}

func (FfiDestroyerSequenceTypeSwapInfo) Destroy(sequence []SwapInfo) {
	for _, value := range sequence {
		FfiDestroyerTypeSwapInfo{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeTlvEntry struct{}

var FfiConverterSequenceTypeTlvEntryINSTANCE = FfiConverterSequenceTypeTlvEntry{}

func (c FfiConverterSequenceTypeTlvEntry) Lift(rb RustBufferI) []TlvEntry {
	return LiftFromRustBuffer[[]TlvEntry](c, rb)
}

func (c FfiConverterSequenceTypeTlvEntry) Read(reader io.Reader) []TlvEntry {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]TlvEntry, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeTlvEntryINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeTlvEntry) Lower(value []TlvEntry) RustBuffer {
	return LowerIntoRustBuffer[[]TlvEntry](c, value)
}

func (c FfiConverterSequenceTypeTlvEntry) Write(writer io.Writer, value []TlvEntry) {
	if len(value) > math.MaxInt32 {
		panic("[]TlvEntry is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeTlvEntryINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeTlvEntry struct{}

func (FfiDestroyerSequenceTypeTlvEntry) Destroy(sequence []TlvEntry) {
	for _, value := range sequence {
		FfiDestroyerTypeTlvEntry{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeUnspentTransactionOutput struct{}

var FfiConverterSequenceTypeUnspentTransactionOutputINSTANCE = FfiConverterSequenceTypeUnspentTransactionOutput{}

func (c FfiConverterSequenceTypeUnspentTransactionOutput) Lift(rb RustBufferI) []UnspentTransactionOutput {
	return LiftFromRustBuffer[[]UnspentTransactionOutput](c, rb)
}

func (c FfiConverterSequenceTypeUnspentTransactionOutput) Read(reader io.Reader) []UnspentTransactionOutput {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]UnspentTransactionOutput, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeUnspentTransactionOutputINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeUnspentTransactionOutput) Lower(value []UnspentTransactionOutput) RustBuffer {
	return LowerIntoRustBuffer[[]UnspentTransactionOutput](c, value)
}

func (c FfiConverterSequenceTypeUnspentTransactionOutput) Write(writer io.Writer, value []UnspentTransactionOutput) {
	if len(value) > math.MaxInt32 {
		panic("[]UnspentTransactionOutput is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeUnspentTransactionOutputINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeUnspentTransactionOutput struct{}

func (FfiDestroyerSequenceTypeUnspentTransactionOutput) Destroy(sequence []UnspentTransactionOutput) {
	for _, value := range sequence {
		FfiDestroyerTypeUnspentTransactionOutput{}.Destroy(value)
	}
}

type FfiConverterSequenceTypePaymentTypeFilter struct{}

var FfiConverterSequenceTypePaymentTypeFilterINSTANCE = FfiConverterSequenceTypePaymentTypeFilter{}

func (c FfiConverterSequenceTypePaymentTypeFilter) Lift(rb RustBufferI) []PaymentTypeFilter {
	return LiftFromRustBuffer[[]PaymentTypeFilter](c, rb)
}

func (c FfiConverterSequenceTypePaymentTypeFilter) Read(reader io.Reader) []PaymentTypeFilter {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]PaymentTypeFilter, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypePaymentTypeFilterINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypePaymentTypeFilter) Lower(value []PaymentTypeFilter) RustBuffer {
	return LowerIntoRustBuffer[[]PaymentTypeFilter](c, value)
}

func (c FfiConverterSequenceTypePaymentTypeFilter) Write(writer io.Writer, value []PaymentTypeFilter) {
	if len(value) > math.MaxInt32 {
		panic("[]PaymentTypeFilter is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypePaymentTypeFilterINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypePaymentTypeFilter struct{}

func (FfiDestroyerSequenceTypePaymentTypeFilter) Destroy(sequence []PaymentTypeFilter) {
	for _, value := range sequence {
		FfiDestroyerTypePaymentTypeFilter{}.Destroy(value)
	}
}

type FfiConverterSequenceTypeSwapStatus struct{}

var FfiConverterSequenceTypeSwapStatusINSTANCE = FfiConverterSequenceTypeSwapStatus{}

func (c FfiConverterSequenceTypeSwapStatus) Lift(rb RustBufferI) []SwapStatus {
	return LiftFromRustBuffer[[]SwapStatus](c, rb)
}

func (c FfiConverterSequenceTypeSwapStatus) Read(reader io.Reader) []SwapStatus {
	length := readInt32(reader)
	if length == 0 {
		return nil
	}
	result := make([]SwapStatus, 0, length)
	for i := int32(0); i < length; i++ {
		result = append(result, FfiConverterTypeSwapStatusINSTANCE.Read(reader))
	}
	return result
}

func (c FfiConverterSequenceTypeSwapStatus) Lower(value []SwapStatus) RustBuffer {
	return LowerIntoRustBuffer[[]SwapStatus](c, value)
}

func (c FfiConverterSequenceTypeSwapStatus) Write(writer io.Writer, value []SwapStatus) {
	if len(value) > math.MaxInt32 {
		panic("[]SwapStatus is too large to fit into Int32")
	}

	writeInt32(writer, int32(len(value)))
	for _, item := range value {
		FfiConverterTypeSwapStatusINSTANCE.Write(writer, item)
	}
}

type FfiDestroyerSequenceTypeSwapStatus struct{}

func (FfiDestroyerSequenceTypeSwapStatus) Destroy(sequence []SwapStatus) {
	for _, value := range sequence {
		FfiDestroyerTypeSwapStatus{}.Destroy(value)
	}
}

func Connect(req ConnectRequest, listener EventListener) (*BlockingBreezServices, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeConnectError{}, func(_uniffiStatus *C.RustCallStatus) unsafe.Pointer {
		return C.uniffi_breez_sdk_bindings_fn_func_connect(FfiConverterTypeConnectRequestINSTANCE.Lower(req), FfiConverterCallbackInterfaceEventListenerINSTANCE.Lower(listener), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue *BlockingBreezServices
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterBlockingBreezServicesINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func DefaultConfig(envType EnvironmentType, apiKey string, nodeConfig NodeConfig) Config {
	return FfiConverterTypeConfigINSTANCE.Lift(rustCall(func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_func_default_config(FfiConverterTypeEnvironmentTypeINSTANCE.Lower(envType), FfiConverterStringINSTANCE.Lower(apiKey), FfiConverterTypeNodeConfigINSTANCE.Lower(nodeConfig), _uniffiStatus)
	}))
}

func MnemonicToSeed(phrase string) ([]uint8, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_func_mnemonic_to_seed(FfiConverterStringINSTANCE.Lower(phrase), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue []uint8
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterSequenceUint8INSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func ParseInput(s string) (InputType, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_func_parse_input(FfiConverterStringINSTANCE.Lower(s), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue InputType
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeInputTypeINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func ParseInvoice(invoice string) (LnInvoice, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_func_parse_invoice(FfiConverterStringINSTANCE.Lower(invoice), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue LnInvoice
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeLNInvoiceINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func ServiceHealthCheck(apiKey string) (ServiceHealthCheckResponse, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_func_service_health_check(FfiConverterStringINSTANCE.Lower(apiKey), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue ServiceHealthCheckResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeServiceHealthCheckResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}

func SetLogStream(logStream LogStream) error {
	_, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) bool {
		C.uniffi_breez_sdk_bindings_fn_func_set_log_stream(FfiConverterCallbackInterfaceLogStreamINSTANCE.Lower(logStream), _uniffiStatus)
		return false
	})
	return _uniffiErr
}

func StaticBackup(req StaticBackupRequest) (StaticBackupResponse, error) {
	_uniffiRV, _uniffiErr := rustCallWithError(FfiConverterTypeSdkError{}, func(_uniffiStatus *C.RustCallStatus) RustBufferI {
		return C.uniffi_breez_sdk_bindings_fn_func_static_backup(FfiConverterTypeStaticBackupRequestINSTANCE.Lower(req), _uniffiStatus)
	})
	if _uniffiErr != nil {
		var _uniffiDefaultValue StaticBackupResponse
		return _uniffiDefaultValue, _uniffiErr
	} else {
		return FfiConverterTypeStaticBackupResponseINSTANCE.Lift(_uniffiRV), _uniffiErr
	}
}
