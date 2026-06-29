"""Generate 伍孝轩 resume — uses 林炜豪 template (icons + tables preserved)."""

from __future__ import annotations

from fill_resume_from_template import build

if __name__ == "__main__":
    print(build(export_pdf_file=False))
