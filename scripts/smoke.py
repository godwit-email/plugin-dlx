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
with subprocess.Popen([str(host), "--library", str(library), "--sha256", digest, "--abi", "1"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True) as proc:
    for index, (method, params, ok) in enumerate(requests, 1):
        proc.stdin.write(json.dumps({"id": index, "method": method, "params": params}) + "\n")
        proc.stdin.flush()
        line = proc.stdout.readline()
        assert line, f"plugin-host stopped at {method}"
        response = json.loads(line)
        assert response.get("id") == index and response.get("ok") is ok, response
        if method == "describe":
            manifest = json.loads((root / "plugin.json").read_text())
            assert response["result"] == {"name": manifest["name"], "version": manifest["version"], "abi": manifest["native"]["abi"], "capabilities": manifest["capabilities"]}, response
        if method == "translate" and ok:
            assert len(response["result"]["texts"]) == len(params["texts"]), response
            if params["texts"][0] == "":
                assert response["result"]["texts"] == params["texts"], response
            else:
                assert response["result"]["texts"][0] != params["texts"][0], response
        print(json.dumps(response, ensure_ascii=False))
    proc.stdin.close()
    assert proc.wait(timeout=60) == 0
denied = subprocess.run([str(host), "--library", str(library), "--sha256", "0" * 64], capture_output=True, text=True)
assert denied.returncode == 3 and "DENIED_BEFORE_DLOPEN" in denied.stderr, denied
print("Godwit ABI smoke test passed (including SHA-256 rejection).")
