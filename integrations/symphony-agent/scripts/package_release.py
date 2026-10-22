#!/usr/bin/env python3
"""Create a deterministic local review ZIP; never upload or include profiles."""
import argparse
import hashlib
import json
from pathlib import Path
import stat
import zipfile

ROOT = Path(__file__).resolve().parents[1]
FILES = ('README.md', 'LICENSE', 'CMakeLists.txt', 'plugin.json', 'mcp.json', '.mcp.json', '.gitignore')
DIRS = ('src', 'include', 'third_party', 'scripts', 'tests', 'skills', 'docs', 'adapters', '.claude-plugin', '.codex-plugin', 'assets')

def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--output', required=True, type=Path)
    p.add_argument('--binary', type=Path, help='Include an explicitly selected built binary as bin/symphony-mcp')
    args = p.parse_args()
    selected = [ROOT / name for name in FILES if (ROOT / name).is_file()]
    for name in DIRS:
        directory = ROOT / name
        if directory.exists():
            selected.extend(x for x in directory.rglob('*') if x.is_file() and '__pycache__' not in x.parts and x.suffix != '.pyc')
    members = {str(x.relative_to(ROOT)): x for x in selected}
    if args.binary:
        if not args.binary.is_file() or args.binary.is_symlink():
            p.error('Select a regular compiled binary')
        members['bin/symphony-mcp'] = args.binary
    if len(members) > 5000:
        p.error('Too many archive members')
    total = 0
    entries = []
    for name, source in sorted(members.items()):
        if source.is_symlink() or Path(name).is_absolute() or '..' in Path(name).parts:
            p.error('Unsafe archive member')
        data = source.read_bytes()
        if name != 'bin/symphony-mcp':
            # Do not ship developer-machine paths, connection profiles or local evidence.
            if str(Path.home()).encode() in data or str(ROOT.parent / 'symphony-agent-plugin-evidence').encode() in data:
                p.error('Private local path/evidence found in '+name)
        total += len(data)
        if len(data)>100*1024*1024 or total>512*1024*1024:
            p.error('Archive exceeds published size limits')
        entries.append((name, data))
    args.output.parent.mkdir(parents=True, exist_ok=True)
    # Exclusive creation avoids silently replacing an already-reviewed candidate.
    with open(args.output, 'xb') as stream, zipfile.ZipFile(stream,'w',compression=zipfile.ZIP_DEFLATED,compresslevel=9) as archive:
        for name,data in entries:
            info=zipfile.ZipInfo(name, (2026,10,22,0,0,0))
            info.create_system=3
            mode=0o755 if name=='bin/symphony-mcp' else 0o644
            info.external_attr=(stat.S_IFREG|mode)<<16
            info.compress_type=zipfile.ZIP_DEFLATED
            archive.writestr(info,data)
    if args.output.stat().st_size>100_000_000:
        args.output.unlink()
        p.error('Compressed archive exceeds100MB')
    result={'artifact':str(args.output),'sha256':hashlib.sha256(args.output.read_bytes()).hexdigest(),'files':len(entries),'expanded_bytes':total,'compressed_bytes':args.output.stat().st_size,'includes_binary':bool(args.binary),'submitted':False}
    args.output.with_suffix(args.output.suffix+'.sha256').write_text(result['sha256']+'  '+args.output.name+'\n')
    print(json.dumps(result,indent=2))

if __name__=='__main__':
    main()
