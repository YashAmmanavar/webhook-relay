import type { EventStatus } from "./api";

const COLORS: Record<EventStatus, string> = {
  PENDING: "#8a8a8a",
  PROCESSING: "#8a8a8a",
  RETRYING: "#b8860b",
  DELIVERED: "#1a7f37",
  FAILED: "#cf222e",
};

export default function StatusBadge({ status }: { status: EventStatus }) {
  return (
    <span
      style={{
        display: "inline-block",
        padding: "2px 8px",
        borderRadius: 4,
        fontSize: 12,
        fontWeight: 600,
        color: "#fff",
        backgroundColor: COLORS[status] ?? "#8a8a8a",
      }}
    >
      {status}
    </span>
  );
}
