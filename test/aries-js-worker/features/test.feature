#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

Feature: Basic JS-WASM integration
    In order to develop aries-js-worker
    As a developer
    I need to ensure basic JS <-> WASM integration

    Scenario: WASM returns input message
        Given an instance of aries
        When I invoke _echo() with the input "hello world"
        Then the result should equal "hello world"
