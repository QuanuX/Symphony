import os

base_record_011 = """
### SCLV-PR-011
- record_id: `SCLV-PR-011`
- title: `Title 2`
- status: `canonical`
- date: `2026-07-05`
- change_type: `canonical_addition`
- related_pr: `https://github.com/QuanuX/Symphony/pull/11`
- merge_commit: `8b92a843e15652d1eab07978fcbb459cd840a318`
- affected_surfaces: `none`
- skvi_references: `none`
- change_summary: `summary`
- relationship_changes: `none`
- doctrine_changes: `none`
- compatibility_consequences: `none`
- publication_consequences: `none`
- projection_consequences: `none`
- evidence: `PR`
- non_authorizations: `none`
- notes: `notes`
"""

def fix_mismatch():
    path = "fixtures_sclv_record_pr_mismatch/knowledge/sclv/CHANGELOG.md"
    with open(path, "r") as f:
        content = f.read()
    # Replace SCLV-PR-011 block with good 011 and a bad 012
    good_011 = base_record_011
    bad_012 = base_record_011.replace("SCLV-PR-011", "SCLV-PR-012").replace("pull/11", "pull/99").replace("8b92a843e15652d1eab07978fcbb459cd840a318", "commit012")
    # In the current file, 011 is actually modified to pull/99. We just replace everything after 010.
    start = content.find("### SCLV-PR-011")
    end = content.find("Symphony Change Log Vector Ledger")
    new_content = content[:start] + good_011.strip() + "\n\n" + bad_012.strip() + "\n\n" + content[end:]
    with open(path, "w") as f:
        f.write(new_content)

def fix_dup_id():
    path = "fixtures_sclv_duplicate_record_id/knowledge/sclv/CHANGELOG.md"
    with open(path, "r") as f:
        content = f.read()
    good_011 = base_record_011
    bad_012 = base_record_011.replace("SCLV-PR-011", "SCLV-PR-011", 1) # ID is same
    # wait, we want the record to be SCLV-PR-011 but id is 011? That's already duplicate.
    # Let's just make the header SCLV-PR-012 but ID SCLV-PR-011.
    bad_012 = base_record_011.replace("### SCLV-PR-011", "### SCLV-PR-012").replace("pull/11", "pull/12").replace("8b92a843e15652d1eab07978fcbb459cd840a318", "commit012")
    start = content.find("### SCLV-PR-010\n- record_id: `SCLV-PR-010`") # Wait, in this one I changed 011 to 010.
    start = content.find("### SCLV-PR-010", content.find("### SCLV-PR-010") + 1)
    if start == -1: # fallback
        start = content.find("### SCLV-PR-011")
    if start != -1:
        end = content.find("Symphony Change Log Vector Ledger")
        new_content = content[:start] + good_011.strip() + "\n\n" + bad_012.strip() + "\n\n" + content[end:]
        with open(path, "w") as f:
            f.write(new_content)

def fix_dup_pr():
    path = "fixtures_sclv_duplicate_related_pr/knowledge/sclv/CHANGELOG.md"
    with open(path, "r") as f:
        content = f.read()
    good_011 = base_record_011
    bad_012 = base_record_011.replace("SCLV-PR-011", "SCLV-PR-012").replace("pull/12", "pull/11").replace("8b92a843e15652d1eab07978fcbb459cd840a318", "commit012")
    start = content.find("### SCLV-PR-011")
    end = content.find("Symphony Change Log Vector Ledger")
    new_content = content[:start] + good_011.strip() + "\n\n" + bad_012.strip() + "\n\n" + content[end:]
    with open(path, "w") as f:
        f.write(new_content)

def fix_dup_commit():
    path = "fixtures_sclv_duplicate_merge_commit/knowledge/sclv/CHANGELOG.md"
    with open(path, "r") as f:
        content = f.read()
    good_011 = base_record_011
    bad_012 = base_record_011.replace("SCLV-PR-011", "SCLV-PR-012").replace("pull/11", "pull/12").replace("8b92a843e15652d1eab07978fcbb459cd840a318", "f2d65890f679107fdd114e51c5c8a22ab6eb2af2")
    start = content.find("### SCLV-PR-011")
    end = content.find("Symphony Change Log Vector Ledger")
    new_content = content[:start] + good_011.strip() + "\n\n" + bad_012.strip() + "\n\n" + content[end:]
    with open(path, "w") as f:
        f.write(new_content)

def fix_gap():
    path = "fixtures_sclv_ledger_gap_warning/knowledge/sclv/CHANGELOG.md"
    with open(path, "r") as f:
        content = f.read()
    good_011 = base_record_011
    bad_033 = base_record_011.replace("SCLV-PR-011", "SCLV-PR-033").replace("pull/11", "pull/33").replace("8b92a843e15652d1eab07978fcbb459cd840a318", "commit033")
    start = content.find("### SCLV-PR-033")
    if start == -1:
        start = content.find("### SCLV-PR-011")
    end = content.find("Symphony Change Log Vector Ledger")
    new_content = content[:start] + good_011.strip() + "\n\n" + bad_033.strip() + "\n\n" + content[end:]
    with open(path, "w") as f:
        f.write(new_content)

fix_mismatch()
fix_dup_id()
fix_dup_pr()
fix_dup_commit()
fix_gap()
