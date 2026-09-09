import { useState } from "react";
import EventList from "./EventList";
import EventDetail from "./EventDetail";
import SendTestEvent from "./SendTestEvent";

type Screen = "list" | "detail" | "new";

export default function App() {
  const [screen, setScreen] = useState<Screen>("list");
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);

  return (
    <div className="app">
      <div className="header-row">
        <h1>Webhook Relay</h1>
        {screen === "list" && <button onClick={() => setScreen("new")}>+ New</button>}
      </div>

      {screen === "list" && (
        <EventList
          onSelect={(id) => {
            setSelectedEventId(id);
            setScreen("detail");
          }}
        />
      )}

      {screen === "detail" && selectedEventId && (
        <EventDetail id={selectedEventId} onBack={() => setScreen("list")} />
      )}

      {screen === "new" && (
        <SendTestEvent
          onSent={(id) => {
            setSelectedEventId(id);
            setScreen("detail");
          }}
          onCancel={() => setScreen("list")}
        />
      )}
    </div>
  );
}
