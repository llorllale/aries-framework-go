/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package verifiable

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	validCred1 = `
{
  "@context": [
    "https://www.w3.org/2018/credentials/v1",
    "%s"
  ],
  "id": "http://example.edu/credentials/1872",
  "type": [
    "VerifiableCredential",
    "CredType1"
  ],
  "credentialSubject": {
    "id": "did:example:ebfeb1f712ebc6f1c276e12ec21",
    "s1": "custom subject 1"
  },

  "c1": "custom field 1",

  "issuer": {
    "id": "did:example:76e12ec712ebc6f1c221ebfeb1f",
    "name": "Example University"
  },

  "issuanceDate": "2010-01-01T19:23:24Z"
}
`

	validCred2 = `
{
  "@context": [
    "https://www.w3.org/2018/credentials/v1",
    "%s"
  ],
  "id": "http://example.edu/credentials/1872",
  "type": [
    "VerifiableCredential",
    "CredType2"
  ],
  "credentialSubject": {
    "id": "did:example:ebfeb1f712ebc6f1c276e12ec21",
    "s2": "custom subject 2"
  },

  "c2": "custom field 2",

  "issuer": {
    "id": "did:example:76e12ec712ebc6f1c221ebfeb1f",
    "name": "Example University"
  },

  "issuanceDate": "2010-01-01T19:23:24Z"
}`

	credMissingMandatoryFields = `
{
  "@context": [
    "https://www.w3.org/2018/credentials/v1",
    "https://www.w3.org/2018/credentials/examples/ext/type2"
  ],
  "id": "http://example.edu/credentials/1872",
  "type": [
    "VerifiableCredential"
  ]
}`
)

// Cred1 can produce itself.
type Cred1 struct {
	Base    Credential
	Subject struct {
		ID                 string `json:"id,omitempty"`
		CustomSubjectField string `json:"s1,omitempty"`
	} `json:"credentialSubject,omitempty"`
	CustomField string `json:"c1,omitempty"`
}

func (c1 *Cred1) Accept(vc *Credential) bool {
	return hasType(vc.Types, "CredType1")
}

func (c1 *Cred1) Apply(vc *Credential, dataJSON []byte) (interface{}, error) {
	err := json.Unmarshal(dataJSON, c1)
	if err != nil {
		return nil, err
	}

	c1.Base = *vc

	return c1, nil
}

// There is a separate producer for Cred2.
type Cred2 struct {
	Base    Credential
	Subject struct {
		ID                 string `json:"id,omitempty"`
		CustomSubjectField string `json:"s2,omitempty"`
	} `json:"credentialSubject,omitempty"`
	CustomField string `json:"c2,omitempty"`
}

type Cred2Producer struct {
}

func (c2p *Cred2Producer) Accept(vc *Credential) bool {
	return hasType(vc.Types, "CredType2")
}

func (c2p *Cred2Producer) Apply(vc *Credential, dataJSON []byte) (interface{}, error) {
	c2 := Cred2{}

	err := json.Unmarshal(dataJSON, &c2)
	if err != nil {
		return nil, err
	}

	c2.Base = *vc

	return &c2, nil
}

func NewCred1Producer() CustomCredentialProducer {
	return &Cred1{}
}

func NewCred2Producer() CustomCredentialProducer {
	return &Cred2Producer{}
}

type FailingCredentialProducer struct {
}

func (fp *FailingCredentialProducer) Accept(*Credential) bool {
	return true
}

func (fp *FailingCredentialProducer) Apply(*Credential, []byte) (interface{}, error) {
	return nil, errors.New("failed to apply credential extension")
}

func hasType(allTypes []string, targetType string) bool {
	for _, thatType := range allTypes {
		if thatType == targetType {
			return true
		}
	}

	return false
}

func TestCredentialExtensibilitySwitch(t *testing.T) {
	producers := []CustomCredentialProducer{NewCred1Producer(), NewCred2Producer()}

	jsonldContext1 := `
{
  "@context": {
    "c1": "https://example.com/vocab#c1",
    "s1": "https://example.com/vocab#s1"
  }
}
`

	jsonldContext2 := `
{
  "@context": {
    "c2": "https://example.com/vocab#c2",
    "s2": "https://example.com/vocab#s2"
  }
}
`

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Query()["context"][0] == "1" {
			res.WriteHeader(http.StatusOK)
			_, err := res.Write([]byte(jsonldContext1))
			require.NoError(t, err)
		} else {
			res.WriteHeader(http.StatusOK)
			_, err := res.Write([]byte(jsonldContext2))
			require.NoError(t, err)
		}
	}))

	defer func() { testServer.Close() }()

	// Producer1 applied.
	i1, err := CreateCustomCredential([]byte(fmt.Sprintf(validCred1, testServer.URL+"?context=1")), producers)
	require.NoError(t, err)
	require.IsType(t, &Cred1{}, i1)
	cred1, correct := i1.(*Cred1)
	require.True(t, correct)
	require.NotNil(t, cred1.Base)
	require.Equal(t, []string{"VerifiableCredential", "CredType1"}, cred1.Base.Types)
	require.Equal(t, "custom field 1", cred1.CustomField)
	require.Equal(t, "custom subject 1", cred1.Subject.CustomSubjectField)

	// Producer2 applied.
	i2, err := CreateCustomCredential([]byte(fmt.Sprintf(validCred2, testServer.URL+"?context=2")), producers)
	require.NoError(t, err)
	require.IsType(t, &Cred2{}, i2)
	cred2, correct := i2.(*Cred2)
	require.True(t, correct)
	require.NotNil(t, cred2.Base)
	require.Equal(t, []string{"VerifiableCredential", "CredType2"}, cred2.Base.Types)
	require.Equal(t, "custom field 2", cred2.CustomField)
	require.Equal(t, "custom subject 2", cred2.Subject.CustomSubjectField)

	// No producers are applied, returned base credential.
	i3, err := CreateCustomCredential([]byte(validCredential), producers)
	require.NoError(t, err)
	require.IsType(t, &Credential{}, i3)

	// Invalid credential.
	i4, err := CreateCustomCredential([]byte(credMissingMandatoryFields), producers)
	require.Error(t, err)
	require.Contains(t, err.Error(), "build base verifiable credential")
	require.Nil(t, i4)

	// Failing ext producer.
	i5, err := CreateCustomCredential([]byte(validCredential), []CustomCredentialProducer{&FailingCredentialProducer{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to apply credential extension")
	require.Nil(t, i5)
}
