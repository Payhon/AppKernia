# Review checklist

- [x] Login remains the only full-width primary action.
- [x] Forgot password and registration are text links below Login.
- [x] Adjacent link targets have at least 8 px separation and 44 px tap height.
- [x] Registration link still follows `registration_enabled`.
- [x] `zh-CN` and `en-US` catalogs remain aligned.
- [x] Coordinate taps open password recovery, registration and both legal destinations.
- [x] Coordinate taps on the top back control return to the immediately previous page.
- [x] A direct guest/legal entry falls back safely when no history exists.
- [x] Mobile static checks and HBuilderX iOS compilation pass.
- [x] iPhone 16 Pro / iOS 18.6 screenshots show the final login and authenticated Home state.

Runtime scope: the final interaction pass used an iPhone 16 Pro / iOS 18.6
simulator and a locally rebuilt simulator base containing the native secure-storage
module. It does not represent iOS hardware, Android or HarmonyOS validation.
