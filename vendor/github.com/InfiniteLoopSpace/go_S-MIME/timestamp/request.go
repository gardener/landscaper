package timestamp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"crypto"
	"crypto/rand"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"

	cms "github.com/InfiniteLoopSpace/go_S-MIME/cms/protocol"
	oid "github.com/InfiniteLoopSpace/go_S-MIME/oid"
)

// TimeStampReq ::= SEQUENCE  {
//	version                  INTEGER  { v1(1) },
//	messageImprint           MessageImprint,
//	  --a hash algorithm OID and the hash value of the data to be
//	  --time-stamped
//	reqPolicy                TSAPolicyId                OPTIONAL,
//	nonce                    INTEGER                    OPTIONAL,
//	certReq                  BOOLEAN                    DEFAULT FALSE,
//	extensions               [0] IMPLICIT Extensions    OPTIONAL  }
type TimeStampReq struct {
	Version        int
	MessageImprint MessageImprint
	ReqPolicy      asn1.ObjectIdentifier `asn1:"optional"`
	Nonce          *big.Int              `asn1:"optional"`
	CertReq        bool                  `asn1:"optional,default:false"`
	Extensions     []pkix.Extension      `asn1:"tag:1,optional"`
}

func newTSRequest(msg []byte, hash crypto.Hash) (TimeStampReq, error) {

	mi, err := NewMessageImprint(hash, msg)
	if err != nil {
		return TimeStampReq{}, err
	}

	return TimeStampReq{
		Version:        1,
		CertReq:        true,
		Nonce:          GenerateNonce(),
		MessageImprint: mi,
	}, nil
}

// GenerateNonce generates a new nonce for this TSR.
func GenerateNonce() *big.Int {
	buf := make([]byte, nonceBytes)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}

	return new(big.Int).SetBytes(buf[:])
}

// Do sends this timestamp request to the specified timestamp service, returning
// the parsed response.
func (req TimeStampReq) Do(url string) (TimeStampResp, error) {
	var nilResp TimeStampResp

	reqDER, err := asn1.Marshal(req)
	if err != nil {
		return nilResp, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqDER))
	if err != nil {
		return nilResp, err
	}
	httpReq.Header.Add("Content-Type", contentTypeTSQuery)

	HTTP := http.DefaultClient

	httpResp, err := HTTP.Do(httpReq)
	if err != nil {
		return nilResp, err
	}
	if ct := httpResp.Header.Get("Content-Type"); ct != contentTypeTSReply {
		return nilResp, fmt.Errorf("Bad content-type: %s", ct)
	}

	buf := bytes.NewBuffer(make([]byte, 0, httpResp.ContentLength))
	if _, err = io.Copy(buf, httpResp.Body); err != nil {
		return nilResp, err
	}

	return ParseResponse(buf.Bytes())
}

//MessageImprint ::= SEQUENCE  {
//	hashAlgorithm                AlgorithmIdentifier,
//	hashedMessage                OCTET STRING  }
type MessageImprint struct {
	HashAlgorithm pkix.AlgorithmIdentifier
	HashedMessage []byte
}

// NewMessageImprint creates a new MessageImprint, digesting msg using the specified hash.
func NewMessageImprint(hash crypto.Hash, msg []byte) (MessageImprint, error) {
	digestAlgorithm := oid.HashToDigestAlgorithm[hash]
	if len(digestAlgorithm) == 0 {
		return MessageImprint{}, cms.ErrUnsupported
	}

	if !hash.Available() {
		return MessageImprint{}, cms.ErrUnsupported
	}
	h := hash.New()
	if _, err := h.Write(msg); err != nil {
		return MessageImprint{}, err
	}

	return MessageImprint{
		HashAlgorithm: pkix.AlgorithmIdentifier{Algorithm: digestAlgorithm},
		HashedMessage: h.Sum(nil),
	}, nil
}
