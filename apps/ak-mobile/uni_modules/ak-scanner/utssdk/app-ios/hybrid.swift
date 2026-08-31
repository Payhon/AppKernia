import Foundation

@objc(AkScannerIOSBridge)
public final class AkScannerIOSBridge: NSObject {
    @objc public static func isSimulator() -> Bool {
        #if targetEnvironment(simulator)
        return true
        #else
        return false
        #endif
    }
}
