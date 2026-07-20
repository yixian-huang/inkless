import { fireEvent, render, screen } from "@/test/test-utils";
import { describe, expect, it, vi } from "vitest";
import { ScheduledPublicationPanel } from "./ScheduledPublicationPanel";

describe("ScheduledPublicationPanel", () => {
  it("shows pending schedule details and cancel action", () => {
    const onCancel = vi.fn();

    render(
      <ScheduledPublicationPanel
        item={{
          id: 9,
          resourceType: "page",
          resourceId: 4,
          status: "pending",
          scheduledAt: "2026-07-17T02:30:00.000Z",
          expectedVersion: 6,
          createdAt: "2026-07-16T00:00:00.000Z",
          updatedAt: "2026-07-16T00:00:00.000Z",
        }}
        canPublish
        onSchedule={vi.fn()}
        onCancel={onCancel}
      />,
    );

    expect(screen.getByText("等待发布")).toBeInTheDocument();
    expect(screen.getByText("6")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "取消" }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("shows failure details and retry action", () => {
    const onRetry = vi.fn();

    render(
      <ScheduledPublicationPanel
        item={{
          id: 10,
          resourceType: "article",
          resourceId: 5,
          status: "failed",
          scheduledAt: "2026-07-17T02:30:00.000Z",
          lastError: "publish worker unavailable",
          createdAt: "2026-07-16T00:00:00.000Z",
          updatedAt: "2026-07-16T00:00:00.000Z",
        }}
        canPublish
        onSchedule={vi.fn()}
        onCancel={vi.fn()}
        onRetry={onRetry}
      />,
    );

    expect(screen.getByText("发布失败")).toBeInTheDocument();
    expect(screen.getByText("publish worker unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "重试" }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("hides publish actions without permission", () => {
    render(
      <ScheduledPublicationPanel
        item={null}
        canPublish={false}
        disabledReason="需要权限"
        onSchedule={vi.fn()}
        onCancel={vi.fn()}
      />,
    );

    expect(screen.queryByRole("button", { name: "安排发布" })).not.toBeInTheDocument();
    expect(screen.getByText("需要权限")).toBeInTheDocument();
  });

  it("renders compact single-line toolbar", () => {
    render(
      <ScheduledPublicationPanel
        compact
        item={null}
        canPublish
        onSchedule={vi.fn()}
        onCancel={vi.fn()}
      />,
    );

    expect(screen.getByText("未安排")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "定时" })).toBeInTheDocument();
  });
});
