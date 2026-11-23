#!/usr/bin/env bash
# Runs the centrifuge library's integration tests inside the `tests` service of
# compose.isolated.yml (condition C4: exact version and commit, results kept
# apart from the server's). Output: /out/library.json (go test -json),
# /out/version.txt.
set -euo pipefail

mod=github.com/centrifugal/centrifuge
src=/hostmod/$mod@$LIBRARY_VERSION
info=/hostmod/cache/download/$mod/@v/$LIBRARY_VERSION.info

# The module cache records the commit the pseudo-version was fetched from.
grep -q "\"Hash\":\"$LIBRARY_COMMIT\"" "$info" || {
	echo "centrifuge $LIBRARY_VERSION: $info does not name commit $LIBRARY_COMMIT" >&2
	exit 1
}
# That the extracted directory matches its zip hash is checked on the host
# (`go mod verify` in make library-test).
rm -rf /work && mkdir /work && cp -R "$src" /work/centrifuge && chmod -R u+w /work/centrifuge
cd /work/centrifuge

{
	echo "module $mod $LIBRARY_VERSION"
	echo "commit $LIBRARY_COMMIT"
	echo "ziphash $(cat "/hostmod/cache/download/$mod/@v/$LIBRARY_VERSION.ziphash")"
	go version
} | tee /out/version.txt

# DragonflyDB subtests (df_*, dragonfly_*) are excluded (decision D2-3): no DragonflyDB runs.
run() { go test -count=1 -race -tags integration -parallel 2 -timeout 45m -json -skip '/(df_|dragonfly)' "$@" >> /out/library.json || true; }
: > /out/library.json
package_list=$(go list ./...)
mapfile -t packages < <(printf '%s\n' "$package_list" | grep -v /_examples/ | grep -vx "$mod")
run "${packages[@]}"

# The root package runs in $shards processes (top-level tests, fuzz seeds and
# examples dealt round-robin in source order): in one -race process its memory
# grew to 6.6 GB. Four shards also produced a killed test process in the
# isolated container. Smaller shards and fewer parallel tests bound memory.
shards=12
listed=$(go test -tags integration -list '.*' .)
mapfile -t names < <(printf '%s\n' "$listed" | grep -E '^(Test|Fuzz|Example)')
if (( ${#names[@]} < shards )); then
	echo "expected at least $shards root-package tests" >&2
	exit 1
fi
for ((i = 0; i < shards; i++)); do
	sel=()
	for ((j = i; j < ${#names[@]}; j += shards)); do sel+=("${names[j]}"); done
	run -run "^($(IFS='|'; echo "${sel[*]}"))\$" .
done
echo "done: /out/library.json (${#names[@]} root-package tests in $shards shards)"
