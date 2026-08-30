# ak-push

`ak-push` is the only push boundary business pages may call. It is deliberately inert until the user has accepted the legal documents and explicitly enables Push.

Android custom bases must be built with `AK_ANDROID_PUSH_VARIANT=google|china`. The Google variant contains only FCM; the China variant contains only vendor SDKs declared by the generated native configuration. iOS uses APNs and HarmonyOS NEXT uses Push Kit. Generated provider files and credentials are never committed.

The native channel returns an opaque registration Token. The authenticated repository sends it to `/api/v1/me/push-devices`; neither UI nor logs print it. Active registrations are refreshed at authenticated startup, and the iOS lifecycle hook also forwards APNs token changes. Notification payload routing is limited to `delivery_id`, `message_id`, an allowlisted `route_key`, and opaque resource identifiers.

Android automatic permission requests are disabled in every generated variant. Firebase Messaging and Huawei Push Kit automatic initialization are disabled; registration runs only after legal consent and the server-side Push preference are both active. Vendor SDKs that cannot demonstrate delayed initialization during the release privacy review must remain disabled.

SDK account files, signing profiles and real devices are external acceptance prerequisites. A successful UTS compile does not prove notification delivery.
