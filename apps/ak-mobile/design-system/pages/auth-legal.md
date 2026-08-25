# Authentication and Legal Page Override

- Keep form cards within 16 px gutters and place the product mark/title above the first field.
- Use the system sans-serif defined by the Master; do not introduce external fonts or marketing-style artwork.
- Form errors are inline text in addition to the field border. Loading disables duplicate submission without changing layout.
- Terms and privacy actions are visually secondary but remain visible on the login and consent paths.
- Legal text uses 16 px body / 28 px line height, restrained callouts, and normal text views only.
- Keep one full-width primary login action. Place password recovery and account registration in a separate secondary text-link row below it, with at least 8 px between adjacent targets and a 44 px minimum tap height.
- Primary authentication labels are explicitly white in the AK button component; never depend on text-color inheritance through a slot.
- Authentication hero, navigation and legal content start below the status-bar safe area on every supported device.
- Custom navigation bars must use an explicit AK back action that invokes platform navigation history; when no prior page exists, guest/legal surfaces fall back to the login route instead of becoming a dead end.
- The first-install privacy gate is an offline pre-bootstrap surface. Below the status bar, use one top-rounded neutral container that fills the remaining viewport, with the packaged icon/name/subtitle centered and the legal statement plus actions anchored near the safe-area bottom.
- Legal links open the packaged allowlisted pages in the current navigation container and return to the unchanged consent state. They must not trigger public configuration or any sensitive SDK initialization.
- Keep Cancel and Agree as equal-width 44 px minimum actions. Android/Harmony may leave through the platform exit port; iOS remains on the blocking surface because applications cannot programmatically terminate there.
- Long localized statements and dynamic type must scroll without pushing either legal links or actions outside the reachable safe area.
