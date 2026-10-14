#!/usr/bin/env python3
"""Exercise ABI v1 through the pinned Godwit plugin-host. Network is opt-in."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys

parser = argparse.ArgumentParser()
parser.add_argument("library", type=Path)
parser.add_argument("--live", action="store_true")
args = parser.parse_args()
root = Path(__file__).resolve().parent.parent
library = args.library.resolve()
digest = hashlib.sha256(library.read_bytes()).hexdigest()
host_dir = root / "tools" / "host"
host = root / "tmp" / ("plugin-host.exe" if sys.platform == "win32" else "plugin-host")
host.parent.mkdir(parents=True, exist_ok=True)
subprocess.run(["go", "build", "-o", str(host), "."], cwd=host_dir, check=True)
requests = [
    ("describe", {}, True),
    ("configure", {"config": {}}, True),
    ("translate", {"texts": ["", " \n"], "source": "", "target": "zh-CN"}, True),
    ("translate", {"texts": ["hello"], "source": "", "target": "xx"}, False),
    ("missing", {}, False),
]
if args.live:
    requests.append(("translate", {"texts": ["Hello, world!"], "source": "en", "target": "zh-CN"}, True))
payload = "".join(json.dumps({"id": index, "method": method, "params": params}) + "\n" for index, (method, params, _) in enumerate(requests, 1))
proc = subprocess.run([str(host), "--library", str(library), "--sha256", digest, "--abi", "1"], input=payload, capture_output=True, text=True, timeout=90)
assert proc.returncode == 0, proc.stderr
if proc.stderr:
    print(proc.stderr, file=sys.stderr)
responses = sorted([json.loads(line) for line in proc.stdout.splitlines()], key=lambda item: item["id"])
assert len(responses) == len(requests), proc.stdout
for index, ((method, params, ok), response) in enumerate(zip(requests, responses), 1):
    assert response.get("id") == index and response.get("ok") is ok, response
    if method == "describe":
        manifest = json.loads((root / "plugin.json").read_text(encoding="utf-8"))
        assert response["result"] == {"name": manifest["name"], "version": manifest["version"], "abi": manifest["native"]["abi"], "capabilities": manifest["capabilities"]}, response
    if method == "translate" and ok:
        assert len(response["result"]["texts"]) == len(params["texts"]), response
        if params["texts"][0] == "":
            assert response["result"]["texts"] == params["texts"], response
        else:
            assert response["result"]["texts"][0] != params["texts"][0], response
    print(json.dumps(response, ensure_ascii=False))
denied = subprocess.run([str(host), "--library", str(library), "--sha256", "0" * 64], capture_output=True, text=True, timeout=30)
assert denied.returncode == 3 and "DENIED_BEFORE_DLOPEN" in denied.stderr, denied
incompatible = subprocess.run([str(host), "--library", str(library), "--sha256", digest, "--abi", "2"], capture_output=True, text=True, timeout=30)
assert incompatible.returncode == 4 and "LOAD_FAILED" in incompatible.stderr, incompatible
print("Godwit ABI smoke test passed (including SHA-256 and incompatible ABI rejection).")
