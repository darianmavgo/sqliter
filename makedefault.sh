#!/bin/bash

# Configuration
APP_NAME="SQLiter"
APP_PATH="/Applications/$APP_NAME.app"
BINARY_PATH="/usr/local/bin/sqliter"
BUNDLE_ID="com.mavgo.sqliter"

echo "�️  Building $APP_NAME.app..."

# 1. Create AppleScript wrapper to handle file opening
# This script handles both opening the app directly and dragging files onto it (or "Open With")
mkdir -p build_tmp
cat > build_tmp/launcher.applescript <<EOF
on open theFiles
    repeat with aFile in theFiles
        set posixPath to POSIX path of aFile
        -- Run the sqliter binary with the file path in background
        do shell script "$BINARY_PATH " & quoted form of posixPath & " > /dev/null 2>&1 &"
    end repeat
end open

on run
    -- Run sqliter without arguments if just opened
    do shell script "$BINARY_PATH > /dev/null 2>&1 &"
end run
EOF

# 2. Compile to .app
# -o overwrites existing bundle
osacompile -o "$APP_PATH" build_tmp/launcher.applescript
rm -rf build_tmp

echo "📝 Updating Info.plist..."
PLIST="$APP_PATH/Contents/Info.plist"

# 3. Patch Info.plist
# Set the Bundle Identifier explicitly so Launch Services recognizes it
plutil -replace CFBundleIdentifier -string "$BUNDLE_ID" "$PLIST"

# Add Document Types to claim ownership of .sqlite and .db files
plutil -replace CFBundleDocumentTypes -xml '
<array>
    <dict>
        <key>CFBundleTypeName</key>
        <string>SQLite Database</string>
        <key>CFBundleTypeRole</key>
        <string>Editor</string>
        <key>LSHandlerRank</key>
        <string>Owner</string>
        <key>LSItemContentTypes</key>
        <array>
            <string>org.sqlite.sqlite3</string>
            <string>com.sqlite.db</string>
            <string>public.database</string>
            <string>io.sqlite.db</string>
            <string>dyn.ah62d4rv4ge80q650</string> 
        </array>
        <key>CFBundleTypeExtensions</key>
        <array>
            <string>sqlite</string>
            <string>db</string>
            <string>sqlite3</string>
        </array>
    </dict>
</array>
' "$PLIST"


# 4. Register with Launch Services
echo "🚀 Registering with Launch Services..."
LSREGISTER="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
# Fallback path for different macOS versions
if [ ! -f "$LSREGISTER" ]; then
    LSREGISTER="/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/LaunchServices.framework/Versions/A/Support/lsregister"
fi

if [ -f "$LSREGISTER" ]; then
    # -f forces re-registration
    "$LSREGISTER" -f "$APP_PATH"
else
    echo "⚠️ lsregister not found. System may not pick up changes immediately."
fi

# 5. Set as Default Handler
echo "🔒 Setting $APP_NAME as default handler..."

# Use Swift code to set the default handler programmatically
swift - <<EOF
import Foundation
import CoreServices

let bundleId = "$BUNDLE_ID" as CFString
// List of UTIs to claim
let utis = [
    "org.sqlite.sqlite3", 
    "com.sqlite.db", 
    "public.database", 
    "io.sqlite.db"
]

print("Setting default handler for Bundle ID: $BUNDLE_ID")

for uti in utis {
    let utiString = uti as CFString
    let result = LSSetDefaultRoleHandlerForContentType(utiString, .editor, bundleId)
    
    if result == 0 {
         print("  ✅ Set default for \(uti)")
    } else {
         print("  ⚠️  Result for \(uti): \(result)") 
    }
}
print("Done.")
EOF

echo "✅ Setup complete. 'SQLiter' in Applications is now the handler."
