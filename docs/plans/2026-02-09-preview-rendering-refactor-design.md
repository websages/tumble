# Preview Rendering Refactor: Provider Config Table

**Date:** 2026-02-09
**Status:** Proposed
**Branch:** tumble-dark

## Problem

The OG preview / oEmbed rendering logic in `index.html` is a ~170-line
monolithic function (`renderPreview()`) with provider-specific behavior
scattered as inline `if` checks throughout. Every provider's special
behavior (skip title for GIPHY, auto-expand for Reddit, clickable image
for Flickr, etc.) is interleaved with the generic card-building logic.

This causes regressions roughly every 3-4 features. Changing one
provider's behavior requires editing inside the monolith, and conditions
interact in ways that are hard to see.

## Solution

Replace scattered provider checks with a declarative configuration
table. The rendering function reads from config instead of branching
on provider names.

### Provider Config Table

```js
var providerConfig = {
  "Reddit":   {
    showTitle: true,
    showDescription: true,
    autoExpand: true,
    defaultIcon: "https://www.redditstatic.com/desktop2x/img/favicon/favicon-32x32.png"
  },
  "Spotify":  { showTitle: false, showDescription: false, autoExpand: true },
  "GIPHY":    { showTitle: false, showDescription: false },
  "Flickr":   { clickableImage: true, forceAspect: "photo" },
  "_default": {
    showTitle: true,
    showDescription: true,
    autoExpand: false,
    clickableImage: false
  }
};
```

At the top of `renderPreview()`, config is resolved:

```js
var cfg = providerConfig[provider] || {};
var defaults = providerConfig["_default"];
for (var key in defaults) {
    if (cfg[key] === undefined) cfg[key] = defaults[key];
}
```

### How Rendering Logic Changes

Before (scattered exclusion checks):

```js
if (title && provider !== "GIPHY" && provider !== "Spotify") { ... }
if ((provider === "Reddit" || provider === "Spotify") && data.embed_html) { ... }
```

After (config-driven):

```js
if (title && cfg.showTitle) { ... }
if (cfg.autoExpand && data.embed_html) { ... }
```

### URL-Based Transforms

Two special cases are URL-based rather than provider-name-based.
These get extracted into named helper functions:

**Imgur animated swap** - currently inline in the image-building section:

```js
function transformImgurAnimated(url, imageUrl) {
    if (url.match(/imgur\.com/) && imageUrl.match(/i\.imgur\.com.*\.jpg/)) {
        return { url: imageUrl.replace(/\.jpg.*$/, '.mp4'), isAnimated: true };
    }
    return { url: imageUrl, isAnimated: false };
}
```

**TikTok OEmbed fallback** - stays in `handlePreviewData()` as a
pre-processing step before `renderPreview()` is called. Already
naturally separated; just gets a comment marking it as a URL-based
transform.

If more URL-based transforms are added in the future (4+), these
can be moved to a table-driven pattern similar to `providerConfig`.

## Scope

### What changes

- Add `providerConfig` table at top of preview JS block
- Extract `transformImgurAnimated()` helper function
- Rewrite `renderPreview()` to read from config instead of branching
- Minor cleanup of `handlePreviewData()` comments

### What does NOT change

- Imgur gallery card early-return (different rendering mode)
- `replaceWithVideo()` (already standalone)
- `escapeHtml()`, `buildLinkUrl()`
- Sticky date widget, vim navigation, theme toggling
- File location (stays inline in `index.html`)
- Go code, CSS, other templates

## Provider Behavior Matrix

All behaviors preserved exactly as-is:

| Provider | Title | Description | Auto-expand | Special |
|----------|-------|-------------|-------------|---------|
| YouTube | yes | yes | no | play button overlay |
| Reddit | yes | yes | yes | default favicon, auto-expand |
| Spotify | no | no | yes | auto-expand iframe |
| GIPHY | no | no | no | -- |
| Flickr | yes | yes | no | clickable image, photo aspect |
| Imgur | yes | yes | no | .jpg to .mp4 animated swap |
| TikTok | yes | yes | no | client-side OEmbed fallback |
| Default | yes | yes | no | -- |

## Functional Changes

None. This is a pure refactoring. Behavior is identical before and
after.

## Future Work

- Add Playwright UI tests using mocked `/ogpreview` responses
- One test per provider verifying the DOM structure matches the
  behavior matrix above
- Visual regression tests (screenshot comparison) for CSS layout
