import os
import glob

for path in glob.glob("*/knowledge/sclv/CHANGELOG.md"):
    if "mismatch" in path or "duplicate" in path or "gap" in path:
        continue
        
    with open(path, "r") as f:
        content = f.read()
        
    # We want to replace pull/10 with pull/11 and the commit hash for SCLV-PR-011
    # Find SCLV-PR-011 block
    idx = content.find("### SCLV-PR-011")
    if idx == -1:
        continue
        
    next_idx = content.find("### SCLV-PR-", idx + 15)
    if next_idx == -1:
        next_idx = content.find("Symphony Change Log Vector Ledger")
    if next_idx == -1:
        next_idx = len(content)
        
    block = content[idx:next_idx]
    
    # Modify block
    new_block = block.replace("pull/10", "pull/11").replace("f2d65890f679107fdd114e51c5c8a22ab6eb2af2", "8b92a843e15652d1eab07978fcbb459cd840a318")
    
    new_content = content[:idx] + new_block + content[next_idx:]
    with open(path, "w") as f:
        f.write(new_content)

