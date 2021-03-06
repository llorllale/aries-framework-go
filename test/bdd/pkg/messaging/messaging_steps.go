/*
 *
 * Copyright SecureKey Technologies Inc. All Rights Reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 * /
 *
 */

package messaging

import (
	"fmt"

	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/service"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/dispatcher"
	"github.com/hyperledger/aries-framework-go/pkg/store/connection"
	"github.com/hyperledger/aries-framework-go/test/bdd/pkg/context"
)

// messagingSDKSteps is steps for messaging using client SDK
type messagingSDKSteps struct {
	bddContext *context.BDDContext
}

// newMessagingSDKSteps return new steps for messaging using client SDK
func newMessagingSDKSteps(ctx *context.BDDContext) *messagingSDKSteps {
	return &messagingSDKSteps{
		bddContext: ctx,
	}
}

func (d *messagingSDKSteps) registerMsgService(agentID string, msgSvc dispatcher.MessageService) error {
	registrar, ok := d.bddContext.MessageRegistrar[agentID]
	if !ok {
		return fmt.Errorf("unable to find message registrar for agent `%s`", agentID)
	}

	err := registrar.Register(msgSvc)
	if err != nil {
		return fmt.Errorf("unable to register message service '%s' : %w",
			msgSvc.Name(), err)
	}

	logger.Debugf("Agent[%s] registered message service '%s'",
		agentID, msgSvc.Name())

	return nil
}

func (d *messagingSDKSteps) sendMessage(fromAgentID, toAgentID string, msg service.DIDCommMsgMap) error {
	messenger, ok := d.bddContext.Messengers[fromAgentID]
	if !ok {
		return fmt.Errorf("unable to find messenger for agent `%s`", fromAgentID)
	}

	ctx, ok := d.bddContext.AgentCtx[fromAgentID]
	if !ok {
		return fmt.Errorf("unable to find context for agent `%s`", fromAgentID)
	}

	// find connection matching destination
	lookup, err := connection.NewLookup(ctx)
	if err != nil {
		return fmt.Errorf("failed to get connection lookup")
	}

	connections, err := lookup.QueryConnectionRecords()
	if err != nil {
		return fmt.Errorf("failed to query connections")
	}

	var target *connection.Record

	for _, conn := range connections {
		if conn.State == "completed" && conn.TheirLabel == toAgentID {
			target = conn
			break
		}
	}

	// send message
	err = messenger.Send(msg, target.MyDID, target.TheirDID)
	if err != nil {
		return fmt.Errorf("failed to send message to agent[%s] : %w", toAgentID, err)
	}

	return nil
}
