#!/bin/bash
THEME=$1
if [ -z "$THEME" ]; then
    echo "Usage: $0 <theme_name>"
    echo "Available themes:"
    ls themes/*.css | grep -v overrides.css | grep -v stylesheet.css | xargs -n 1 basename | sed 's/.css//'
    exit 1
fi

if [ ! -f "themes/$THEME.css" ]; then
    echo "Theme '$THEME' not found."
    exit 1
fi

cat "themes/$THEME.css" "themes/overrides.css" > "themes/cssjs/default.css"
echo "Swapped to theme: $THEME"
