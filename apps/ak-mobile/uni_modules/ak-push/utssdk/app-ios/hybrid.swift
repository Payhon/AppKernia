import Foundation
import UIKit
import UserNotifications

@objc(AkPushIOSBridge)
public final class AkPushIOSBridge: NSObject {
    private static var tokenCallback: ((String) -> Void)?
    private static var failureCallback: ((String) -> Void)?
    private static var observer: NSObjectProtocol?

    @objc public static func authorizationStatus(_ callback: @escaping (String) -> Void) {
        UNUserNotificationCenter.current().getNotificationSettings { settings in
            let value: String
            switch settings.authorizationStatus {
            case .authorized, .provisional, .ephemeral: value = "authorized"
            case .denied: value = "denied"
            default: value = "not_determined"
            }
            DispatchQueue.main.async { callback(value) }
        }
    }

    @objc public static func requestAuthorization(_ callback: @escaping (Bool) -> Void) {
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .badge, .sound]) { granted, _ in
            DispatchQueue.main.async { callback(granted) }
        }
    }

    @objc public static func register(_ success: @escaping (String) -> Void, _ failure: @escaping (String) -> Void) {
        tokenCallback = success
        failureCallback = failure
        installObserver()
        DispatchQueue.main.async { UIApplication.shared.registerForRemoteNotifications() }
    }

    // The custom base AppDelegate posts these notifications from its APNs
    // callbacks. Keeping the hook explicit avoids replacing DCloud's delegate.
    @objc public static func didRegisterDeviceToken(_ token: Data) {
        let value = tokenString(token)
        NotificationCenter.default.post(name: .akPushDidRegister, object: value)
    }

    @objc public static func tokenString(_ token: Data) -> String {
        token.map { String(format: "%02x", $0) }.joined()
    }

    @objc public static func didFailRegistration() {
        NotificationCenter.default.post(name: .akPushDidFail, object: nil)
    }

    @objc public static func normalizedEvent(_ userInfo: [AnyHashable: Any]) -> String {
        guard JSONSerialization.isValidJSONObject(userInfo),
              let data = try? JSONSerialization.data(withJSONObject: userInfo),
              let value = String(data: data, encoding: .utf8) else { return "" }
        return value
    }

    @objc public static func unregister() {
        DispatchQueue.main.async { UIApplication.shared.unregisterForRemoteNotifications() }
        tokenCallback = nil
        failureCallback = nil
    }

    @objc public static func openSettings() -> Bool {
        guard let target = URL(string: UIApplication.openSettingsURLString), UIApplication.shared.canOpenURL(target) else { return false }
        DispatchQueue.main.async { UIApplication.shared.open(target) }
        return true
    }

    @objc public static func sdkVersion() -> String { return UIDevice.current.systemVersion }

    private static func installObserver() {
        guard observer == nil else { return }
        observer = NotificationCenter.default.addObserver(forName: .akPushDidRegister, object: nil, queue: .main) { note in
            guard let token = note.object as? String, !token.isEmpty else { failureCallback?("token_empty"); return }
            tokenCallback?(token)
        }
        NotificationCenter.default.addObserver(forName: .akPushDidFail, object: nil, queue: .main) { _ in failureCallback?("registration_failed") }
    }
}

private extension Notification.Name {
    static let akPushDidRegister = Notification.Name("com.appkernia.push.did-register")
    static let akPushDidFail = Notification.Name("com.appkernia.push.did-fail")
}
