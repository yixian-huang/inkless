import { TOAST_MS } from "./constants";

/** Fire-and-forget message that auto-clears via the setter. */
export function toast(setMsg: (s: string) => void, msg: string, ms = TOAST_MS) {
  setMsg(msg);
  window.setTimeout(() => setMsg(""), ms);
}
