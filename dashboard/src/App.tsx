import { useState } from "react";
import EventList from "./EventList";
import EventDetail from "./EventDetail";

export default function App() {
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);

  return (
    <div className="app">
      <h1>Webhook Relay</h1>
      {selectedEventId ? (
        <EventDetail id={selectedEventId} onBack={() => setSelectedEventId(null)} />
      ) : (
        <EventList onSelect={setSelectedEventId} />
      )}
    </div>
  );
}
