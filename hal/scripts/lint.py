#!/usr/bin/env python3
"""HAL lint — catch the refactor-leftover bug classes that `py_compile` and the
off-hardware test run miss, because both are runtime ImportError/NameError that
never surface until the code path executes on a device:

  1. broken local import — a `from .x` / `from hal.x` still points at a module
     that was renamed or moved (e.g. follower/leader `lelamp_*` -> `hal_*`).
  2. undefined name — a function still references a constant that a refactor
     deleted (e.g. `PI_DEBOUNCE_NS` after debounce moved to boards.json).

Dependency-light: (1) is stdlib AST; (2) uses pyflakes (dev dependency). Files in
the upstream LeLamp core are excluded — their inherited issues are not ours to fix.

Exit 1 if anything is found. Run: `make hal-lint`.
"""
import ast
import os
import subprocess
import sys

HAL = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))  # hal
# Upstream LeLamp robot core kept verbatim (see CLAUDE.md KEEP list). Its
# inherited undefined-name issues predate the fork and must not be "fixed".
KEEP = {"main.py", "smooth_animation.py"}


def py_files():
    out = []
    for root, dirs, files in os.walk(HAL):
        if ".venv" in root or "__pycache__" in root:
            continue
        for f in files:
            if f.endswith(".py"):
                out.append(os.path.join(root, f))
    return out


# ---- (1) broken local imports (stdlib AST; no third-party needed) -----------
def _mod_exists(base, parts):
    p = os.path.join(base, *parts)
    return os.path.isfile(p + ".py") or os.path.isdir(p)


_LOCAL_TOPS = {
    e[:-3] if e.endswith(".py") else e
    for e in os.listdir(HAL)
    if e.endswith(".py") or os.path.isdir(os.path.join(HAL, e))
}

_REPO = os.path.dirname(HAL)  # parent of hal/ — where the `hal.` package roots
_STDLIB = getattr(sys, "stdlib_module_names", frozenset())  # never a moved-local import


def _build_module_index():
    """basename (e.g. video_capture_device) -> [hal.-dotted paths that provide it].

    Lets us recognise a bare `from devices.x import Y` as a *moved* local module
    (now `hal.drivers.camera.x`) even after the old dir was deleted — the case the
    hal/-prefix and _LOCAL_TOPS checks miss because a gone dir reads as third-party.
    """
    idx = {}
    for root, dirs, files in os.walk(HAL):
        if ".venv" in root or "__pycache__" in root:
            continue
        for f in files:
            if f.endswith(".py") and f != "__init__.py":
                base = f[:-3]
                dotted = os.path.relpath(os.path.join(root, base), _REPO).replace(os.sep, ".")
                idx.setdefault(base, []).append(dotted)
    return idx


_HAL_MODULES = _build_module_index()


def _defined_names(path):
    """Top-level names a module exports (class/def/assign/import), for FP-safe matching."""
    try:
        tree = ast.parse(open(path).read(), path)
    except (SyntaxError, OSError):
        return set()
    names = set()
    for node in tree.body:
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
            names.add(node.name)
        elif isinstance(node, ast.Assign):
            names.update(t.id for t in node.targets if isinstance(t, ast.Name))
        elif isinstance(node, (ast.Import, ast.ImportFrom)):
            names.update(a.asname or a.name.split(".")[0] for a in node.names)
    return names


def check_imports(files):
    bad = []
    for path in files:
        try:
            tree = ast.parse(open(path).read(), path)
        except SyntaxError as e:
            bad.append(f"{path}: SYNTAX {e}")
            continue
        for n in ast.walk(tree):
            if not isinstance(n, ast.ImportFrom):
                continue
            if n.level:  # relative: from .x / from ..x
                base = os.path.dirname(path)
                for _ in range(n.level - 1):
                    base = os.path.dirname(base)
                if n.module and not _mod_exists(base, n.module.split(".")):
                    bad.append(f"{path}:{n.lineno}: from {'.' * n.level}{n.module} -> module not found")
            elif n.module:  # absolute, in-tree only
                parts = n.module.split(".")
                if parts[0] == "hal" and not _mod_exists(HAL, parts[1:]):
                    bad.append(f"{path}:{n.lineno}: from {n.module} -> not found under hal")
                elif parts[0] in _LOCAL_TOPS and not _mod_exists(HAL, parts):
                    bad.append(f"{path}:{n.lineno}: from {n.module} -> not found under hal")
                elif parts[0] not in ("hal", *_LOCAL_TOPS) and parts[0] not in _STDLIB:
                    # Looks third-party, but may be a moved local module imported by
                    # its old bare path (e.g. `from devices.x` after devices/ was
                    # deleted). Flag only when a hal module of the same basename
                    # actually defines every imported name — avoids third-party FPs.
                    # Stdlib is excluded so a local `typing.py` can't shadow `from typing`.
                    imported = {a.name for a in n.names}
                    for dotted in _HAL_MODULES.get(parts[-1], []):
                        cand = os.path.join(_REPO, *dotted.split(".")) + ".py"
                        if imported and imported <= _defined_names(cand):
                            bad.append(f"{path}:{n.lineno}: from {n.module} -> stale/moved local import; use {dotted}")
                            break
    return bad


# ---- (2) undefined names (pyflakes) -----------------------------------------
def check_undefined(files):
    try:
        import pyflakes  # noqa: F401
    except ImportError:
        print(
            "⚠ pyflakes not installed — skipping undefined-name check "
            "(install: cd hal && uv sync --extra dev)",
            file=sys.stderr,
        )
        return []
    res = subprocess.run(
        [sys.executable, "-m", "pyflakes", *files], capture_output=True, text=True
    )
    bad = []
    for line in (res.stdout + res.stderr).splitlines():
        if "undefined name" not in line:
            continue
        if os.path.basename(line.split(":", 1)[0]) in KEEP:
            continue
        bad.append(line)
    return bad


def main():
    files = py_files()
    problems = check_imports(files) + check_undefined(files)
    if problems:
        print("✗ HAL lint found refactor-leftover bugs (runtime Import/NameError):")
        for p in sorted(problems):
            print("  " + p)
        return 1
    print(
        f"✓ HAL lint clean — {len(files)} files: no broken local imports / "
        f"undefined names (upstream-keep {', '.join(sorted(KEEP))} excluded)."
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
