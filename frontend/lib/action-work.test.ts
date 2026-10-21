import { describe, expect, it } from "vitest";
import { formatTaskRow, isStoredTask, parseTasksPayload } from "./action-work";

describe("parseTasksPayload", () => {
  it("reads work and empty tasks without inventing rows", () => {
    const parsed = parseTasksPayload({
      data: {
        tasks: [],
        work: [
          { id: "connect_gsc", kind: "loop", title: "Connect Google Search Console", href: "/integrations", done: false },
        ],
      },
    });
    expect(parsed.tasks).toEqual([]);
    expect(parsed.work).toHaveLength(1);
    expect(parsed.work[0].id).toBe("connect_gsc");
    expect(isStoredTask(parsed.work[0])).toBe(false);
  });

  it("keeps a legacy array of task rows", () => {
    const parsed = parseTasksPayload({ data: [{ id: 9, title: "Manual", completed: true, source: "manual" }] });
    expect(parsed.work).toEqual([]);
    expect(parsed.tasks).toHaveLength(1);
    expect(formatTaskRow(parsed.tasks[0]).isDone).toBe(true);
    expect(isStoredTask(parsed.tasks[0])).toBe(true);
  });
});
