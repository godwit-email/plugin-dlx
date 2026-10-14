#!/usr/bin/env python3
"""Build the current platform's library and a registry release fragment."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import zipfile

root = Path(__file__).resolve().parent.parent
os.chdir(root)
manifest = json.loads((root / "plugin.json").read_text(encoding="utf-8"))
goos, goarch = subprocess.check_output(["go", "env", "GOOS", "GOARCH"], text=True).split()
extension = {"darwin": "dylib", "linux": "so", "windows": "dll"}[goos]
platform = f"{goos}-{goarch}"
dest = root / "dist" / platform
dest.mkdir(parents=True, exist_ok=True)
library = dest / f"plugin.{extension}"
subprocess.run(["go", "build", "-trimpath", "-buildmode=c-shared", "-ldflags=-s -w", "-o", str(library), "."], check=True)
data = library.read_bytes()
assert 0 < len(data) <= 64 << 20, "Library exceeds Godwit's native artifact limit"
asset_name = f"dlx-{platform}.{extension}"
shutil.copyfile(library, dest / asset_name)
for name in ["plugin.json", "LICENSE", "THIRD_PARTY_NOTICES"]:
    shutil.copyfile(root / name, dest / name)
module_dirs = subprocess.check_output(["go", "list", "-deps", "-f", "{{if .Module}}{{.Module.Dir}}{{end}}", "."], text=True).splitlines()
notices = []
for directory in sorted(set(module_dirs) - {"", str(root)}):
    module = Path(directory)
    for name in ["LICENSE", "LICENSE.txt", "LICENSE.md", "COPYING", "NOTICE"]:
        license_path = module / name
        if license_path.is_file():
            notices.append(f"=== {module.name} / {name} ===\n{license_path.read_text(encoding='utf-8')}\n")
(dest / "DEPENDENCY_LICENSES.txt").write_text("\n".join(notices), encoding="utf-8")
go_root = Path(subprocess.check_output(["go", "env", "GOROOT"], text=True).strip())
with (dest / "DEPENDENCY_LICENSES.txt").open("a", encoding="utf-8") as stream:
    stream.write("\n=== Go standard library and runtime ===\n" + (go_root / "LICENSE").read_text(encoding="utf-8"))
with zipfile.ZipFile(dest / f"dlx-{platform}.zip", "w", compression=zipfile.ZIP_DEFLATED) as bundle:
    for name in [library.name, "plugin.json", "LICENSE", "THIRD_PARTY_NOTICES", "DEPENDENCY_LICENSES.txt"]:
        bundle.write(dest / name, name)
artifact = {
    "url": f"https://github.com/godwit-email/plugin-dlx/releases/download/v{manifest['version']}/{asset_name}",
    "sha256": hashlib.sha256(data).hexdigest(),
    "size": len(data),
}
(dest / "registry-release.json").write_text(json.dumps({
    "version": manifest["version"], "manifest": manifest, "artifacts": {platform: artifact}
}, indent=2) + "\n", encoding="utf-8")
(dest / "SHA256SUMS").write_text(f"{artifact['sha256']}  {asset_name}\n", encoding="utf-8")
print(json.dumps({"platform": platform, **artifact}, indent=2))
