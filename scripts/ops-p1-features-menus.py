#!/usr/bin/env python3
"""Seed primary menu for inkless-ops product site. Features page ships in theme package."""
from __future__ import annotations

import sqlite3
import time
import shutil
from datetime import datetime, timezone
from pathlib import Path

DB = Path("/opt/inkless-ops/data/inkless.db")
DOCS_URL = "https://docs.inkless.run"
GITHUB = "https://github.com/yixian-huang/inkless"


def main() -> None:
    if not DB.exists() or "inkless-ops" not in str(DB):
        raise SystemExit(f"refuse: {DB}")
    bak = DB.with_name(f"inkless.db.bak-p1-menus-{int(time.time())}")
    shutil.copy2(DB, bak)
    print("backup", bak)

    con = sqlite3.connect(str(DB))
    con.row_factory = sqlite3.Row
    cur = con.cursor()
    now = datetime.now(timezone.utc).isoformat()

    # Clear existing menus for a clean primary
    cur.execute("delete from menu_items")
    cur.execute("delete from menu_groups")

    cur.execute(
        "insert into menu_groups (name, slug, is_primary, created_at, updated_at) values (?,?,?,?,?)",
        ("Primary", "primary", 1, now, now),
    )
    group_id = cur.lastrowid
    print("group", group_id)

    visible = 1
    items = [
        # zh_name, en_name, type, target, url, sort
        ("首页", "Home", "custom_link", "_self", "/", 0),
        ("能力", "Features", "custom_link", "_self", "/features", 10),
        ("文档", "Docs", "custom_link", "_blank", DOCS_URL, 20),
        ("联系", "Contact", "custom_link", "_self", "/contact", 30),
        ("GitHub", "GitHub", "custom_link", "_blank", GITHUB, 40),
    ]
    for zh, en, typ, target, url, sort in items:
        cur.execute(
            """insert into menu_items
            (group_id, parent_id, zh_name, en_name, type, target, url, ref_id, ref_slug,
             visible, metadata, sort_order, created_at, updated_at)
            values (?,?,?,?,?,?,?,?,?,?,?,?,?,?)""",
            (
                group_id,
                None,
                zh,
                en,
                typ,
                target,
                url,
                None,
                "",
                visible,
                "{}",
                sort,
                now,
                now,
            ),
        )
        print("item", zh, url)

    con.commit()
    # verify
    n = cur.execute("select count(*) from menu_items").fetchone()[0]
    print("items", n)
    con.close()
    print("menus seeded")


if __name__ == "__main__":
    main()
