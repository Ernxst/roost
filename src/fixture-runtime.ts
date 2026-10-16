// The development server resolves "mygo-runtime" to this module (see
// vite.config.ts), so with a fixture chosen in the URL every call the page
// makes is answered from src/fixtures.ts instead of Go. Without one it is the
// runtime itself. Builds never include it.
import { call as runtimeCall, event as runtimeEvent, type GoEvent } from "mygo-runtime";
import { fixtureMode } from "./fixture-mode";

export * from "mygo-runtime";

const fixtures = fixtureMode === "demo" ? undefined : import("./fixtures");

export function call<T = unknown>(method: string, ...args: unknown[]): Promise<T> {
  if (!fixtures) return runtimeCall<T>(method, ...args);
  return fixtures.then((f) => f.answer(method, args) as T);
}

const listeners = new Map<string, Set<(payload: unknown) => void>>();

/** Sends an event to the page as Go would, for fixtures that act out a sync. */
export function emit(name: string, payload: unknown) {
  for (const listener of [...(listeners.get(name) ?? [])]) listener(payload);
}

export function event<T>(name: string): GoEvent<T> {
  if (!fixtures) return runtimeEvent<T>(name);
  const on = (listener: (payload: T) => void) => {
    const set = listeners.get(name) ?? new Set();
    listeners.set(name, set);
    const l = listener as (payload: unknown) => void;
    set.add(l);
    return () => void set.delete(l);
  };
  return {
    name,
    on,
    once: (listener) => {
      const off = on((payload) => (off(), listener(payload)));
      return off;
    },
  };
}
