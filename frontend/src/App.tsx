import { useEffect, useRef, useState } from "react";
import { Checkbox, Flex, Slider, Text } from "@radix-ui/themes";
import {
  GetState,
  IsStartupEnabled,
  SetLevel,
  SetStartupEnabled,
  ToggleLock,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { main } from "../wailsjs/go/models";
import Button from "./components/ui/Button";

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

// The reason an action failed is shown in the alert below, so the status line
// only states the lock state.
function statusText(s: main.State): string {
  if (!s.available) {
    return s.locked ? `Locked at ${s.target}%. Waiting for a microphone.` : "Not locked.";
  }
  if (s.locked) {
    return `Locked at ${s.target}%`;
  }
  return `Not locked. Microphone is at ${s.level}%.`;
}

function App() {
  const [state, setState] = useState<main.State | null>(null);
  const [startup, setStartup] = useState<boolean>(false);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState("");
  // While the user drags, the slider shows this value instead of the backend's.
  const [draft, setDraft] = useState<number | null>(null);

  const sliderRoot = useRef<HTMLDivElement>(null);
  const pendingLevel = useRef<number | null>(null);
  const sending = useRef(false);

  useEffect(() => {
    let active = true;

    GetState()
      .then((s) => active && setState(s))
      .catch((e) => active && setActionError(errorText(e)));
    IsStartupEnabled()
      .then((on) => active && setStartup(on))
      .catch((e) => active && setActionError(errorText(e)));

    const offState = EventsOn("state", (s: main.State) => setState(s));
    const offStartup = EventsOn("startup", (on: boolean) => setStartup(on));
    return () => {
      active = false;
      offState();
      offStartup();
    };
  }, []);

  // Radix only labels a slider thumb from the thumb's own props, which this
  // wrapper cannot reach, so the label is set on the rendered thumb.
  useEffect(() => {
    sliderRoot.current
      ?.querySelectorAll<HTMLElement>('[role="slider"]')
      .forEach((thumb) => thumb.setAttribute("aria-label", "Volume"));
  });

  // Slider moves are sent one at a time. If the user keeps dragging while a
  // request is in flight, only the latest value is sent next.
  const sendLevel = async () => {
    if (sending.current) return;
    sending.current = true;
    try {
      while (pendingLevel.current !== null) {
        const level = pendingLevel.current;
        pendingLevel.current = null;
        setState(await SetLevel(level));
      }
    } catch (e) {
      pendingLevel.current = null;
      setActionError(errorText(e));
    } finally {
      sending.current = false;
      if (pendingLevel.current === null) setDraft(null);
    }
  };

  const handleSlider = (values: number[]) => {
    setDraft(values[0]);
    setActionError("");
    pendingLevel.current = values[0];
    void sendLevel();
  };

  const handleToggle = async () => {
    setBusy(true);
    setActionError("");
    try {
      setState(await ToggleLock());
    } catch (e) {
      setActionError(errorText(e));
    } finally {
      setBusy(false);
    }
  };

  const handleStartup = async (want: boolean) => {
    setActionError("");
    try {
      setStartup(await SetStartupEnabled(want));
    } catch (e) {
      setActionError(errorText(e));
      setStartup(await IsStartupEnabled().catch(() => !want));
    }
  };

  if (!state) {
    return <div className="container mx-auto p-2 max-w-md" aria-busy="true" />;
  }

  const shown = draft ?? state.target;
  const message = actionError || state.message;

  return (
    <div className="container mx-auto p-2 max-w-md">
      <div className="space-y-3 px-[4px]">
        <p
          role="status"
          aria-live="polite"
          className={state.locked ? "text-blue-300 text-sm" : "text-gray-300 text-sm"}
        >
          {statusText(state)}
        </p>

        <Button
          onClick={handleToggle}
          disabled={busy || (!state.available && !state.locked)}
          className="w-full h-10 text-base font-bold"
          variant={state.locked ? "primary-dark" : "primary"}
        >
          {state.locked
            ? "Unlock Microphone Volume"
            : "Lock Microphone Volume"}
        </Button>

        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span>Volume</span>
            <span>{shown}%</span>
          </div>
          <div ref={sliderRoot}>
            <Slider
              min={0}
              step={1}
              max={100}
              value={[shown]}
              onValueChange={handleSlider}
              disabled={!state.available}
              className="w-full rounded-lg"
              color="gray"
            />
          </div>
        </div>

        <Text as="label" size="2">
          <Flex align="center" justify="center" gap="2">
            <Checkbox
              checked={startup}
              onCheckedChange={(v) => handleStartup(v === true)}
            />
            Run at Windows startup
          </Flex>
        </Text>

        {message && (
          <p role="alert" className="text-amber-300 text-sm">
            {message}
          </p>
        )}
      </div>
    </div>
  );
}

export default App;
