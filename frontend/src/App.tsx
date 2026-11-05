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

function LockIcon({ open = false }: { open?: boolean }) {
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2.25"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <rect x="5" y="11" width="14" height="10" rx="2" />
      {open ? (
        <path d="M8 11V7a4 4 0 0 1 7.7-1.5" />
      ) : (
        <path d="M8 11V7a4 4 0 0 1 8 0v4" />
      )}
    </svg>
  );
}

function StatusPill({ state }: { state: main.State }) {
  let tone: string;
  let label: string;
  if (!state.available) {
    tone = "bg-amber-400/15 text-amber-300";
    label = state.locked ? "Waiting" : "No mic";
  } else if (state.locked) {
    tone = "bg-blue-500/20 text-blue-200";
    label = "Locked";
  } else {
    tone = "bg-white/10 text-gray-300";
    label = "Unlocked";
  }
  return (
    <span
      role="status"
      aria-live="polite"
      className={`inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ${tone}`}
    >
      {state.locked && state.available && <LockIcon />}
      {label}
    </span>
  );
}

function App() {
  const [state, setState] = useState<main.State | null>(null);
  const [startup, setStartup] = useState(false);
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

  // Radix labels a slider thumb only from props the thumb itself receives, so
  // the accessible name is set on the rendered thumb.
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
    return <div className="h-full" aria-busy="true" />;
  }

  const level = draft ?? state.target;
  const problem = actionError || state.message;

  return (
    <div className="app flex flex-col gap-3 px-5 py-4 text-left">
      <header className="flex items-end justify-between gap-3">
        <div className="min-w-0">
          {problem ? (
            <p
              role="alert"
              title={problem}
              className="truncate text-xs text-amber-300"
            >
              {problem}
            </p>
          ) : (
            <p className="text-xs text-gray-400">Microphone volume</p>
          )}
          <p className="text-[2.5rem] font-semibold leading-none tabular-nums">
            {level}
            <span className="ml-0.5 text-xl font-medium text-gray-400">%</span>
          </p>
        </div>
        <StatusPill state={state} />
      </header>

      <div ref={sliderRoot}>
        <Slider
          min={0}
          step={1}
          max={100}
          value={[level]}
          onValueChange={handleSlider}
          disabled={!state.available}
          className="w-full"
        />
      </div>

      <Button
        onClick={handleToggle}
        variant={state.locked ? "secondary" : "primary"}
        disabled={busy || (!state.available && !state.locked)}
        className="w-full"
      >
        <LockIcon open={state.locked} />
        {state.locked ? "Unlock" : `Lock at ${level}%`}
      </Button>

      <Text as="label" size="2" color="gray">
        <Flex align="center" gap="2">
          <Checkbox
            size="1"
            checked={startup}
            onCheckedChange={(v) => handleStartup(v === true)}
          />
          Start with Windows
        </Flex>
      </Text>
    </div>
  );
}

export default App;
