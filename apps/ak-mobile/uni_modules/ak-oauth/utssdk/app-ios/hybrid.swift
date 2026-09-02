import AuthenticationServices
import Foundation
import UIKit

@objc(AkOAuthIOSBridge)
public final class AkOAuthIOSBridge: NSObject, ASAuthorizationControllerDelegate, ASAuthorizationControllerPresentationContextProviding {
    private static let shared = AkOAuthIOSBridge()
    private var success: ((String, String, String) -> Void)?
    private var failure: ((String) -> Void)?
    private var expectedState: String = ""

    @objc public static func authorizeApple(
        _ nonce: String,
        _ state: String,
        _ success: @escaping (String, String, String) -> Void,
        _ failure: @escaping (String) -> Void
    ) {
        guard !nonce.isEmpty, !state.isEmpty else { failure("configuration_missing"); return }
        let request = ASAuthorizationAppleIDProvider().createRequest()
        request.requestedScopes = [.fullName, .email]
        request.nonce = nonce
        request.state = state
        let bridge = AkOAuthIOSBridge.shared
        bridge.success = success
        bridge.failure = failure
        bridge.expectedState = state
        let controller = ASAuthorizationController(authorizationRequests: [request])
        controller.delegate = bridge
        controller.presentationContextProvider = bridge
        controller.performRequests()
    }

    public func presentationAnchor(for controller: ASAuthorizationController) -> ASPresentationAnchor {
        let scenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
        if let window = scenes.flatMap({ $0.windows }).first(where: { $0.isKeyWindow }) { return window }
        return ASPresentationAnchor()
    }

    public func authorizationController(controller: ASAuthorizationController, didCompleteWithAuthorization authorization: ASAuthorization) {
        guard let credential = authorization.credential as? ASAuthorizationAppleIDCredential,
              credential.state == expectedState,
              let token = credential.identityToken,
              let code = credential.authorizationCode,
              let value = String(data: token, encoding: .utf8),
              let authorizationCode = String(data: code, encoding: .utf8),
              !value.isEmpty, !authorizationCode.isEmpty else { completeFailure("native_failure"); return }
        let callback = success
        clear()
        callback?(value, authorizationCode, normalizedDisplayName(credential.fullName))
    }

    public func authorizationController(controller: ASAuthorizationController, didCompleteWithError error: Error) {
        let code: String
        if let authError = error as? ASAuthorizationError, authError.code == .canceled { code = "authorization_denied" }
        else { code = "native_failure" }
        completeFailure(code)
    }

    private func completeFailure(_ code: String) {
        let callback = failure
        clear()
        callback?(code)
    }

    private func normalizedDisplayName(_ components: PersonNameComponents?) -> String {
        guard let components else { return "" }
        let raw = PersonNameComponentsFormatter().string(from: components)
        if raw.unicodeScalars.contains(where: { CharacterSet.controlCharacters.contains($0) }) { return "" }
        let collapsed = raw.split(whereSeparator: { $0.isWhitespace }).joined(separator: " ")
        return String(collapsed.prefix(120))
    }

    private func clear() { success = nil; failure = nil; expectedState = "" }
}
