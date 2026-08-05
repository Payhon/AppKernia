## Design System: AppKernia Admin Login CAPTCHA

### Pattern
- **Name:** Enterprise Gateway
- **Conversion Focus:**  logo carousel,  tab switching for industries, Path selection (I am a...). Mega menu navigation. Trust signals prominent.
- **CTA Placement:** Contact Sales (Primary) + Login (Secondary)
- **Color Strategy:** Corporate: Navy/Grey. High integrity. Conservative accents.
- **Sections:** 1. Hero (Video/Mission), 2. Solutions by Industry, 3. Solutions by Role, 4. Client Logos, 5. Contact Sales

### Style
- **Name:** Vibrant & Block-based
- **Keywords:** Bold, energetic, playful, block layout, geometric shapes, high color contrast, duotone, modern, energetic
- **Best For:** Startups, creative agencies, gaming, social media, youth-focused, entertainment, consumer
- **Performance:** ⚡ Good | **Accessibility:** ◐ Ensure WCAG

### Colors
| Role | Hex |
|------|-----|
| Primary | #0369A1 |
| Secondary | #0EA5E9 |
| CTA | #22C55E |
| Background | #F0F9FF |
| Text | #0C4A6E |

*Notes: Security blue + protected green*

### Typography
- **Heading:** Inter
- **Body:** Inter
- **Mood:** minimal, clean, swiss, functional, neutral, professional
- **Best For:** Dashboards, admin panels, documentation, enterprise apps, design systems
- **Google Fonts:** https://fonts.google.com/share?selection.family=Inter:wght@300;400;500;600;700
- **CSS Import:**
```css
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
```

### Key Effects
Large sections (48px+ gaps), animated patterns, bold hover (color shift), scroll-snap, large type (32px+), 200-300ms

### Avoid (Anti-patterns)
- Complex onboarding flow
- Cluttered layout

### Pre-Delivery Checklist
- [ ] No emojis as icons (use SVG: Heroicons/Lucide)
- [ ] cursor-pointer on all clickable elements
- [ ] Hover states with smooth transitions (150-300ms)
- [ ] Light mode: text contrast 4.5:1 minimum
- [ ] Focus states visible for keyboard nav
- [ ] prefers-reduced-motion respected
- [ ] Responsive: 375px, 768px, 1024px, 1440px

## UI Pro Max Search Results
**Domain:** ux | **Query:** captcha refresh accessibility validation error retry
**Source:** ux-guidelines.csv | **Found:** 12 results

### Result 1
- **Category:** Accessibility
- **Issue:** Error Messages
- **Platform:** All
- **Description:** Error messages must be announced
- **Do:** Use aria-live or role=alert for errors
- **Don't:** Visual-only error indication
- **Code Example Good:** role='alert'
- **Code Example Bad:** Red border only
- **Severity:** High

### Result 2
- **Category:** Touch
- **Issue:** Pull to Refresh
- **Platform:** Mobile
- **Description:** Accidental refresh is frustrating
- **Do:** Disable where not needed
- **Don't:** Enable by default everywhere
- **Code Example Good:** overscroll-behavior: contain
- **Code Example Bad:** Default overscroll
- **Severity:** Low

### Result 3
- **Category:** Forms
- **Issue:** Inline Validation
- **Platform:** All
- **Description:** Validate as user types or on blur
- **Do:** Validate on blur for most fields
- **Don't:** Validate only on submit
- **Code Example Good:** onBlur validation
- **Code Example Bad:** Submit-only validation
- **Severity:** Medium

### Result 4
- **Category:** Feedback
- **Issue:** Error Recovery
- **Platform:** All
- **Description:** Help users recover from errors
- **Do:** Provide clear next steps
- **Don't:** Error without recovery path
- **Code Example Good:** Try again button + help link
- **Code Example Bad:** Error message only
- **Severity:** Medium

### Result 5
- **Category:** Interaction
- **Issue:** Error Feedback
- **Platform:** All
- **Description:** Users need to know when something fails
- **Do:** Show clear error messages near problem
- **Don't:** Silent failures with no feedback
- **Code Example Good:** Red border + error message
- **Code Example Bad:** No indication of error
- **Severity:** High

### Result 6
- **Category:** Forms
- **Issue:** Error Placement
- **Platform:** All
- **Description:** Errors should appear near the problem
- **Do:** Show error below related input
- **Don't:** Single error message at top of form
- **Code Example Good:** Error under each field
- **Code Example Bad:** All errors at form top
- **Severity:** Medium

### Result 7
- **Category:** Accessibility
- **Issue:** Alt Text
- **Platform:** All
- **Description:** Images need text alternatives
- **Do:** Descriptive alt text for meaningful images
- **Don't:** Empty or missing alt attributes
- **Code Example Good:** alt='Dog playing in park'
- **Code Example Bad:** alt='' for content images
- **Severity:** High

### Result 8
- **Category:** Accessibility
- **Issue:** Color Contrast
- **Platform:** All
- **Description:** Text must be readable against background
- **Do:** Minimum 4.5:1 ratio for normal text
- **Don't:** Low contrast text
- **Code Example Good:** #333 on white (7:1)
- **Code Example Bad:** #999 on white (2.8:1)
- **Severity:** High

### Result 9
- **Category:** Accessibility
- **Issue:** Color Only
- **Platform:** All
- **Description:** Don't convey information by color alone
- **Do:** Use icons/text in addition to color
- **Don't:** Red/green only for error/success
- **Code Example Good:** Red text + error icon
- **Code Example Bad:** Red border only for error
- **Severity:** High

### Result 10
- **Category:** Accessibility
- **Issue:** ARIA Labels
- **Platform:** All
- **Description:** Interactive elements need accessible names
- **Do:** Add aria-label for icon-only buttons
- **Don't:** Icon buttons without labels
- **Code Example Good:** aria-label='Close menu'
- **Code Example Bad:** <button><Icon/></button>
- **Severity:** High

### Result 11
- **Category:** Accessibility
- **Issue:** Keyboard Navigation
- **Platform:** Web
- **Description:** All functionality accessible via keyboard
- **Do:** Tab order matches visual order
- **Don't:** Keyboard traps or illogical tab order
- **Code Example Good:** tabIndex for custom order
- **Code Example Bad:** Unreachable elements
- **Severity:** High

### Result 12
- **Category:** Accessibility
- **Issue:** Form Labels
- **Platform:** All
- **Description:** Inputs must have associated labels
- **Do:** Use label with for attribute or wrap input
- **Don't:** Placeholder-only inputs
- **Code Example Good:** <label for='email'>
- **Code Example Bad:** placeholder='Email' only
- **Severity:** High

## UI Pro Max Stack Guidelines
**Stack:** react | **Query:** login form captcha conditional field async validation
**Source:** stacks/react.csv | **Found:** 3 results

### Result 1
- **Category:** ErrorHandling
- **Guideline:** Handle async errors
- **Description:** Catch errors in async operations
- **Do:** try/catch in async handlers
- **Don't:** Unhandled promise rejections
- **Code Good:** try { await fetch() } catch(e) {}
- **Code Bad:** await fetch() // no catch
- **Severity:** High
- **Docs URL:**

### Result 2
- **Category:** Props
- **Guideline:** Validate props with TypeScript
- **Description:** Use TypeScript interfaces for prop types
- **Do:** interface Props { name: string }
- **Don't:** PropTypes or no validation
- **Code Good:** interface ButtonProps { onClick: () => void }
- **Code Bad:** Button.propTypes = {}
- **Severity:** Medium
- **Docs URL:**

### Result 3
- **Category:** Accessibility
- **Guideline:** Label form controls
- **Description:** Associate labels with inputs
- **Do:** htmlFor matching input id
- **Don't:** Placeholder as only label
- **Code Good:** <label htmlFor="email">Email</label>
- **Code Bad:** <input placeholder="Email"/>
- **Severity:** High
- **Docs URL:**

## UI Pro Max Search Results
**Domain:** web | **Query:** captcha image alt keyboard refresh focus error
**Source:** web-interface.csv | **Found:** 7 results

### Result 1
- **Category:** Forms
- **Issue:** Inline Errors
- **Platform:** Web
- **Description:** Show error messages inline near the problem field
- **Do:** Inline error with focus on first error
- **Don't:** Single error at top
- **Code Example Good:** <input /><span class='text-red-500'>{error}</span>
- **Code Example Bad:** <div class='error'>{allErrors}</div> // at top
- **Severity:** High

### Result 2
- **Category:** Accessibility
- **Issue:** Keyboard Handlers
- **Platform:** Web
- **Description:** Interactive elements must support keyboard interaction
- **Do:** Add onKeyDown alongside onClick
- **Don't:** Click-only interaction
- **Code Example Good:** <div onClick={fn} onKeyDown={fn} tabIndex={0}>
- **Code Example Bad:** <div onClick={fn}>
- **Severity:** High

### Result 3
- **Category:** Focus
- **Issue:** Visible Focus States
- **Platform:** Web
- **Description:** All interactive elements need visible focus states
- **Do:** Use :focus-visible with ring/outline
- **Don't:** No focus indication
- **Code Example Good:** focus-visible:ring-2 focus-visible:ring-blue-500
- **Code Example Bad:** outline-none // no replacement
- **Severity:** Critical

### Result 4
- **Category:** Performance
- **Issue:** Lazy Load Images
- **Platform:** Web
- **Description:** Lazy-load images below the fold
- **Do:** Use loading='lazy' for below-fold images
- **Don't:** Load all images eagerly
- **Code Example Good:** <img loading='lazy' src='...' />
- **Code Example Bad:** <img src='...' /> // above fold only
- **Severity:** Medium
### Result 5
- **Category:** Focus
- **Issue:** Never Remove Outline
- **Platform:** Web
- **Description:** Never remove outline without providing replacement
- **Do:** Replace outline with visible alternative
- **Don't:** Remove outline completely
- **Code Example Good:** focus:outline-none focus:ring-2
- **Code Example Bad:** focus:outline-none // nothing else
- **Severity:** Critical

### Result 6
- **Category:** Anti-Pattern
- **Issue:** Outline Replacement
- **Platform:** Web
- **Description:** Never use outline-none without replacement
- **Do:** Provide visible focus replacement
- **Don't:** Remove outline with nothing
- **Code Example Good:** focus:outline-none focus:ring-2 focus:ring-blue-500
- **Code Example Bad:** focus:outline-none // alone
- **Severity:** Critical

### Result 7
- **Category:** Focus
- **Issue:** Checkbox Radio Hit Target
- **Platform:** Web
- **Description:** Checkbox/radio must share hit target with label
- **Do:** Wrap input and label together
- **Don't:** Separate tiny checkbox
- **Code Example Good:** <label class='flex gap-2'><input type='checkbox' /><span>Option</span></label>
- **Code Example Bad:** <input type='checkbox' id='x' /><label for='x'>Option</label>
- **Severity:** Medium
