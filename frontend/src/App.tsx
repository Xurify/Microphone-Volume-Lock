import { useEffect, useRef, useState } from "react";
import { Checkbox, Slider } from "@radix-ui/themes";
import {
  GetState,
  IsStartupEnabled,
  SetLevel,
  SetStartupEnabled,
  ToggleLock,
  Version,
} from "../wailsjs/go/main/App";
import { EventsOn, Quit, WindowHide } from "../wailsjs/runtime/runtime";
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
    tone = "bg-[var(--red-a3)] text-[var(--red-11)]";
    label = state.locked ? "Waiting" : "No mic";
  } else if (state.locked) {
    tone = "bg-[var(--accent-a4)] text-[var(--accent-11)]";
    label = "Locked";
  } else {
    tone = "bg-[var(--gray-a3)] text-[var(--gray-11)]";
    label = "Unlocked";
  }
  return (
    <span
      role="status"
      aria-live="polite"
      className={`mb-1 inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ${tone}`}
    >
      {state.locked && state.available && <LockIcon />}
      {label}
    </span>
  );
}

function App() {
  const [state, setState] = useState<main.State | null>(null);
  const [startup, setStartup] = useState(false);
  const [version, setVersion] = useState("");
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState("");
  // While the user drags, the slider shows this value instead of the backend's.
  const [draft, setDraft] = useState<number | null>(null);

  const sliderRoot = useRef<HTMLDivElement>(null);
  const pendingLevel = useRef<number | null>(null);
  const sending = useRef(false);
  const startupRef = useRef(startup);
  startupRef.current = startup;

  useEffect(() => {
    let active = true;

    GetState()
      .then((s) => active && setState(s))
      .catch((e) => active && setActionError(errorText(e)));
    IsStartupEnabled()
      .then((on) => active && setStartup(on))
      .catch((e) => active && setActionError(errorText(e)));
    Version()
      .then((v) => active && setVersion(v))
      .catch(() => {});

    const offState = EventsOn("state", (s: main.State) => setState(s));
    const offStartup = EventsOn("startup", (on: boolean) => setStartup(on));
    return () => {
      active = false;
      offState();
      offStartup();
    };
  }, []);

  // The window has no menu bar, so its old shortcuts live here.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!e.ctrlKey || e.altKey || e.shiftKey || e.metaKey) return;
      const key = e.key.toLowerCase();
      if (key === "h") {
        e.preventDefault();
        WindowHide();
      } else if (key === "q") {
        e.preventDefault();
        Quit();
      } else if (key === "s") {
        e.preventDefault();
        void handleStartup(!startupRef.current);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
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

  async function handleStartup(want: boolean) {
    setActionError("");
    try {
      setStartup(await SetStartupEnabled(want));
    } catch (e) {
      setActionError(errorText(e));
      setStartup(await IsStartupEnabled().catch(() => !want));
    }
  }

  if (!state) {
    return <div aria-busy="true" />;
  }

  const level = draft ?? state.target;
  const problem = actionError || state.message;

  return (
    <div className="app flex flex-col gap-4 px-5 pb-3 pt-4 text-left">
      <header className="flex items-end justify-between gap-3">
        <div className="min-w-0">
          {problem ? (
            <p
              role="alert"
              title={problem}
              className="truncate text-xs text-[var(--red-11)]"
            >
              {problem}
            </p>
          ) : (
            <p className="text-xs text-[var(--gray-11)]">Microphone volume</p>
          )}
          <p
            className={`mt-1 text-[2.75rem] font-semibold leading-none tracking-tight tabular-nums ${
              state.available ? "text-[var(--gray-12)]" : "text-[var(--gray-9)]"
            }`}
          >
            {level}
            <span className="ml-1 text-xl font-medium text-[var(--gray-10)]">
              %
            </span>
          </p>
        </div>
        <StatusPill state={state} />
      </header>

      <div
        ref={sliderRoot}
        className={state.available ? "" : "cursor-not-allowed opacity-40 grayscale"}
      >
        <Slider
          min={0}
          step={1}
          max={100}
          value={[level]}
          onValueChange={handleSlider}
          disabled={!state.available}
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

      <footer className="flex items-center justify-between border-t border-[var(--gray-a4)] pt-3 text-xs text-[var(--gray-11)]">
        <label className="flex cursor-pointer items-center gap-2">
          <Checkbox
            size="1"
            checked={startup}
            onCheckedChange={(v) => handleStartup(v === true)}
          />
          Start with Windows
        </label>
        <span
          className="text-[var(--gray-10)]"
          title="Ctrl+H hides to the tray · Ctrl+Q quits"
        >
          {version && version !== "dev" ? `v${version}` : "Ctrl+H to hide"}
        </span>
      </footer>
    </div>
  );
}

export default App;
