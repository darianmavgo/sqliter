#!/bin/bash
THEMES="cerulean cosmo cyborg darkly flatly journal litera lumen lux materia minty morph pulse quartz sandstone simplex sketchy slate solar spacelab superhero united vapor yeti zephyr"
BASE_URL="https://cdn.jsdelivr.net/npm/bootswatch@5.3.2/dist"

for theme in $THEMES; do
    echo "Downloading $theme..."
    curl -sL "$BASE_URL/$theme/bootstrap.min.css" -o "themes/$theme.css"
done
