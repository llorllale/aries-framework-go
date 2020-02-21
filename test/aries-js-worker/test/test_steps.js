/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

const Given = Cucumber.Given
const When = Cucumber.When
const Then = Cucumber.Then

class Context {
    constructor() {
        this.agents = new Map()
        this.results = new Map()
    }

    addAries(name, instance) {
        this.agents.set(name, instance)
    }

    aries(name) {
        return this.agents.get(name)
    }

    pushResult(name, result) {
        if (!this.results.has(name)) {
            this.results.set(name, [result])
            return
        }
        this.results.get(name).push(result)
    }

    popResult(name) {
        if (this.results.has(name)) {
            return this.results.get(name).pop()
        }
        return undefined
    }
}

const CONTEXT = new Context()

const agentName = "Alice"

Given(/^an instance of aries$/, function() {
    new Aries.Framework({
        assetsPath: "/base/public/aries-framework-go/assets",
        "agent-default-label": "dem-js-agent",
        "http-resolver-url": "",
        "auto-accept": true,
        "outbound-transport": ["ws", "http"],
        "transport-return-route": "all",
        "log-level": "debug"
    }).then(
        aries => CONTEXT.addAries(agentName, aries),
        err => {throw new Error(err.message)}
    )
})

When("I invoke _echo() with the input {string}", function(msg) {
    CONTEXT.aries(agentName)._test._echo(msg).then(
        result => CONTEXT.pushResult(agentName, result),
        err => {throw new Error(err.message)}
    )
})

Then("the result should equal {string}", function(expected) {
    assert.equal(CONTEXT.popResult(agentName), expected)
})
