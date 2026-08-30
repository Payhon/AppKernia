import Foundation
import UIKit
import UserNotifications

@objc(AkPermissionsIOSBridge)
public final class AkPermissionsIOSBridge: NSObject {
    @objc public static func notificationStatus(_ callback: @escaping (String) -> Void) {
        UNUserNotificationCenter.current().getNotificationSettings { settings in
            let value: String
            switch settings.authorizationStatus {
            case .authorized: value = "authorized"
            case .provisional, .ephemeral: value = "limited"
            case .denied: value = "denied"
            case .notDetermined: value = "not_determined"
            @unknown default: value = "restricted"
            }
            DispatchQueue.main.async { callback(value) }
        }
    }

    @objc public static func requestNotifications(_ callback: @escaping (String) -> Void) {
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .badge, .sound]) { _, _ in notificationStatus(callback) }
    }

    @objc public static func openNotificationSettings() -> Bool {
        let raw: String
        if #available(iOS 16.0, *) { raw = UIApplication.openNotificationSettingsURLString }
        else { raw = UIApplication.openSettingsURLString }
        guard let target = URL(string: raw), UIApplication.shared.canOpenURL(target) else { return false }
        DispatchQueue.main.async { UIApplication.shared.open(target) }
        return true
    }
}
