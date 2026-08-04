# AKADM-060 Avatar Skill Output

三条项目本地 Skill 命令均真实执行并退出 0。

## Design system

## Design System: AppKernia Admin Profile Avatar

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
Clear focus rings (3-4px), ARIA labels, skip links, responsive design, reduced motion, 44x44px touch targets

### Avoid (Anti-patterns)
- Confusing layout
- Privacy concerns
- AI purple/pink gradients

### Pre-Delivery Checklist
- [ ] No emojis as icons (use SVG: Heroicons/Lucide)
- [ ] cursor-pointer on all clickable elements
- [ ] Hover states with smooth transitions (150-300ms)
- [ ] Light mode: text contrast 4.5:1 minimum
- [ ] Focus states visible for keyboard nav
- [ ] prefers-reduced-motion respected
- [ ] Responsive: 375px, 768px, 1024px, 1440px



## UX search

## UI Pro Max Search Results
**Domain:** ux | **Query:** avatar upload image crop file validation progress retry accessibility privacy
**Source:** ux-guidelines.csv | **Found:** 12 results

### Result 1
- **Category:** Feedback
- **Issue:** Progress Indicators
- **Platform:** All
- **Description:** Show progress for multi-step processes
- **Do:** Step indicators or progress bar
- **Don't:** No indication of progress
- **Code Example Good:** Step 2 of 4 indicator
- **Code Example Bad:** No step information
- **Severity:** Medium

### Result 2
- **Category:** Forms
- **Issue:** Inline Validation
- **Platform:** All
- **Description:** Validate as user types or on blur
- **Do:** Validate on blur for most fields
- **Don't:** Validate only on submit
- **Code Example Good:** onBlur validation
- **Code Example Bad:** Submit-only validation
- **Severity:** Medium

### Result 3
- **Category:** Performance
- **Issue:** Image Optimization
- **Platform:** All
- **Description:** Large images slow page load
- **Do:** Use appropriate size and format (WebP)
- **Don't:** Unoptimized full-size images
- **Code Example Good:** srcset with multiple sizes
- **Code Example Bad:** 4000px image for 400px display
- **Severity:** High

### Result 4
- **Category:** Responsive
- **Issue:** Image Scaling
- **Platform:** Web
- **Description:** Images should scale with container
- **Do:** Use max-width: 100% on images
- **Don't:** Fixed width images overflow
- **Code Example Good:** max-w-full h-auto
- **Code Example Bad:** width='800' fixed
- **Severity:** Medium

### Result 5
- **Category:** Sustainability
- **Issue:** Asset Weight
- **Platform:** Web
- **Description:** Heavy 3D/Image assets increase carbon footprint
- **Do:** Compress and lazy load 3D models
- **Don't:** Load 50MB textures
- **Code Example Good:** Draco compression
- **Code Example Bad:** Raw .obj files
- **Severity:** Medium

### Result 6
- **Category:** Accessibility
- **Issue:** Alt Text
- **Platform:** All
- **Description:** Images need text alternatives
- **Do:** Descriptive alt text for meaningful images
- **Don't:** Empty or missing alt attributes
- **Code Example Good:** alt='Dog playing in park'
- **Code Example Bad:** alt='' for content images
- **Severity:** High

### Result 7
- **Category:** Accessibility
- **Issue:** Error Messages
- **Platform:** All
- **Description:** Error messages must be announced
- **Do:** Use aria-live or role=alert for errors
- **Don't:** Visual-only error indication
- **Code Example Good:** role='alert'
- **Code Example Bad:** Red border only
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



## React stack

## UI Pro Max Stack Guidelines
**Stack:** react | **Query:** profile avatar upload progress error responsive accessibility react hook form
**Source:** stacks/react.csv | **Found:** 3 results

### Result 1
- **Category:** Performance
- **Guideline:** Use React DevTools Profiler
- **Description:** Profile to identify performance bottlenecks
- **Do:** Profile before optimizing
- **Don't:** Optimize without measuring
- **Code Good:** React DevTools Profiler
- **Code Bad:** Guessing at bottlenecks
- **Severity:** Medium
- **Docs URL:** https://react.dev/learn/react-developer-tools

### Result 2
- **Category:** Accessibility
- **Guideline:** Label form controls
- **Description:** Associate labels with inputs
- **Do:** htmlFor matching input id
- **Don't:** Placeholder as only label
- **Code Good:** <label htmlFor="email">Email</label>
- **Code Bad:** <input placeholder="Email"/>
- **Severity:** High
- **Docs URL:** 

### Result 3
- **Category:** State
- **Guideline:** Use useState for local state
- **Description:** Simple component state should use useState hook
- **Do:** useState for form inputs toggles counters
- **Don't:** Class components this.state
- **Code Good:** const [count, setCount] = useState(0)
- **Code Bad:** this.state = { count: 0 }
- **Severity:** Medium
- **Docs URL:** https://react.dev/reference/react/useState


