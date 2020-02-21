/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

process.env.CHROME_BIN = require('puppeteer').executablePath()

module.exports = function(config) {
    config.set({
        frameworks: ["mocha", "chai", "cucumber-js"],
        browsers: ["ChromeHeadless"],
        singleRun: true,
        files: [
            {pattern: "test/**/*.js", type: "module"},
            {pattern: "public/aries-framework-go/assets/*", included: false},
            {pattern: "node_modules/@hyperledger/aries-framework-go/dist/web/*", type: "module"},
            {pattern: "features/*.feature", included: false},
        ],
        plugins: [
            "karma-mocha",
            "karma-chai",
            "karma-chrome-launcher",
            "karma-cucumber-js-latest"
        ]
    })
}