---
title: Get started
description: Choose the best way to start AppKernia.
---

# Get started

You do not need to understand every blueprint before trying AppKernia. Run the complete system first, then go deeper into Mobile, Admin, or Server as your work requires.

| Goal                             | Start here                                  | Requirements                      |
| -------------------------------- | ------------------------------------------- | --------------------------------- |
| See the complete system quickly  | [Docker quick start](./quick-start)         | Docker Desktop and Git            |
| Change Go or React source        | [Source development](./source-development)  | Go 1.26, Node 24, pnpm 11, Docker |
| Build Android, iOS, or HarmonyOS | [Mobile development](./mobile-development)  | HBuilderX and platform toolchains |
| Understand the repository first  | [Repository structure](./project-structure) | 5–10 minutes                      |

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

| Goal                                | Next step                                                                      |
| ----------------------------------- | ------------------------------------------------------------------------------ |
| Add a business API                  | Read [server conventions](../api/conventions) and the OpenAPI source           |
| Add an Admin page                   | Confirm permission, menu, generated client, and bilingual keys                 |
| Add a Mobile page or component      | Continue with [Mobile development](./mobile-development) and AK UI             |
| Change auth, authorization, tenancy | Read [authentication](../concepts/authentication) and ownership boundaries     |
| Prepare a contribution              | Follow the [contribution guide](../community/contributing) and evidence checks |

## Suggested reading order

1. [Run from the source repository](./quick-start)
2. [Architecture](../concepts/architecture)
3. [Authentication and sessions](../concepts/authentication)
4. [Server API](../api/)
5. [Mobile components](../mobile-components/)
6. [Contributing](../community/contributing)
