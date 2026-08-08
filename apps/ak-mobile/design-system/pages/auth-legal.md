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
