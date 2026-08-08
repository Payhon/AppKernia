# Mobile acceptance repair screenshots

Runtime: HBuilderX 5.06 generated iOS app resources, iPhone 16 Pro simulator, iOS 18.6, `zh-CN`.

- `ios-iphone16pro-login-auth-links.zh-CN.png`: login, forgot password, registration, privacy policy and terms links render without a white screen.
- `ios-iphone16pro-forgot-password.zh-CN.png`: password-reset email and verification-code request screen.
- `ios-iphone16pro-register-form.zh-CN.png`: registration form after both required legal pages are published in the isolated acceptance database.
- `ios-iphone16pro-privacy-policy.zh-CN.png`: privacy policy loaded from the local Go API.
- `ios-iphone16pro-terms-of-service.zh-CN.png`: terms of service loaded from the local Go API.

The HBuilderX UI completed compilation, but its resource-sync phase stalled on this machine. For runtime acceptance, the freshly generated `unpackage/dist/dev/app-ios` resources were copied into the already installed official standard-base data container before launch. This proves the compiled app resources render and interact in the simulator; it does not prove HBuilderX one-click resource sync, a custom base, secure-storage runtime behavior, a physical device, Android or HarmonyOS.
