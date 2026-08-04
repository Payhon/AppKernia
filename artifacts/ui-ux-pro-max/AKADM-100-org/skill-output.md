# AKADM-100 Organization UI Skill Output

四条项目本地 `ui-ux-pro-max` 命令均真实执行并退出 0。

## Departments design system

## Design System: AppKernia Admin Departments

### Pattern
- **Name:** Enterprise Gateway
- **Conversion Focus:**  logo carousel,  tab switching for industries, Path selection (I am a...). Mega menu navigation. Trust signals prominent.
- **CTA Placement:** Contact Sales (Primary) + Login (Secondary)
- **Color Strategy:** Corporate: Navy/Grey. High integrity. Conservative accents.
- **Sections:** 1. Hero (Video/Mission), 2. Solutions by Industry, 3. Solutions by Role, 4. Client Logos, 5. Contact Sales

### Style
- **Name:** Accessible & Ethical
- **Keywords:** High contrast, large text (16px+), keyboard navigation, screen reader friendly, WCAG compliant, focus state, semantic
- **Best For:** Government, healthcare, education, inclusive products, large audience, legal compliance, public
- **Performance:** ⚡ Excellent | **Accessibility:** ✓ WCAG AAA

### Colors
| Role | Hex |
|------|-----|
| Primary | #7C3AED |
| Secondary | #A78BFA |
| CTA | #F97316 |
| Background | #FAF5FF |
| Text | #4C1D95 |

*Notes: Excitement purple + action orange*

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
Clear focus rings (3-4px), ARIA labels, skip links, responsive design, reduced motion, 44x44px touch targets

### Avoid (Anti-patterns)
- Outdated design
- Hidden info

### Pre-Delivery Checklist
- [ ] No emojis as icons (use SVG: Heroicons/Lucide)
- [ ] cursor-pointer on all clickable elements
- [ ] Hover states with smooth transitions (150-300ms)
- [ ] Light mode: text contrast 4.5:1 minimum
- [ ] Focus states visible for keyboard nav
- [ ] prefers-reduced-motion respected
- [ ] Responsive: 375px, 768px, 1024px, 1440px


============================================================
✅ Design system persisted to design-system/appkernia-admin-departments/
   📄 design-system/appkernia-admin-departments/MASTER.md (Global Source of Truth)
   📄 design-system/appkernia-admin-departments/pages/system-users-departments.md (Page Overrides)

📖 Usage: When building a page, check design-system/appkernia-admin-departments/pages/[page].md first.
   If exists, its rules override MASTER.md. Otherwise, use MASTER.md.
============================================================

## Positions design system

## Design System: AppKernia Admin Positions

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
| Primary | #2563EB |
| Secondary | #3B82F6 |
| CTA | #F97316 |
| Background | #F8FAFC |
| Text | #1E293B |

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
- Hidden benefits
- No community proof

### Pre-Delivery Checklist
- [ ] No emojis as icons (use SVG: Heroicons/Lucide)
- [ ] cursor-pointer on all clickable elements
- [ ] Hover states with smooth transitions (150-300ms)
- [ ] Light mode: text contrast 4.5:1 minimum
- [ ] Focus states visible for keyboard nav
- [ ] prefers-reduced-motion respected
- [ ] Responsive: 375px, 768px, 1024px, 1440px


============================================================
✅ Design system persisted to design-system/appkernia-admin-positions/
   📄 design-system/appkernia-admin-positions/MASTER.md (Global Source of Truth)
   📄 design-system/appkernia-admin-positions/pages/system-users-positions.md (Page Overrides)

📖 Usage: When building a page, check design-system/appkernia-admin-positions/pages/[page].md first.
   If exists, its rules override MASTER.md. Otherwise, use MASTER.md.
============================================================

## UX search

## UI Pro Max Search Results
**Domain:** ux | **Query:** tree management prevent cycle move keyboard fallback delete occupied confirmation filters empty error loading
**Source:** ux-guidelines.csv | **Found:** 12 results

### Result 1
- **Category:** Interaction
- **Issue:** Confirmation Dialogs
- **Platform:** All
- **Description:** Prevent accidental destructive actions
- **Do:** Confirm before delete/irreversible actions
- **Don't:** Delete without confirmation
- **Code Example Good:** Are you sure modal
- **Code Example Bad:** Direct delete on click
- **Severity:** High

### Result 2
- **Category:** Interaction
- **Issue:** Loading Buttons
- **Platform:** All
- **Description:** Prevent double submission during async actions
- **Do:** Disable button and show loading state
- **Don't:** Allow multiple clicks during processing
- **Code Example Good:** disabled={loading} spinner
- **Code Example Bad:** Button clickable while loading
- **Severity:** High

### Result 3
- **Category:** Accessibility
- **Issue:** Error Messages
- **Platform:** All
- **Description:** Error messages must be announced
- **Do:** Use aria-live or role=alert for errors
- **Don't:** Visual-only error indication
- **Code Example Good:** role='alert'
- **Code Example Bad:** Red border only
- **Severity:** High

### Result 4
- **Category:** Accessibility
- **Issue:** Keyboard Navigation
- **Platform:** Web
- **Description:** All functionality accessible via keyboard
- **Do:** Tab order matches visual order
- **Don't:** Keyboard traps or illogical tab order
- **Code Example Good:** tabIndex for custom order
- **Code Example Bad:** Unreachable elements
- **Severity:** High

### Result 5
- **Category:** Feedback
- **Issue:** Empty States
- **Platform:** All
- **Description:** Guide users when no content exists
- **Do:** Show helpful message and action
- **Don't:** Blank empty screens
- **Code Example Good:** No items yet. Create one!
- **Code Example Bad:** Empty white space
- **Severity:** Medium

### Result 6
- **Category:** Feedback
- **Issue:** Confirmation Messages
- **Platform:** All
- **Description:** Confirm successful actions
- **Do:** Brief success message
- **Don't:** Silent success
- **Code Example Good:** Saved successfully toast
- **Code Example Bad:** No confirmation
- **Severity:** Medium

### Result 7
- **Category:** Layout
- **Issue:** Z-Index Management
- **Platform:** Web
- **Description:** Stacking context conflicts cause hidden elements
- **Do:** Define z-index scale system (10 20 30 50)
- **Don't:** Use arbitrary large z-index values
- **Code Example Good:** z-10 z-20 z-50
- **Code Example Bad:** z-[9999]
- **Severity:** High

### Result 8
- **Category:** Accessibility
- **Issue:** Skip Links
- **Platform:** Web
- **Description:** Allow keyboard users to skip navigation
- **Do:** Provide skip to main content link
- **Don't:** No skip link on nav-heavy pages
- **Code Example Good:** Skip to main content link
- **Code Example Bad:** 100 tabs to reach content
- **Severity:** Medium

### Result 9
- **Category:** Feedback
- **Issue:** Error Recovery
- **Platform:** All
- **Description:** Help users recover from errors
- **Do:** Provide clear next steps
- **Don't:** Error without recovery path
- **Code Example Good:** Try again button + help link
- **Code Example Bad:** Error message only
- **Severity:** Medium

### Result 10
- **Category:** Performance
- **Issue:** Lazy Loading
- **Platform:** All
- **Description:** Load content as needed
- **Do:** Lazy load below-fold images and content
- **Don't:** Load everything upfront
- **Code Example Good:** loading='lazy'
- **Code Example Bad:** All images eager load
- **Severity:** Medium

### Result 11
- **Category:** Interaction
- **Issue:** Focus States
- **Platform:** All
- **Description:** Keyboard users need visible focus indicators
- **Do:** Use visible focus rings on interactive elements
- **Don't:** Remove focus outline without replacement
- **Code Example Good:** focus:ring-2 focus:ring-blue-500
- **Code Example Bad:** outline-none without alternative
- **Severity:** High

### Result 12
- **Category:** Interaction
- **Issue:** Error Feedback
- **Platform:** All
- **Description:** Users need to know when something fails
- **Do:** Show clear error messages near problem
- **Don't:** Silent failures with no feedback
- **Code Example Good:** Red border + error message
- **Code Example Bad:** No indication of error
- **Severity:** High

## React stack search

## UI Pro Max Stack Guidelines
**Stack:** react | **Query:** organization tree detail drawer position data table forms permissions accessibility
**Source:** stacks/react.csv | **Found:** 12 results

### Result 1
- **Category:** Forms
- **Guideline:** Controlled components for forms
- **Description:** Use state to control form inputs
- **Do:** value + onChange for inputs
- **Don't:** Uncontrolled inputs with refs
- **Code Good:** <input value={val} onChange={setVal}>
- **Code Bad:** <input ref={inputRef}>
- **Severity:** Medium
- **Docs URL:** https://react.dev/reference/react-dom/components/input#controlling-an-input-with-a-state-variable

### Result 2
- **Category:** ErrorHandling
- **Guideline:** Use error boundaries
- **Description:** Catch JavaScript errors in component tree
- **Do:** ErrorBoundary wrapping sections
- **Don't:** Let errors crash entire app
- **Code Good:** <ErrorBoundary><App/></ErrorBoundary>
- **Code Bad:** No error handling
- **Severity:** High
- **Docs URL:** https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary

### Result 3
- **Category:** Patterns
- **Guideline:** Container/Presentational split
- **Description:** Separate data logic from UI
- **Do:** Container fetches presentational renders
- **Don't:** Mixed data and UI in one
- **Code Good:** <UserContainer><UserView/></UserContainer>
- **Code Bad:** <User /> with fetch and render
- **Severity:** Low
- **Docs URL:** 

### Result 4
- **Category:** Props
- **Guideline:** Avoid prop drilling
- **Description:** Use context or composition for deeply nested data
- **Do:** Context for global data composition for UI
- **Don't:** Passing props through 5+ levels
- **Code Good:** <UserContext.Provider>
- **Code Bad:** <A user={u}><B user={u}><C user={u}>
- **Severity:** Medium
- **Docs URL:** https://react.dev/learn/passing-data-deeply-with-context

### Result 5
- **Category:** Context
- **Guideline:** Use context for global data
- **Description:** Context for theme auth locale
- **Do:** Context for app-wide state
- **Don't:** Context for frequently changing data
- **Code Good:** <ThemeContext.Provider>
- **Code Bad:** Context for form field values
- **Severity:** Medium
- **Docs URL:** https://react.dev/learn/passing-data-deeply-with-context

### Result 6
- **Category:** Effects
- **Guideline:** Avoid unnecessary effects
- **Description:** Don't use effects for transforming data or events
- **Do:** Transform data during render handle events directly
- **Don't:** useEffect for derived state or event handling
- **Code Good:** const filtered = items.filter(...)
- **Code Bad:** useEffect(() => setFiltered(items.filter(...)))
- **Severity:** High
- **Docs URL:** https://react.dev/learn/you-might-not-need-an-effect

### Result 7
- **Category:** Forms
- **Guideline:** Debounce rapid input changes
- **Description:** Debounce search/filter inputs
- **Do:** useDeferredValue or debounce for search
- **Don't:** Filter on every keystroke
- **Code Good:** useDeferredValue(searchTerm)
- **Code Bad:** useEffect filtering on every change
- **Severity:** Medium
- **Docs URL:** https://react.dev/reference/react/useDeferredValue

### Result 8
- **Category:** Accessibility
- **Guideline:** Label form controls
- **Description:** Associate labels with inputs
- **Do:** htmlFor matching input id
- **Don't:** Placeholder as only label
- **Code Good:** <label htmlFor="email">Email</label>
- **Code Bad:** <input placeholder="Email"/>
- **Severity:** High
- **Docs URL:** 

### Result 9
- **Category:** Forms
- **Guideline:** Handle form submission properly
- **Description:** Prevent default and handle in submit handler
- **Do:** onSubmit with preventDefault
- **Don't:** onClick on submit button only
- **Code Good:** <form onSubmit={handleSubmit}>
- **Code Bad:** <button onClick={handleSubmit}>
- **Severity:** Medium
- **Docs URL:** 

### Result 10
- **Category:** Accessibility
- **Guideline:** Manage focus properly
- **Description:** Handle focus for modals dialogs
- **Do:** Focus trap in modals return focus on close
- **Don't:** No focus management
- **Code Good:** useEffect to focus input
- **Code Bad:** Modal without focus trap
- **Severity:** High
- **Docs URL:** 

### Result 11
- **Category:** Accessibility
- **Guideline:** Announce dynamic content
- **Description:** Use ARIA live regions for updates
- **Do:** aria-live for dynamic updates
- **Don't:** Silent updates to screen readers
- **Code Good:** <div aria-live="polite">{msg}</div>
- **Code Bad:** <div>{msg}</div>
- **Severity:** Medium
- **Docs URL:** 

### Result 12
- **Category:** Accessibility
- **Guideline:** Use semantic HTML
- **Description:** Proper HTML elements for their purpose
- **Do:** button for clicks nav for navigation
- **Don't:** div with onClick for buttons
- **Code Good:** <button onClick={...}>
- **Code Bad:** <div onClick={...}>
- **Severity:** High
- **Docs URL:** https://react.dev/reference/react-dom/components#all-html-components

