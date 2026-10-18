// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import Foundation
import Testing

@testable import WOPR

/// The app's security properties (ios/README.md, "Security"): it takes no permissions, has no
/// network exceptions, collects nothing, and removes anything but printable ASCII from input.
/// ios/scripts/check-app.sh checks the same of the release build.
@Suite("Security")
struct SecurityTests {
    private var info: [String: Any] { Bundle.main.infoDictionary ?? [:] }

    @Test("The app is the one under test")
    func hostApp() {
        #expect(Bundle.main.bundleIdentifier == "io.github.ghostofgoes.wopr")
    }

    @Test("App Transport Security has no exceptions")
    func noATSExceptions() {
        #expect(info["NSAppTransportSecurity"] == nil)
    }

    @Test("The app asks for no permission and opens no URL")
    func noPermissions() {
        let usage = info.keys.filter { $0.hasSuffix("UsageDescription") }
        #expect(usage.isEmpty, "\(usage)")
        #expect(info["CFBundleURLTypes"] == nil)
        #expect(info["LSApplicationQueriesSchemes"] == nil)
        #expect(info["UIBackgroundModes"] == nil)
        #expect(info["UIFileSharingEnabled"] == nil)
        #expect(info["LSSupportsOpeningDocumentsInPlace"] == nil)
    }

    @Test("The app needs iOS 26 or later and uses no export-controlled encryption")
    func minimumOS() {
        #expect(info["MinimumOSVersion"] as? String == "26.0")
        #expect(info["ITSAppUsesNonExemptEncryption"] as? Bool == false)
    }

    @Test("The privacy manifest declares no tracking, no data and no reasoned APIs")
    func privacyManifest() throws {
        let url = try #require(Bundle.main.url(forResource: "PrivacyInfo", withExtension: "xcprivacy"))
        let data = try Data(contentsOf: url)
        let plist = try #require(
            try PropertyListSerialization.propertyList(from: data, format: nil) as? [String: Any])
        #expect(plist["NSPrivacyTracking"] as? Bool == false)
        for key in ["NSPrivacyTrackingDomains", "NSPrivacyCollectedDataTypes", "NSPrivacyAccessedAPITypes"] {
            let list = try #require(plist[key] as? [Any], "\(key)")
            #expect(list.isEmpty, "\(key)")
        }
    }

    @Test(
        "Input loses escape sequences, control characters and other scripts",
        arguments: [
            ("JOSHUA\u{1B}[2J", "JOSHUA[2J"),
            ("\u{1B}]0;title\u{07}hi", "]0;titlehi"),
            ("a\tb\nc\r\u{7F}d\u{0}", "abcd"),
            ("NA\u{EF}VE \u{1F600}", "NAVE "),
            ("\u{202E}evil", "evil"),
            ("PLAIN TEXT 123 !?", "PLAIN TEXT 123 !?"),
        ])
    func cleansInput(input: String, want: String) {
        #expect(InputFilter.clean(input) == want)
    }

    @Test("Input is cut to the longest line the session reads")
    func capsInput() {
        let long = String(repeating: "A", count: 100_000)
        #expect(InputFilter.clean(long).count == InputFilter.maxLength)
        var session = Session()
        session.enter(long)
        #expect(session.screen.allSatisfy { $0.count <= 80 })
    }

    @Test("The text field never holds what the session would remove")
    @MainActor
    func modelFiltersDraft() {
        let model = TerminalModel()
        model.draft = "JOSH\u{1B}UA\u{0}"
        #expect(model.draft == "JOSHUA")
        model.submit()
        #expect(model.draft.isEmpty)
        #expect(model.session.phase == .greeting(step: 0))
    }
}
