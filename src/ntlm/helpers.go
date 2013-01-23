package ntlm

// Miscellaneous helpers for processing NTLM messages

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"unicode/utf16"
)

// Create a 0 initialized slice of bytes
func zeroBytes(length int) []byte {
	return make([]byte, length, length)
}

// Concatenate two byte slices into a new slice
func concat(ar ...[]byte) []byte {
	return bytes.Join(ar, nil)
}

func randomBytes(length int) []byte {
	randombytes := make([]byte, length)
	_, err := rand.Read(randombytes)
	if err != nil {
	} // TODO: What to do with err here
	return randombytes
}

// Zero pad the input byte slice to the given size
// bytes  input byte slice
// offset  where to start taking the bytes from the input slice
// size  size of the output byte slize
func zeroPaddedBytes(bytes []byte, offset int, size int) []byte {
	newSlice := zeroBytes(size)
	for i := 0; i < size && i+offset < len(bytes); i++ {
		newSlice[i] = bytes[i+offset]
	}
	return newSlice
}

func utf16FromString(s string) []byte {
	encoded := utf16.Encode([]rune(s))
	// TODO: I'm sure there is an easier way to do the conversion from utf16 to bytes
	result := zeroBytes(len(encoded) * 2)
	for i := 0; i < len(encoded); i++ {
		result[i*2] = byte(encoded[i])
		result[i*2+1] = byte(encoded[i] << 8)
	}
	return result
}

// Convert a UTF16 string to UTF8 string for Go usage
func Utf16ToString(bytes []byte) string {
	var data []uint16

	// NOTE: This is definitely not the best way to do this, but when I tried using a buffer.Read I could not get it to work
	for offset := 0; offset < len(bytes); offset = offset + 2 {
		i := binary.LittleEndian.Uint16(bytes[offset : offset+2])
		data = append(data, i)
	}

	return string(utf16.Decode(data))
}

func StringToUtf16(value string) []byte {
	result := make([]byte, len(value)*2)
	stringBytes := []byte(value)
	for i := 0; i < len(value); i++ {
		result[i*2] = stringBytes[i]
	}
	return result
}

func Uint32ToBytes(v uint32) []byte {
	bytes := make([]byte, 4)
	bytes[0] = byte(v & 0xff)
	bytes[1] = byte((v >> 8) & 0xff)
	bytes[2] = byte((v >> 16) & 0xff)
	bytes[3] = byte((v >> 24) & 0xff)
	return bytes
}
