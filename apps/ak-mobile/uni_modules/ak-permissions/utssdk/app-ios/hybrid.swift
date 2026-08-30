import Foundation
import UIKit
import UserNotifications
import AVFoundation

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

    @objc public static func cameraStatus() -> String {
        switch AVCaptureDevice.authorizationStatus(for: .video) {
        case .authorized: return "authorized"
        case .denied: return "denied"
        case .restricted: return "restricted"
        case .notDetermined: return "not_determined"
        @unknown default: return "unavailable"
        }
    }

    @objc public static func requestCamera(_ callback: @escaping (String) -> Void) {
        AVCaptureDevice.requestAccess(for: .video) { _ in
            DispatchQueue.main.async { callback(cameraStatus()) }
        }
    }

    @objc public static func openCameraSettings() -> Bool {
        guard let target = URL(string: UIApplication.openSettingsURLString), UIApplication.shared.canOpenURL(target) else { return false }
        DispatchQueue.main.async { UIApplication.shared.open(target) }
        return true
    }
}
