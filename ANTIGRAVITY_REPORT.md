# Bug Report: HTML Template Corruption via Autocorrect

**Date:** 2026-01-28
**Reporter:** Antigravity (on behalf of User)
**Component:** Tooling / Editor Integration

## Issue Description
During the development of the `sqliter` Go application, specifically when editing `sqliter/templates/head.html`, an automated formatting or "autocorrect" process corrupted standard Go template syntax.

## Symptoms
The Go template action for injecting CSS:
```html
<style>
  {{.CSS}}
</style>
```

Was repeatedly reformatted into invalid syntax:
```html
<style>
  {
      {
      .CSS
    }
  }
</style>
```

## Impact
- **Build/Test Failures:** The corrupted syntax caused `go test` failures in `TestStartHTMLTable` because the output HTML did not match expected patterns (CSS was not injected).
- **Development Friction:** The user and agent had to repeatedly attempt to fix the file, only for it to be reverted/corrupted again.
- **Workflow Interruption:** Required manual intervention (deleting and recreating the file) to resolve.

## Recommendations
1. **Disable Auto-Formatting for HTML Templates:** Ensure that generic HTML formatters do not run on `.html` files that contain Go template syntax (`{{ ... }}`), or configure them to respect that syntax.
2. **Tooling Review:** Investigate if the `replace_file_content` tool or the underlying editor environment triggers an aggressive formatter on save/write.
