# ui-ux-pro-max output
## Design-system query

Query: `admin dictionary management master detail grouped navigation editable data table`

- Recommended a data-dense dashboard with strong contrast, restrained blue selection, explicit hover feedback, and visible focus.
- Required checks include keyboard access, reduced motion, and responsive behavior at 375/768/1024/1440 px.

## Hierarchical navigation query

Query: `hierarchical navigation grouped list collapsible keyboard accessibility`

- All functionality must remain keyboard accessible and tab order must match visual order.
- Interactive category and type controls need visible focus rings.
- Heading hierarchy and browser navigation state should remain predictable.

## Editable table query

Query: `editable table drawer form disabled immutable key tenant override`

- Wide tables need a contained horizontal-scroll region on mobile.
- Immutable fields require a clear disabled state and a text explanation.
- Submit operations require loading plus success/error feedback, and every control needs a real label.

## React query

Query: `data table actions responsive horizontal scroll`

- Derive grouped presentation data during render instead of syncing duplicated state in effects.
- Keep frequently changing form values local to the existing RHF-controlled drawer.
