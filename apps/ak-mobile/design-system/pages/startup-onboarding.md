# Startup Onboarding Page Override

This page inherits `../MASTER.md` and appears only after privacy acceptance, public App configuration and the mandatory upgrade gate.

- Render the currently published locale-specific images as the primary content. Do not add marketing titles, subtitles or a skip action outside the versioned images.
- Preload every image before entering this route. A failed preload is a fail-open bootstrap event and must not mark the version complete.
- Use a swipeable viewport with a text-independent page indicator. Announce the current position and give every image the localized non-visual description from the public configuration.
- The final primary action is visible only on the last page and remains disabled until every position has been visited. It uses a 44 px minimum target and the standard AK blue action token.
- Completion stores the highest observed published version for the packaged public App UUID. A higher version is shown again; rollback to an older version is not.
- Block accidental system-back completion. Rotation, safe areas and larger text must keep the image viewport, indicator and final action reachable without page-level horizontal overflow.
