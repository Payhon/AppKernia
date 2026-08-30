# ak-permissions

`ak-permissions` is the single operating-system permission source for the
mobile application. The registry is compile-time only: a capability is shown
only when it is enabled and implemented by the current build.

The current release enables `notifications` and `camera`. Photos, file picker,
microphone, location and Bluetooth remain registered but disabled until a real
feature requires them. Reading status never displays a system prompt and OS
status is never uploaded to AppKernia.

Native verification is still required on Android 13+, iOS and HarmonyOS NEXT.
