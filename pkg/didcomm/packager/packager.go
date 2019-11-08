/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package packager

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/btcsuite/btcutil/base58"

	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/transport"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/packer"
)

// Provider contains dependencies for the base packager and is typically created by using aries.Context()
type Provider interface {
	InboundPackers() []packer.Packer
	Packer() packer.Packer
}

// Creator method to create new packager service
type Creator func(prov Provider) (transport.Packager, error)

// Packager is the basic implementation of Packager
type Packager struct {
	packer         packer.Packer
	inboundPackers map[string]packer.Packer
}

// PackerCreator holds a creator function for a Packer and the name of the Packer's encoding method.
type PackerCreator struct {
	PackerName string
	Creator    packer.Creator
}

// New return new instance of KMS implementation
func New(ctx Provider) (*Packager, error) {
	basePackager := Packager{
		packer:         nil,
		inboundPackers: map[string]packer.Packer{},
	}

	for _, packerType := range ctx.InboundPackers() {
		basePackager.addPacker(packerType)
	}

	basePackager.packer = ctx.Packer()
	if basePackager.packer == nil {
		return nil, fmt.Errorf("need outbound packer to initialize packager")
	}

	basePackager.addPacker(basePackager.packer)

	return &basePackager, nil
}

func (bp *Packager) addPacker(pack packer.Packer) {
	if bp.inboundPackers[pack.EncodingType()] == nil {
		bp.inboundPackers[pack.EncodingType()] = pack
	}
}

// PackMessage Pack a message for one or more recipients.
func (bp *Packager) PackMessage(messageEnvelope *transport.Envelope) ([]byte, error) {
	if messageEnvelope == nil {
		return nil, errors.New("envelope argument is nil")
	}

	var recipients [][]byte

	for _, verKey := range messageEnvelope.ToVerKeys {
		// TODO It is possible to have different key schemes in an interop situation
		// there is no guarantee that each recipient is using the same key types
		// for now this package uses Ed25519 signing keys. Other key schemes should have their own
		// envelope implementations.
		// decode base58 ver key
		verKeyBytes := base58.Decode(verKey)
		// create 32 byte key
		recipients = append(recipients, verKeyBytes)
	}
	// pack message
	bytes, err := bp.packer.Pack(messageEnvelope.Message, base58.Decode(messageEnvelope.FromVerKey), recipients)
	if err != nil {
		return nil, fmt.Errorf("pack: %w", err)
	}

	return bytes, nil
}

type envelopeStub struct {
	Protected string `json:"protected,omitempty"`
}

type headerStub struct {
	Type string `json:"typ,omitempty"`
}

func getEncodingType(encMessage []byte) (string, error) {
	env := &envelopeStub{}

	err := json.Unmarshal(encMessage, env)
	if err != nil {
		return "", fmt.Errorf("parse envelope: %w", err)
	}

	var protBytes []byte

	protBytes1, err1 := base64.URLEncoding.DecodeString(env.Protected)
	protBytes2, err2 := base64.RawURLEncoding.DecodeString(env.Protected)

	switch {
	case err1 == nil:
		protBytes = protBytes1
	case err2 == nil:
		protBytes = protBytes2
	default:
		return "", fmt.Errorf("decode header: %w", err1)
	}

	prot := &headerStub{}

	err = json.Unmarshal(protBytes, prot)
	if err != nil {
		return "", fmt.Errorf("parse header: %w", err)
	}

	return prot.Type, nil
}

// UnpackMessage Unpack a message.
func (bp *Packager) UnpackMessage(encMessage []byte) (*transport.Envelope, error) {
	encType, err := getEncodingType(encMessage)
	if err != nil {
		return nil, fmt.Errorf("getEncodingType: %w", err)
	}

	p, ok := bp.inboundPackers[encType]
	if !ok {
		return nil, fmt.Errorf("message Type not recognized")
	}

	bytes, err := p.Unpack(encMessage)
	if err != nil {
		return nil, fmt.Errorf("unpack: %w", err)
	}
	// TODO extract fromVerKey and toVerKey from packer.Unpack() call above and set them here
	return &transport.Envelope{Message: bytes}, nil
}