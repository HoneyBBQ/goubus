#!/usr/bin/env bash
# Print the GitHub release body for an exact CHANGELOG.md tag.
# Usage: scripts/release-notes.sh v2.0.0-alpha2 > release-notes.md
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <tag>" >&2
  exit 2
fi

tag="$1"
repo="HoneyBBQ/goubus"
module="github.com/honeybbq/goubus/v2"
cd "$(dirname "$0")/.."
changelog="CHANGELOG.md"

# Take the matching "## <tag>" section, excluding the heading itself.
section="$(awk -v tag="$tag" '
  /^## / {
    if (found) exit
    title = substr($0, 4)
    if (title == tag || index(title, tag " ") == 1) { found = 1; next }
  }
  found { print }
' "$changelog")"

section="$(printf '%s\n' "$section" | sed -e '/./,$!d' | sed -e ':a' -e '/^\n*$/{$d;N;ba' -e '}')"

if [ -z "$section" ]; then
  echo "release-notes: $changelog has no section for $tag (or it is empty); add a \"## $tag (date)\" heading" >&2
  exit 1
fi

# Convert links relative to CHANGELOG.md to tag-pinned GitHub links, but leave
# literal examples in fenced code blocks and inline code spans alone.
base="https://github.com/$repo/blob/$tag"
# shellcheck disable=SC2016 # backticks inside single-quoted awk program are literal
section="$(printf '%s\n' "$section" | awk -v base="$base" -v changelog="$changelog" '
  function rewrite_plain(segment, output, offset, start, rest, link_end, target, relative) {
    output = ""
    offset = 1
    while ((start = index(substr(segment, offset), "](")) > 0) {
      start += offset - 1
      output = output substr(segment, offset, start - offset + 2)
      rest = substr(segment, start + 2)
      link_end = index(rest, ")")
      if (link_end == 0) { output = output rest; return output }
      target = substr(rest, 1, link_end - 1)
      relative = target
      if (substr(target, 1, 1) == "#") {
        target = base "/" changelog target
      } else {
        if (substr(relative, 1, 2) == "./") relative = substr(relative, 3)
        if (relative !~ /^[\/:#]/ && relative !~ /:/) target = base "/" relative
      }
      output = output target ")"
      offset = start + link_end + 2
    }
    return output substr(segment, offset)
  }
  function rewrite_links(line, out, pos, tick, tick_start, ticks, search_pos, close_rel, close_pos, span_end) {
    out = ""
    pos = 1
    while ((tick = index(substr(line, pos), "`")) > 0) {
      tick_start = pos + tick - 1
      out = out rewrite_plain(substr(line, pos, tick_start - pos))
      ticks = "`"
      while (substr(line, tick_start + length(ticks), 1) == "`") {
        ticks = ticks "`"
      }
      search_pos = tick_start + length(ticks)
      close_pos = 0
      while ((close_rel = index(substr(line, search_pos), ticks)) > 0) {
        close_pos = search_pos + close_rel - 1
        if (substr(line, close_pos + length(ticks), 1) != "`") break
        while (substr(line, close_pos, 1) == "`") close_pos++
        search_pos = close_pos
        close_pos = 0
      }
      if (close_pos == 0) {
        out = out ticks
        pos = tick_start + length(ticks)
      } else {
        span_end = close_pos + length(ticks)
        out = out substr(line, tick_start, span_end - tick_start)
        pos = span_end
      }
    }
    return out rewrite_plain(substr(line, pos))
  }
  /^[ \t]*```/ { print; code = !code; next }
  code { print; next }
  { print rewrite_links($0) }
')"

# Unwrap paragraphs and list items, keeping code blocks and table rows intact.
section="$(printf '%s\n' "$section" | awk '
  function flush() { if (buf != "") print buf; buf = "" }
  /^[ \t]*```/ { flush(); print; code = !code; next }
  code { print; next }
  /^[ \t]*$/ { flush(); print; next }
  /^#/ { flush(); print; next }
  /^[ \t]*([-*+] |[0-9]+\. )/ || /^[ \t]*\|/ || /^>/ { flush(); buf = $0; next }
  buf == "" { buf = $0; next }
  { sub(/^[ \t]+/, ""); buf = buf " " $0 }
  END { flush() }
')"

previous="$(git -c versionsort.suffix=- tag --list 'v*' --sort=-v:refname |
  awk -v tag="$tag" 'found { print; exit } $0 == tag { found = 1 }')"

# shellcheck disable=SC2016 # the backticks are Markdown
printf 'Install:\n\n```sh\ngo get %s@%s\n```\n\n' "$module" "$tag"
printf '%s\n' "$section"
if [ -n "$previous" ]; then
  printf '\n**Full changelog**: https://github.com/%s/compare/%s...%s\n' "$repo" "$previous" "$tag"
fi
