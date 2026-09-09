import { useState } from "react";
import { createEndpoint, createEvent } from "./api";

const DEFAULT_PAYLOAD = '{\n  "hello": "world"\n}';

export default function SendTestEvent({
  onSent,
  onCancel,
}: {
  onSent: (eventId: string) => void;
  onCancel: () => void;
}) {
  const [url, setUrl] = useState("");
  const [eventType, setEventType] = useState("test.event");
  const [payload, setPayload] = useState(DEFAULT_PAYLOAD);
  const [error, setError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    let parsedPayload: unknown;
    try {
      parsedPayload = JSON.parse(payload);
    } catch {
      setError("Payload must be valid JSON");
      return;
    }

    setSending(true);
    try {
      const endpoint = await createEndpoint(url);
      const event = await createEvent(endpoint.id, eventType, parsedPayload);
      onSent(event.event_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSending(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="send-form">
      <p className="send-form-note">
        Registers a fresh endpoint for this URL, then sends one event to it — a quick way to try delivery
        without the API directly.
      </p>

      <label>
        Destination URL
        <input
          type="url"
          required
          placeholder="https://webhook.site/your-id"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
        />
      </label>

      <label>
        Event type
        <input
          type="text"
          required
          value={eventType}
          onChange={(e) => setEventType(e.target.value)}
        />
      </label>

      <label>
        Payload (JSON)
        <textarea
          className="mono"
          rows={6}
          value={payload}
          onChange={(e) => setPayload(e.target.value)}
        />
      </label>

      {error && <p className="error">{error}</p>}

      <div className="send-form-actions">
        <button type="submit" disabled={sending}>
          {sending ? "Sending…" : "Send"}
        </button>
        <button type="button" className="secondary" onClick={onCancel} disabled={sending}>
          Cancel
        </button>
      </div>
    </form>
  );
}
