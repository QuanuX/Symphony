import glob

for path in glob.glob("*/knowledge/sclv/CHANGELOG.md"):
    with open(path, "r") as f:
        content = f.read()
        
    idx = content.find("### SCLV-PR-010")
    if idx == -1: continue
    
    next_idx = content.find("### SCLV-PR-", idx + 15)
    if next_idx == -1: next_idx = content.find("Symphony Change Log Vector Ledger")
    if next_idx == -1: next_idx = len(content)
    
    block = content[idx:next_idx]
    
    new_block = block.replace("pull/11", "pull/10").replace("8b92a843e15652d1eab07978fcbb459cd840a318", "f2d65890f679107fdd114e51c5c8a22ab6eb2af2")
    
    new_content = content[:idx] + new_block + content[next_idx:]
    with open(path, "w") as f:
        f.write(new_content)

