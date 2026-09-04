---
title: ak-interactive-captcha
description: Four-mode uni-app x CAPTCHA component and SMS-gate integration.
---

# `ak-interactive-captcha`

`ak-interactive-captcha` lives in `apps/ak-mobile/uni_modules/ak-interactive-captcha` and supports the uni-app x Android, iOS, and HarmonyOS Core targets. It renders a server challenge and emits an answer; it never owns networking, persists a token, or sends SMS.

```vue
<ak-interactive-captcha
  :open="captchaOpen"
  :challenge="captchaChallenge"
  :loading="captchaLoading"
  :error-text="captchaError"
  @confirm="confirmCaptcha"
  @refresh="refreshCaptcha"
  @close="closeCaptcha"
/>
```

## Props and events

| Prop        | Type                                      | Meaning                            |
| ----------- | ----------------------------------------- | ---------------------------------- |
| `open`      | `boolean`                                 | Shows the modal                    |
| `challenge` | `AkInteractiveCaptchaChallenge` or `null` | Current server challenge           |
| `loading`   | `boolean`                                 | Blocks confirm, close, and refresh |
| `errorText` | `string`                                  | Caller-translated visible error    |

| Event     | Payload                        | Caller responsibility                     |
| --------- | ------------------------------ | ----------------------------------------- |
| `confirm` | `AkInteractiveCaptchaResponse` | Wrap the answer and call the SMS endpoint |
| `refresh` | None                           | Request a new challenge                   |
| `close`   | None                           | Close and release caller-owned state      |

Changing the challenge, closing, refreshing, or reaching the 300-second expiry resets old points and angles. Tapping the backdrop does not dismiss the modal.

## Interaction types

| Type     | Interaction                    | `confirm` output                     |
| -------- | ------------------------------ | ------------------------------------ |
| `click`  | Tap prompt targets in order    | `{ type: 'click', points: [...] }`   |
| `slide`  | Move the native slider         | `{ type: 'slide', point: { x, y } }` |
| `drag`   | Drag the puzzle tile directly  | `{ type: 'drag', point: { x, y } }`  |
| `rotate` | Rotate the thumbnail by slider | `{ type: 'rotate', angle }`          |

Click and drag coordinates are normalized to the server image's original dimensions. Never submit raw display coordinates.

## Repository integration

App bootstrap configures `captchaRuntime` with the shared `AkHttpClient`. A feature creates a business intent and owns the returned challenge:

```typescript
const repository = captchaRuntime.getRepository();
if (repository == null) return;

repository.create(
  { scene: 'login', mobile: '+15551234567', identifierId: '', purpose: '', resource: '' },
  (challenge) => {
    this.captchaChallenge = challenge;
    this.captchaOpen = true;
  },
  (error) => {
    this.captchaError = akI18n.t(error.messageKey, null);
  },
);
```

On `confirm(response)`, call `repository.submission(challenge, response)` to build `{ id, token, response }`, then immediately call the original SMS endpoint. Close the modal after success and start the OTP countdown from the server's `retry_after_seconds`. Keep the modal on recoverable errors; refresh when the proof is expired, invalid, or exhausted.

Every send and resend needs a fresh challenge. Email OTP never opens this component. See [Mobile authentication API](../api/mobile-auth#interactive-captcha-before-sms-delivery) for the HTTP contract.

## Reuse boundaries

- Reuse the existing `captchaRuntime` and models; do not add networking or business scenes to the component.
- Treat `token` as opaque transit data: never parse, cache, or log it.
- `AkI18n` supplies visible strings. The component includes safe-area padding, 44px controls, error announcements, and no decorative animation.
- Supported scope is uni-app x Android/iOS/HarmonyOS Core, not classic uni-app, H5, or mini programs.
