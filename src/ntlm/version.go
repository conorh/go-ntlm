package ntlm

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// See MS-MNLP - 2.2.2.10:
// The VERSION structure contains Windows version information that SHOULD be
// ignored. This structure is used for debugging purposes only and its value
// does not affect NTLM message processing. It is present in the NEGOTIATE_MESSAGE, 
// CHALLENGE_MESSAGE, and AUTHENTICATE_MESSAGE messages only if NTLMSSP_NEGOTIATE_VERSION 
// is negotiated.<28>
type VersionStruct struct {
	ProductMajorVersion uint8
	ProductMinorVersion uint8
	ProductBuild        uint16
	Reserved            []byte
	NTLMRevisionCurrent uint8
}

func ReadVersionStruct(structSource []byte) (*VersionStruct, error) {
	versionStruct := new(VersionStruct)

	versionStruct.ProductMajorVersion = uint8(structSource[0])
	versionStruct.ProductMinorVersion = uint8(structSource[1])
	versionStruct.ProductBuild = binary.LittleEndian.Uint16(structSource[2:4])
	versionStruct.Reserved = structSource[4:7]
	versionStruct.NTLMRevisionCurrent = uint8(structSource[7])

	return versionStruct, nil
}

func (v *VersionStruct) String() string {
	return fmt.Sprintf("%d.%d.%d Ntlm %d", v.ProductMajorVersion, v.ProductMinorVersion, v.ProductBuild, v.NTLMRevisionCurrent)
}

func (v *VersionStruct) Bytes() []byte {
	dest := make([]byte, 0, 8)
	buffer := bytes.NewBuffer(dest)

	binary.Write(buffer, binary.LittleEndian, v.ProductMajorVersion)
	binary.Write(buffer, binary.LittleEndian, v.ProductMinorVersion)
	binary.Write(buffer, binary.LittleEndian, v.ProductBuild)
	buffer.Write(make([]byte, 3))
	binary.Write(buffer, binary.LittleEndian, uint8(v.NTLMRevisionCurrent))

	return buffer.Bytes()
}
