# Decisions

1. `apps/ak-admin/public/brand` is the only source for the site mark, favicon,
   Apple Touch Icon, and manifest icons. The former documentation-only SVG mark
   is removed.
2. The wide documentation shell is exactly 1488 px: 280 px sidebar, 960 px
   content, and 248 px outline. At 1920 px this yields equal 216 px outer
   margins. The middle column receives 56 px padding while prose is capped at
   72ch.
3. The Hero receives local `heroShowcase` frontmatter rather than using the
   Rspress feature/image schema. The theme type requires one `admin` object and
   a two-item `mobile` tuple; every object contains `src` and `alt`.
4. The Admin screenshot comes from the loaded Dashboard of an isolated local
   Docker acceptance environment using a synthetic `.example.test` account.
   No credential, token, production record, or personal data appears in the
   image.
5. Mobile images were freshly captured from an iPhone 16 Pro / iOS 18.6
   simulator after an iOS compile and local API sign-in. This is simulator
   evidence only and is not Android, HarmonyOS, or physical-device acceptance.
6. Homepage value cards use semantic anchors with one border and one top marker.
   Rspress `features`, nested frames, tilt, and shine are not used.
7. No API or Mobile component contract is expanded beyond repository evidence.
   New entry-page material explains navigation, responsibilities, first-use
   flow, and contribution paths.
8. No generated image is used in this iteration: authentic product surfaces
   satisfy the visual need. The social image is regenerated from the final
   homepage rather than reusing the old ecosystem illustration.
