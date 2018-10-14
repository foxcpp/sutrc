#!/usr/bin/python3
# Generate translated versions of pages by replacing placeholders in form of
# $PLACEHOLDER_NAME or ${PLACEHOLDER_NAME} with PLACEHOLDER_NAME value from
# lang. file. Output is written to out directory in cwd.
#
# Requires yaml (PyYAML) module.
#
# Usage: langsubst.py langfile

import sys
import yaml
import pathlib
import string

if len(sys.argv) != 2:
    print("Usage:", sys.argv[0], "<langfile>")
    sys.exit(1)

langfile = yaml.load(open(sys.argv[1]))

outdir = pathlib.Path("out")
outdir.mkdir(exist_ok=True)
for pattern in ["*.css", "*.js", "*.html"]:
    for file in pathlib.Path(".").rglob(pattern):
        if outdir in file.parents:
            continue
        output = outdir.joinpath(file)
        output.parent.mkdir(exist_ok=True)
        print(file, "=>", output)
        tmplt = string.Template(file.read_text())
        output.open("w").write(tmplt.safe_substitute(**langfile))
