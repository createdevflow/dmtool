import { describe, expect, it } from "vitest";
import { formatHealthOver100, formatHealthPercent } from "./health-score";

describe("health score honesty", () => {
  it("shows em dash when unaudited", () => {
    expect(formatHealthOver100(0)).toBe("—");
    expect(formatHealthOver100(undefined)).toBe("—");
    expect(formatHealthPercent(0)).toBe("—");
  });

  it("shows the stored score after an audit", () => {
    expect(formatHealthOver100(70)).toBe("70/100");
    expect(formatHealthPercent(70)).toBe("70%");
  });
});
