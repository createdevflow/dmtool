import { describe, expect, it } from "vitest";
import { parseAlertsPayload, unreadAlertCount } from "./alerts";

describe("parseAlertsPayload", () => {
  it("counts stored alerts and treats empty as zero", () => {
    expect(parseAlertsPayload({ data: [] })).toEqual([]);
    expect(unreadAlertCount([])).toBe(0);
  });

  it("counts unread derived alerts without inventing extras", () => {
    const alerts = parseAlertsPayload({
      data: [
        { id: "gsc_missing", title: "Connect Google Search Console", is_read: false, href: "/integrations" },
        { id: "seo_high", title: "2 high-severity SEO issues", is_read: false },
      ],
    });
    expect(alerts).toHaveLength(2);
    expect(unreadAlertCount(alerts)).toBe(2);
    expect(alerts.map((a) => a.id)).not.toContain("traffic_drop");
  });
});
