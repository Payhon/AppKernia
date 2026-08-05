import Foundation
import Security

/**
 * Kept in the plugin target so no keychain entitlement is shared with another
 * application. Values use ThisDeviceOnly and never enter iCloud backup.
 */
@objc(AkSecureStorageIOSBridge)
public final class AkSecureStorageIOSBridge: NSObject {
    private static let service = "com.appkernia.ak-secure-storage.v1"
    private static let namespacePrefix = "entry."

    @objc public static func isAvailable() -> Bool {
        return true
    }

    @objc public static func write(_ key: String, value: String) -> Bool {
        guard let data = value.data(using: .utf8) else { return false }
        let query = baseQuery(key)
        let update: [String: Any] = [kSecValueData as String: data]
        let status = SecItemUpdate(query as CFDictionary, update as CFDictionary)
        if status == errSecSuccess { return true }
        if status != errSecItemNotFound { return false }
        var add = query
        add[kSecValueData as String] = data
        add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        return SecItemAdd(add as CFDictionary, nil) == errSecSuccess
    }

    @objc public static func read(_ key: String) -> String? {
        var query = baseQuery(key)
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var result: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        guard status == errSecSuccess, let data = result as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    @objc public static func remove(_ key: String) -> Bool {
        let status = SecItemDelete(baseQuery(key) as CFDictionary)
        return status == errSecSuccess || status == errSecItemNotFound
    }

    @objc public static func clearNamespace() -> Bool {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
        ]
        let status = SecItemDelete(query as CFDictionary)
        return status == errSecSuccess || status == errSecItemNotFound
    }

    private static func baseQuery(_ key: String) -> [String: Any] {
        return [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: namespacePrefix + key,
        ]
    }
}
