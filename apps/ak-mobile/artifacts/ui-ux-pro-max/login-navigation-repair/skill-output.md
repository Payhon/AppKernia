# ui-ux-pro-max output

Executed before implementation:

```text
python3 .codex/skills/ui-ux-pro-max/scripts/search.py
  "mobile app authentication login password recovery registration legal navigation minimal professional"
  --design-system -p "AppKernia Mobile Auth Navigation" -f markdown

python3 .codex/skills/ui-ux-pro-max/scripts/search.py
  "mobile authentication secondary text links touch target spacing back navigation accessibility"
  --domain ux -n 12

python3 .codex/skills/ui-ux-pro-max/scripts/search.py
  "authentication form secondary links native back navigation"
  --stack react-native
```

Relevant returned guidance:

- adjacent mobile touch targets need at least an 8 px gap;
- touch targets should be at least 44 × 44 px;
- preserving predictable navigation history is high severity;
- secondary authentication actions should not compete visually with the primary CTA;
- light-mode text contrast must remain at least 4.5:1.

The generic design-system search also suggested marketing/landing-page treatments and external fonts. Those results were rejected because the repository Master and auth/legal override require minimal product UI, system typography and no marketing artwork.
