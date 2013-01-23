package ntlm

// See MS-NLMP - 2.2.1.1 NEGOTIATE_MESSAGE
// The NEGOTIATE_MESSAGE defines an NTLM Negotiate message that is sent from the
// client to the server. This message allows the client to specify its supported
// NTLM options to the server.

type NegotiateMessage struct {
	// All bytes of the message
	Bytes []byte

	// sig - 8 bytes
	Signature []byte
	// message type - 4 bytes
	MessageType uint32
	// negotiate flags - 4bytes
	NegotiateFlags uint32
	// If the NTLMSSP_NEGOTIATE_OEM_DOMAIN_SUPPLIED flag is not set in NegotiateFlags,
	// indicating that no DomainName is supplied in Payload  - then this should have Len 0 / MaxLen 0 
	// this contains a domain name
	DomainNameFields *PayloadStruct
	// If the NTLMSSP_NEGOTIATE_OEM_WORKSTATION_SUPPLIED flag is not set in NegotiateFlags, 
	// indicating that no WorkstationName is supplied in Payload - then this should have Len 0 / MaxLen 0 
	WorkstationFields *PayloadStruct
	// version - 8 bytes
	Version *VersionStruct
	// payload - variable
	Payload       []byte
	PayloadOffset int
}
