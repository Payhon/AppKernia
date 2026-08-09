---
title: Community
description: Contribute to AppKernia through code, docs, tests, issues, and Stars.
---

# Community

AppKernia addresses problems developers experience together, so developers should be able to shape it together.

You do not need to be a Go, React, UTS, or HarmonyOS expert first. A clear reproduction, a documentation fix, a missing error state, or one physical-device check may be more valuable than a large rewrite.

## Contribution ladder

| Start here      | What you can do                                                                  | What matters                             |
| --------------- | -------------------------------------------------------------------------------- | ---------------------------------------- |
| Issue           | Minimal reproduction, environment, first failing command, expected/actual result | Repeatable evidence without credentials  |
| Documentation   | Fix copy, links, examples, language gaps, evidence boundaries                    | Paired zh-CN/en-US and real commands     |
| Tests           | Unit, contract, integration, E2E, or platform regression                         | Show that the test fails on old behavior |
| Device evidence | Recheck Mobile on a named OS, version, and device                                | Separate build, simulator, and hardware  |
| Code            | Change Server, Admin, Mobile, or infrastructure through the contract chain       | Small scope, complete tests, no secrets  |

## The path of a pull request

1. Search existing issues, roadmap, ADRs, and the relevant blueprint before duplicating work.
2. Link or open an issue for behavior changes and state scope, acceptance, and explicit non-goals.
3. Implement in small steps; API changes update OpenAPI, database/permissions when relevant, generated clients, and tests.
4. Maintain `zh-CN` and `en-US` for visible copy; UI changes keep design decisions and screenshot evidence.
5. Run the affected check/build/test commands and list commands, exit codes, and unverified boundaries in the PR.
6. Respond to review with evidence and tradeoffs; do not hide the original problem inside unrelated rewrites.

<div class="ak-doc-callout"><strong>First contribution</strong>If you are unsure where to begin, open an issue with a complete reproduction or fix one piece of documentation that just cost you time. Maintainers can help route it to the right blueprint and evidence level.</div>

- [Start contributing](./contributing)
- [Support the project and Star](./support-project)
- [Report a vulnerability](./security)
- [MIT License](./license)

Follow `CODE_OF_CONDUCT.md` in the repository. Keep discussion professional, kind, and centered on evidence and the problem.

If the direction helps you, leave a Star on [GitHub](https://github.com/Payhon/AppKernia). We do not publish invented metrics; every real Star helps another developer looking for a complete cross-platform foundation find the project.
