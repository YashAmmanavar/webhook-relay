import { useEffect, useState } from "react";
import { listEvents, type Event, type EventStatus } from "./api";
import StatusBadge from "./StatusBadge";

const FILTERS: { label: string; value: EventStatus | "" }[] = [
  { label: "All", value: "" },
  { label: "Delivered", value: "DELIVERED" },
  { label: "Retrying", value: "RETRYING" },
  { label: "Failed", value: "FAILED" },
];

const POLL_INTERVAL_MS = 3000;

export default function EventList({ onSelect }: { onSelect: (id: string) => void }) {
  const [filter, setFilter] = useState<EventStatus | "">("");
  const [events, setEvents] = useState<Event[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const data = await listEvents(filter || undefined);
        if (!cancelled) {
          setEvents(data);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      }
    }

    load();
    const interval = setInterval(load, POLL_INTERVAL_MS);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [filter]);

  return (
    <div>
      <div className="filters">
        {FILTERS.map((f) => (
          <button
            key={f.label}
            className={filter === f.value ? "filter-active" : ""}
            onClick={() => setFilter(f.value)}
          >
            {f.label}
          </button>
        ))}
      </div>

      {error && <p className="error">{error}</p>}

      <table>
        <thead>
          <tr>
            <th>Event ID</th>
            <th>Type</th>
            <th>Status</th>
            <th>Attempts</th>
            <th>Created</th>
          </tr>
        </thead>
        <tbody>
          {events.map((ev) => (
            <tr key={ev.id} onClick={() => onSelect(ev.id)} className="clickable-row">
              <td className="mono">{ev.id}</td>
              <td>{ev.event_type}</td>
              <td>
                <StatusBadge status={ev.status} />
              </td>
              <td>{ev.attempt_count}</td>
              <td>{new Date(ev.created_at).toLocaleString()}</td>
            </tr>
          ))}
          {events.length === 0 && (
            <tr>
              <td colSpan={5} className="empty">
                No events yet.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
