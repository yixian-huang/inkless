import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { listUnifiedPageVersions } from "@/api/unifiedPages";
import { VersionHistoryPanel } from "./VersionHistoryPanel";

vi.mock("@/api/unifiedPages", () => ({
  listUnifiedPageVersions: vi.fn(),
}));

describe("VersionHistoryPanel permissions", () => {
  it("hides rollback actions without pages:publish", async () => {
    vi.mocked(listUnifiedPageVersions).mockResolvedValue({
      items: [{ id: 1, version: 1, createdAt: "2026-07-16T00:00:00Z" }],
    } as never);

    render(
      <VersionHistoryPanel
        pageId={1}
        onClose={vi.fn()}
        onRollback={vi.fn()}
        canRollback={false}
      />,
    );

    await waitFor(() => expect(screen.getByText("版本 1")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: "回滚" })).not.toBeInTheDocument();
  });

  it("shows rollback actions with pages:publish", async () => {
    vi.mocked(listUnifiedPageVersions).mockResolvedValue({
      items: [{ id: 1, version: 1, createdAt: "2026-07-16T00:00:00Z" }],
    } as never);

    render(
      <VersionHistoryPanel
        pageId={1}
        onClose={vi.fn()}
        onRollback={vi.fn()}
        canRollback
      />,
    );

    await waitFor(() => expect(screen.getByRole("button", { name: "回滚" })).toBeInTheDocument());
  });
});
