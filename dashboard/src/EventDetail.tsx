import { useEffect, useState } from "react";
import { getEvent, retryEvent, type EventDetail as EventDetailType } from "./api";
import StatusBadge from "./StatusBadge";

const POLL_INTERVAL_MS = 3000;

export default function EventDetail({ id, onBack }: { id: string; onBack: () => void }) {
  const [detail, setDetail] = useState<EventDetailType | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [retrying, setRetrying] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const data = await getEvent(id);
        if (!cancelled) {
          setDetail(data);
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
  }, [id]);

  async function handleRetry() {
    setRetrying(true);
    try {
      await retryEvent(id);
      const data = await getEvent(id);
      setDetail(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setRetrying(false);
    }
  }

  return (
    <div>
      <button onClick={onBack} className="back-link">
        &larr; Back to events
      </button>

      {error && <p className="error">{error}</p>}

      {detail && (
        <>
          <h2 className="mono">{detail.id}</h2>
          <p>
            Type: <strong>{detail.event_type}</strong>
          </p>
          <p>
            Status: <StatusBadge status={detail.status} />
          </p>
          <p>Attempts: {detail.attempt_count}</p>
          {detail.status === "RETRYING" && detail.next_retry_at && (
            <p>Next retry: {new Date(detail.next_retry_at).toLocaleString()}</p>
          )}
          {detail.status === "FAILED" && (
            <button onClick={handleRetry} disabled={retrying}>
              {retrying ? "Retrying…" : "Retry"}
            </button>
          )}

          <h3>Delivery Attempts</h3>
          {detail.attempts.length === 0 && <p>No attempts yet.</p>}
          <ul className="attempts">
            {detail.attempts.map((a) => (
              <li key={a.id}>
                <div>
                  <strong>Attempt {a.attempt_number}</strong>
                </div>
                <div>{a.status_code ? `HTTP ${a.status_code}` : a.error ?? "no response"}</div>
                <div>{a.duration_ms != null ? `${a.duration_ms} ms` : ""}</div>
                <div className="attempt-time">{new Date(a.created_at).toLocaleString()}</div>
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}
