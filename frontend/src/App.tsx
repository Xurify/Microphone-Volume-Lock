import { useState, useEffect, useCallback } from "react";
import {
  GetVolume,
  SetVolume,
  ToggleLock,
  IsLocked,
  StopAllNircmd,
  IsStartupEnabled,
} from "../wailsjs/go/main/App";
import { EventsOn, EventsEmit } from "../wailsjs/runtime/runtime";
import { Slider } from "@radix-ui/themes";
import Button from "./components/ui/Button";

interface AppState {
  volume: number;
  isLocked: boolean;
  startupEnabled: boolean;
}

function App() {
  const [state, setState] = useState<AppState>({
    volume: 75,
    isLocked: false,
    startupEnabled: false,
  });

  useEffect(() => {
    const loadState = async () => {
      try {
        const [volume, isLocked, startupEnabled] = await Promise.all([
          GetVolume(),
          IsLocked(),
          IsStartupEnabled(),
        ]);
        setState((prev) => ({
          ...prev,
          volume,
          isLocked,
          startupEnabled,
        }));
      } catch (error) {
        console.error("Failed to load initial state:", error);
      }
    };
    loadState();
  }, []);

  useEffect(() => {
    const unsubscribe = EventsOn(
      "lock-state-changed",
      (newLockState: boolean) => {
        setState((prev) => ({ ...prev, isLocked: newLockState }));
      }
    );

    return () => {
      unsubscribe();
    };
  }, []);

  const handleVolumeChange = useCallback(async (newVolume: number[]) => {
    try {
      const volumeValue = newVolume[0];
      await SetVolume(volumeValue);
      setState((prev) => ({ ...prev, volume: volumeValue }));
    } catch (error) {
      console.error("Failed to set volume:", error);
    }
  }, []);

  const handleLockToggle = useCallback(async () => {
    try {
      await ToggleLock();
      const newLockState = !state.isLocked;
      setState((prev) => ({ ...prev, isLocked: newLockState }));
      EventsEmit("update-tray-lock", newLockState);
    } catch (error) {
      console.error("Failed to toggle lock:", error);
    }
  }, [state.isLocked]);

  const handleStopAll = useCallback(async () => {
    try {
      await StopAllNircmd();
      if (state.isLocked) {
        setState((prev) => ({ ...prev, isLocked: false }));
        EventsEmit("update-tray-lock", false);
      }
    } catch (error) {
      console.error("Failed to stop processes:", error);
    }
  }, [state.isLocked]);

  return (
    <div className="container mx-auto p-2 max-w-md">
      <div className="space-y-4 px-[4px]">
        <Button
          onClick={handleLockToggle}
          className="w-full h-10 text-base font-bold"
          variant={state.isLocked ? "primary-dark" : "primary"}
        >
          {state.isLocked
            ? "Unlock Microphone Volume"
            : "Lock Microphone Volume"}
        </Button>

        <div className="space-y-2">
          <label className="flex">Volume: {state.volume}%</label>
          <Slider
            min={0}
            step={1}
            max={100}
            value={[state.volume]}
            onValueChange={handleVolumeChange}
            disabled={state.isLocked}
            className="w-full rounded-lg"
            color="gray"
          />
        </div>

        <Button
          onClick={handleStopAll}
          variant="danger"
          className="w-full h-10 text-base font-bold"
        >
          Stop All Nircmd Processes
        </Button>
      </div>
    </div>
  );
}

export default App;
