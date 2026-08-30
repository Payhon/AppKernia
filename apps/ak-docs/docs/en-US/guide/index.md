---
title: Get started
description: Choose the best way to start AppKernia.
---

# Get started

You do not need to understand every blueprint before trying AppKernia. Run the complete system first, then go deeper into Mobile, Admin, or Server as your work requires.

If you are still deciding whether it fits your project, start with [What is AppKernia?](./what-is-appkernia) for the origin, name, technology choices, real product surfaces, and future direction.

| Goal                                | Start here                                           | Requirements                       |
| ----------------------------------- | ---------------------------------------------------- | ---------------------------------- |
| See the complete system quickly     | [Docker quick start](./quick-start)                  | Docker Desktop and Git             |
| Change Go or React source           | [Source development](./source-development)           | Go 1.26, Node 24, pnpm 11, Docker  |
| Build Android, iOS, or HarmonyOS    | [Mobile development](./mobile-development)           | HBuilderX and platform toolchains  |
| Build custom bases or releases      | [Mobile packaging](./mobile-packaging)               | HBuilderX, DevEco, and signing     |
| Configure multi-provider push       | [Push channel configuration](./push-channels)        | Provider accounts and app identity |
| Inspect queues and handle failures  | [Notification operations](./notification-operations) | Admin notification permission      |
| Integrate or inspect OS permissions | [Mobile permission center](./mobile-permissions)     | Target-platform toolchain          |
| Understand the repository first     | [Repository structure](./project-structure)          | 5–10 minutes                       |
| Decide whether the project fits     | [What is AppKernia?](./what-is-appkernia)            | 8–12 minutes                       |

<div class="ak-doc-callout"><strong>Project maturity</strong>AppKernia has not published a stable release yet. APIs, migrations, and component contracts may change. Before production use, complete security, performance, and platform acceptance in your own environment.</div>

## What a successful first run means

Do not treat “the command printed no error” as the finish line. A complete local start should meet all of these checks:

1. PostgreSQL, API, and Admin are healthy in `docker compose ps`; Migration and Seed exit with `0`.
2. `http://localhost:4173/healthz` succeeds, the Admin sign-in page loads, and the browser has no blocking console errors.
3. Sign in with a local account you created through `bootstrap-admin`; the repository never ships a fixed default password.
4. Dashboard summary and trend requests finish instead of remaining in Skeleton or error states.
5. Count a Mobile platform as passed only after its real build, install, or runtime check completes on that target.

If any check fails, go to [troubleshooting](./troubleshooting), keep the first failing command and exit code, and avoid diagnosing from downstream errors.

## Where to go after the first run

| Goal                                 | Next step                                                                                                                       |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| Add a business API                   | Read [server conventions](../api/conventions) and the OpenAPI source                                                            |
| Add an Admin page                    | Confirm permission, menu, generated client, and bilingual keys                                                                  |
| Add a Mobile page or component       | Continue with [Mobile development](./mobile-development) and AK UI                                                              |
| Integrate business notifications     | Read [notification architecture](../concepts/notification-architecture) and the [notification API](../api/mobile-notifications) |
| Configure and preflight providers    | Prepare accounts and build identity with [push channel configuration](./push-channels)                                          |
| Investigate queue or delivery errors | Use [notification operations](./notification-operations)                                                                        |
| Package a custom base or release     | Follow [Mobile packaging](./mobile-packaging) for toolchains and signing                                                        |
| Change auth, authorization, tenancy  | Read [authentication](../concepts/authentication) and ownership boundaries                                                      |
| Prepare a contribution               | Follow the [contribution guide](../community/contributing) and evidence checks                                                  |

## Suggested reading order

1. [Run from the source repository](./quick-start)
2. [What is AppKernia?](./what-is-appkernia)
3. [Architecture](../concepts/architecture)
4. [Authentication and sessions](../concepts/authentication)
5. [Notification and push architecture](../concepts/notification-architecture)
6. [Server API](../api/)
7. [Mobile components](../mobile-components/)
8. [Mobile packaging](./mobile-packaging)
9. [Contributing](../community/contributing)
