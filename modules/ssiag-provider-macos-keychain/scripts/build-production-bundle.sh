#!/bin/sh
set -eu

usage() {
  echo "usage: build-production-bundle.sh --output DIR --bundle-id ID --signing-identity IDENTITY --policy FILE --notary-profile PROFILE [--entitlements FILE] [--provisioning-profile FILE]" >&2
  exit 64
}

output=
bundle_id=
signing_identity=
policy=
notary_profile=
entitlements=
provisioning_profile=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output) [ "$#" -ge 2 ] || usage; output=$2; shift 2 ;;
    --bundle-id) [ "$#" -ge 2 ] || usage; bundle_id=$2; shift 2 ;;
    --signing-identity) [ "$#" -ge 2 ] || usage; signing_identity=$2; shift 2 ;;
    --policy) [ "$#" -ge 2 ] || usage; policy=$2; shift 2 ;;
    --notary-profile) [ "$#" -ge 2 ] || usage; notary_profile=$2; shift 2 ;;
    --entitlements) [ "$#" -ge 2 ] || usage; entitlements=$2; shift 2 ;;
    --provisioning-profile) [ "$#" -ge 2 ] || usage; provisioning_profile=$2; shift 2 ;;
    *) usage ;;
  esac
done

[ -n "$output" ] && [ -n "$bundle_id" ] && [ -n "$signing_identity" ] && [ -n "$policy" ] && [ -n "$notary_profile" ] || usage
case "$output" in /*) ;; *) echo "--output must be absolute" >&2; exit 65 ;; esac
echo "$bundle_id" | grep -Eq '^[A-Za-z0-9][A-Za-z0-9.-]{2,254}$' || { echo "invalid bundle identifier" >&2; exit 65; }
[ -f "$policy" ] && /usr/bin/plutil -lint "$policy" >/dev/null
[ -z "$entitlements" ] || [ -f "$entitlements" ] || { echo "entitlements file is unavailable" >&2; exit 66; }
[ -z "$provisioning_profile" ] || [ -f "$provisioning_profile" ] || { echo "provisioning profile is unavailable" >&2; exit 66; }
[ ! -e "$output/SymphonySSIAGMacOSKeychainProvider.app" ] || { echo "output bundle already exists" >&2; exit 73; }

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
module_dir=$(dirname -- "$script_dir")
cd "$module_dir"
/usr/bin/swift build -c release

stage=$(mktemp -d "${TMPDIR:-/tmp}/symphony-ssiag-provider-bundle.XXXXXX")
trap 'rm -rf "$stage"' EXIT HUP INT TERM
bundle="$stage/SymphonySSIAGMacOSKeychainProvider.app"
mkdir -p "$bundle/Contents/MacOS" "$bundle/Contents/Resources"
cp ".build/release/symphony-ssiag-provider-macos-keychain" "$bundle/Contents/MacOS/symphony-ssiag-provider-macos-keychain"
chmod 500 "$bundle/Contents/MacOS/symphony-ssiag-provider-macos-keychain"
cp "$policy" "$bundle/Contents/Resources/ssiag-signing-policy.json"
chmod 400 "$bundle/Contents/Resources/ssiag-signing-policy.json"
if [ -n "$provisioning_profile" ]; then
  cp "$provisioning_profile" "$bundle/Contents/embedded.provisionprofile"
  chmod 400 "$bundle/Contents/embedded.provisionprofile"
fi

cat >"$bundle/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleExecutable</key><string>symphony-ssiag-provider-macos-keychain</string>
  <key>CFBundleIdentifier</key><string>$bundle_id</string>
  <key>CFBundleName</key><string>Symphony SSIAG macOS Keychain Provider</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>0.1.0</string>
  <key>CFBundleVersion</key><string>1</string>
  <key>LSBackgroundOnly</key><true/>
  <key>LSMinimumSystemVersion</key><string>13.0</string>
</dict></plist>
EOF
chmod 400 "$bundle/Contents/Info.plist"
/usr/bin/plutil -lint "$bundle/Contents/Info.plist" >/dev/null

if [ -n "$entitlements" ]; then
  /usr/bin/codesign --force --sign "$signing_identity" --options runtime --timestamp --entitlements "$entitlements" "$bundle"
else
  /usr/bin/codesign --force --sign "$signing_identity" --options runtime --timestamp "$bundle"
fi
/usr/bin/codesign --verify --strict --deep --all-architectures "$bundle"

archive="$stage/SymphonySSIAGMacOSKeychainProvider.zip"
/usr/bin/ditto -c -k --keepParent "$bundle" "$archive"
/usr/bin/xcrun notarytool submit "$archive" --keychain-profile "$notary_profile" --wait
/usr/bin/xcrun stapler staple "$bundle"
/usr/bin/xcrun stapler validate "$bundle"

mkdir -p "$output"
mv "$bundle" "$output/SymphonySSIAGMacOSKeychainProvider.app"
trap - EXIT HUP INT TERM
rm -rf "$stage"
echo "$output/SymphonySSIAGMacOSKeychainProvider.app"
