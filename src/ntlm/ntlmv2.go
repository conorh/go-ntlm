// Receve an Authenticate message and authenticate the user
package ntlm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"time"
)

/*******************************
 Shared Session Data and Methods
*******************************/

type V2Session struct {
	SessionData
}

func (n *V2Session) SetUserInfo(username string, password string, domain string) {
	n.user = username
	n.password = password
	n.userDomain = domain
}

func (n *V2Session) SetMode(mode Mode) {
	n.mode = mode
}

func (n *V2Session) fetchResponseKeys() (err error) {
	n.responseKeyLM = lmowfv2(n.user, n.password, n.userDomain)
	n.responseKeyNT = ntowfv2(n.user, n.password, n.userDomain)
	return
}

// Define ComputeResponse(NegFlg, ResponseKeyNT, ResponseKeyLM, CHALLENGE_MESSAGE.ServerChallenge, ClientChallenge, Time, ServerName)
// ServerNameBytes - The NtChallengeResponseFields.NTLMv2_RESPONSE.NTLMv2_CLIENT_CHALLENGE.AvPairs field structure of the AUTHENTICATE_MESSAGE payload.
func (n *V2Session) computeExpectedResponses(timestamp []byte, avPairBytes []byte) (err error) {
	temp := concat([]byte{0x01}, []byte{0x01}, zeroBytes(6), timestamp, n.clientChallenge, zeroBytes(4), avPairBytes, zeroBytes(4))
	ntProofStr := hmacMd5(n.responseKeyNT, concat(n.serverChallenge, temp))
	n.ntChallengeResponse = concat(ntProofStr, temp)
	n.lmChallengeResponse = concat(hmacMd5(n.responseKeyLM, concat(n.serverChallenge, n.clientChallenge)), n.clientChallenge)
	n.sessionBaseKey = hmacMd5(n.responseKeyNT, ntProofStr)
	return
}

func (n *V2Session) computeKeyExchangeKey() (err error) {
	n.keyExchangeKey = n.sessionBaseKey
	return
}

func (n *V2Session) calculateKeys(ntlmRevisionCurrent uint8) (err error) {
	// This lovely piece of code comes courtesy of an the excellent Open Document support system from MSFT
	// In order to calculate the keys correctly when the client has set the NTLMRevisionCurrent to 0xF (15)
	// We must treat the flags as if NTLMSSP_NEGOTIATE_LM_KEY is set.
	// This information is not contained (at least currently, until they correct it) in the MS-NLMP document
	if ntlmRevisionCurrent == 15 {
		n.negotiateFlags = NTLMSSP_NEGOTIATE_LM_KEY.Set(n.negotiateFlags)
	}

	n.clientSigningKey = signKey(n.negotiateFlags, n.exportedSessionKey, "Client")
	n.serverSigningKey = signKey(n.negotiateFlags, n.exportedSessionKey, "Server")
	n.clientSealingKey = sealKey(n.negotiateFlags, n.exportedSessionKey, "Client")
	n.serverSealingKey = sealKey(n.negotiateFlags, n.exportedSessionKey, "Server")
	return
}

func (n *V2Session) Seal(message []byte) ([]byte, error) {
	return nil, nil
}
func (n *V2Session) Sign(message []byte) ([]byte, error) {
	return nil, nil
}

func (n *V2Session) Mac(message []byte, sequenceNumber int) ([]byte, error) {
	// TODO: Need to keep track of the sequence number for connection oriented NTLM
	if NTLMSSP_NEGOTIATE_DATAGRAM.IsSet(n.negotiateFlags) && NTLMSSP_NEGOTIATE_EXTENDED_SESSIONSECURITY.IsSet(n.negotiateFlags) {
		n.serverHandle, _ = reinitSealingKey(n.serverSealingKey, sequenceNumber)
	} else if NTLMSSP_NEGOTIATE_DATAGRAM.IsSet(n.negotiateFlags) {
		// CONOR: Reinitializing the rc4 cipher on every requst, but not using the 
		// algorithm as described in the MS-NTLM document. Just reinitialize it directly.
		n.serverHandle, _ = rc4Init(n.serverSealingKey)
	}
	sig := mac(n.negotiateFlags, n.serverHandle, n.serverSigningKey, uint32(sequenceNumber), message)
	return sig.Bytes(), nil
}

/**************
 Server Session
**************/

type V2ServerSession struct {
	V2Session
}

func (n *V2ServerSession) SetServerChallenge(challenge []byte) {
	n.serverChallenge = challenge
}

func (n *V2ServerSession) ProcessNegotiateMessage(nm *NegotiateMessage) (err error) {
	n.negotiateMessage = nm
	return
}

func (n *V2ServerSession) GenerateChallengeMessage() (cm *ChallengeMessage, err error) {
	cm = new(ChallengeMessage)
	cm.Signature = []byte("NTLMSSP\x00")
	cm.MessageType = uint32(2)
	cm.TargetName, _ = CreateBytePayload(make([]byte, 0))

	flags := uint32(0)
	flags = NTLMSSP_NEGOTIATE_KEY_EXCH.Set(flags)
	flags = NTLMSSP_NEGOTIATE_VERSION.Set(flags)
	flags = NTLMSSP_NEGOTIATE_EXTENDED_SESSIONSECURITY.Set(flags)
	flags = NTLMSSP_NEGOTIATE_TARGET_INFO.Set(flags)
	flags = NTLMSSP_NEGOTIATE_IDENTIFY.Set(flags)
	flags = NTLMSSP_NEGOTIATE_ALWAYS_SIGN.Set(flags)
	flags = NTLMSSP_NEGOTIATE_NTLM.Set(flags)
	flags = NTLMSSP_NEGOTIATE_DATAGRAM.Set(flags)
	flags = NTLMSSP_NEGOTIATE_SIGN.Set(flags)
	flags = NTLMSSP_REQUEST_TARGET.Set(flags)
	flags = NTLMSSP_NEGOTIATE_UNICODE.Set(flags)
	flags = NTLMSSP_NEGOTIATE_128.Set(flags)

	cm.NegotiateFlags = flags

	n.serverChallenge = randomBytes(8)
	cm.ServerChallenge = n.serverChallenge
	cm.Reserved = make([]byte, 8)

	// Create the AvPairs we need
	pairs := new(AvPairs)
	pairs.AddAvPair(MsvAvNbDomainName, StringToUtf16("REUTERS"))
	pairs.AddAvPair(MsvAvNbComputerName, StringToUtf16("UKBP-CBTRMFE06"))
	pairs.AddAvPair(MsvAvDnsDomainName, StringToUtf16("Reuters.net"))
	pairs.AddAvPair(MsvAvDnsComputerName, StringToUtf16("ukbp-cbtrmfe06.Reuters.net"))
	pairs.AddAvPair(MsvAvDnsTreeName, StringToUtf16("Reuters.net"))
	pairs.AddAvPair(MsvAvEOL, make([]byte, 0))
	cm.TargetInfo = pairs
	cm.TargetInfoPayloadStruct, _ = CreateBytePayload(pairs.Bytes())

	cm.Version = &VersionStruct{ProductMajorVersion: uint8(5), ProductMinorVersion: uint8(1), ProductBuild: uint16(2600), NTLMRevisionCurrent: uint8(15)}
	return cm, nil
}

func (n *V2ServerSession) ProcessAuthenticateMessage(am *AuthenticateMessage) (err error) {
	n.authenticateMessage = am
	n.negotiateFlags = am.NegotiateFlags
	n.clientChallenge = am.ClientChallenge()
	n.encryptedRandomSessionKey = am.EncryptedRandomSessionKey.Payload

	err = n.fetchResponseKeys()
	if err != nil {
		return err
	}

	timestamp := am.NtlmV2Response.NtlmV2ClientChallenge.TimeStamp
	avPairsBytes := am.NtlmV2Response.NtlmV2ClientChallenge.AvPairs.Bytes()

	err = n.computeExpectedResponses(timestamp, avPairsBytes)
	if err != nil {
		return err
	}

	err = n.computeKeyExchangeKey()
	if err != nil {
		return err
	}

	if !bytes.Equal(am.NtChallengeResponseFields.Payload, n.ntChallengeResponse) {
		if !bytes.Equal(am.LmChallengeResponse.Payload, n.lmChallengeResponse) {
			return errors.New("Could not authenticate")
		}
	}

	n.mic = am.Mic
	am.Mic = zeroBytes(16)

	err = n.computeExportedSessionKey()
	if err != nil {
		return err
	}

	err = n.calculateKeys(am.Version.NTLMRevisionCurrent)
	if err != nil {
		return err
	}

	n.clientHandle, err = rc4Init(n.clientSealingKey)
	if err != nil {
		return err
	}
	n.serverHandle, err = rc4Init(n.serverSealingKey)
	if err != nil {
		return err
	}

	return nil
}

func (n *V2ServerSession) computeExportedSessionKey() (err error) {
	if NTLMSSP_NEGOTIATE_KEY_EXCH.IsSet(n.negotiateFlags) {
		n.exportedSessionKey, err = rc4K(n.keyExchangeKey, n.encryptedRandomSessionKey)
		if err != nil {
			return err
		}
		// TODO: Calculate mic correctly. This calculation is not producing the right results now
		// n.calculatedMic = HmacMd5(n.exportedSessionKey, concat(n.challengeMessage.Payload, n.authenticateMessage.Bytes))
	} else {
		n.exportedSessionKey = n.keyExchangeKey
		// TODO: Calculate mic correctly. This calculation is not producing the right results now
		// n.calculatedMic = HmacMd5(n.keyExchangeKey, concat(n.challengeMessage.Payload, n.authenticateMessage.Bytes))
	}
	return nil
}

/*************
 Client Session
**************/

type V2ClientSession struct {
	V2Session
}

func (n *V2ClientSession) GenerateNegotiateMessage() (nm *NegotiateMessage, err error) {
	return nil, nil
}

func (n *V2ClientSession) ProcessChallengeMessage(cm *ChallengeMessage) (err error) {
	n.challengeMessage = cm
	n.serverChallenge = cm.ServerChallenge
	n.clientChallenge = randomBytes(8)

	// Set up the default flags for processing the response. These are the flags that we will return
	// in the authenticate message
	flags := uint32(0)
	flags = NTLMSSP_NEGOTIATE_KEY_EXCH.Set(flags)
	flags = NTLMSSP_NEGOTIATE_VERSION.Set(flags)
	flags = NTLMSSP_NEGOTIATE_EXTENDED_SESSIONSECURITY.Set(flags)
	flags = NTLMSSP_NEGOTIATE_TARGET_INFO.Set(flags)
	flags = NTLMSSP_NEGOTIATE_IDENTIFY.Set(flags)
	flags = NTLMSSP_NEGOTIATE_ALWAYS_SIGN.Set(flags)
	flags = NTLMSSP_NEGOTIATE_NTLM.Set(flags)
	flags = NTLMSSP_NEGOTIATE_DATAGRAM.Set(flags)
	flags = NTLMSSP_NEGOTIATE_SIGN.Set(flags)
	flags = NTLMSSP_REQUEST_TARGET.Set(flags)
	flags = NTLMSSP_NEGOTIATE_UNICODE.Set(flags)
	flags = NTLMSSP_NEGOTIATE_128.Set(flags)

	n.negotiateFlags = flags

	err = n.fetchResponseKeys()
	if err != nil {
		return err
	}

	timestamp := timeToWindowsFileTime(time.Now())
	err = n.computeExpectedResponses(timestamp, cm.TargetInfoPayloadStruct.Payload)
	if err != nil {
		return err
	}

	err = n.computeKeyExchangeKey()
	if err != nil {
		return err
	}

	err = n.computeEncryptedSessionKey()
	if err != nil {
		return err
	}

	err = n.calculateKeys(cm.Version.NTLMRevisionCurrent)
	if err != nil {
		return err
	}

	n.clientHandle, err = rc4Init(n.clientSealingKey)
	if err != nil {
		return err
	}
	n.serverHandle, err = rc4Init(n.serverSealingKey)
	if err != nil {
		return err
	}
	return nil
}

func (n *V2ClientSession) GenerateAuthenticateMessage() (am *AuthenticateMessage, err error) {
	am = new(AuthenticateMessage)
	am.Signature = []byte("NTLMSSP\x00")
	am.MessageType = uint32(3)
	am.LmChallengeResponse, _ = CreateBytePayload(n.lmChallengeResponse)
	am.NtChallengeResponseFields, _ = CreateBytePayload(n.ntChallengeResponse)
	am.DomainName, _ = CreateStringPayload(n.userDomain)
	am.UserName, _ = CreateStringPayload(n.user)
	am.Workstation, _ = CreateStringPayload("SQUAREMILL")
	am.EncryptedRandomSessionKey, _ = CreateBytePayload(n.encryptedRandomSessionKey)
	am.NegotiateFlags = n.negotiateFlags
	am.Mic = make([]byte, 16)
	am.Version = &VersionStruct{ProductMajorVersion: uint8(5), ProductMinorVersion: uint8(1), ProductBuild: uint16(2600), NTLMRevisionCurrent: 0x0F}
	return am, nil
}

func (n *V2ClientSession) computeEncryptedSessionKey() (err error) {
	if NTLMSSP_NEGOTIATE_KEY_EXCH.IsSet(n.negotiateFlags) {
		n.exportedSessionKey = randomBytes(16)
		n.encryptedRandomSessionKey, err = rc4K(n.keyExchangeKey, n.exportedSessionKey)
		if err != nil {
			return err
		}
	} else {
		n.encryptedRandomSessionKey = n.keyExchangeKey
	}
	return nil
}

/********************************
 NTLM V2 Password hash functions
*********************************/

// Define ntowfv2(Passwd, User, UserDom) as
func ntowfv2(user string, passwd string, userDom string) []byte {
	concat := utf16FromString(strings.ToUpper(user) + userDom)
	return hmacMd5(md4(utf16FromString(passwd)), concat)
}

// Define lmowfv2(Passwd, User, UserDom) as
func lmowfv2(user string, passwd string, userDom string) []byte {
	return ntowfv2(user, passwd, userDom)
}

/********************************
 Helper functions
*********************************/

func timeToWindowsFileTime(t time.Time) []byte {
	var ll int64
	ll = (int64(t.Unix()) * int64(10000000)) + int64(116444736000000000)
	buffer := bytes.NewBuffer(make([]byte, 0, 8))
	binary.Write(buffer, binary.LittleEndian, ll)
	return buffer.Bytes()
}
