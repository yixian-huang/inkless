import type { AIMetaFeedbackKind } from "./applyAIMeta";

const EVENTS_KEY = "inkless.aiMeta.events";
const FEEDBACK_KEY = "inkless.aiMeta.feedback";
const MAX_EVENTS = 200;

export type AIMetaEventType =
  | "open"
  | "generate_ok"
  | "generate_err"
  | "apply"
  | "dismiss"
  | "feedback";

export type AIMetaEvent = {
  type: AIMetaEventType;
  at: number;
  model?: string;
  applied?: number;
  warningCodes?: string[];
  warnCount?: number;
  feedback?: AIMetaFeedbackKind;
  mode?: string;
  sourceLang?: string;
};

export type { AIMetaFeedbackKind };

function readEvents(): AIMetaEvent[] {
  try {
    const raw = JSON.parse(localStorage.getItem(EVENTS_KEY) || "[]");
    return Array.isArray(raw) ? (raw as AIMetaEvent[]) : [];
  } catch {
    return [];
  }
}

function writeEvents(events: AIMetaEvent[]): void {
  try {
    localStorage.setItem(EVENTS_KEY, JSON.stringify(events.slice(-MAX_EVENTS)));
  } catch {
    /* ignore */
  }
}

export function recordAIMetaEvent(event: Omit<AIMetaEvent, "at"> & { at?: number }): void {
  const entry: AIMetaEvent = { ...event, at: event.at ?? Date.now() };
  const next = [...readEvents(), entry];
  writeEvents(next);

  // Keep legacy feedback key in sync for older readers
  if (event.type === "feedback" && event.feedback) {
    try {
      const prev = JSON.parse(localStorage.getItem(FEEDBACK_KEY) || "[]") as unknown[];
      const fb = {
        kind: event.feedback,
        at: entry.at,
        model: event.model,
        applied: event.applied,
      };
      localStorage.setItem(
        FEEDBACK_KEY,
        JSON.stringify([...(Array.isArray(prev) ? prev : []), fb].slice(-50)),
      );
    } catch {
      /* ignore */
    }
  }
}

export type AIMetaStatsSummary = {
  opens: number;
  generateOk: number;
  generateErr: number;
  applies: number;
  dismisses: number;
  /** applies / generateOk (0 if no successful generates) */
  applyRate: number;
  feedback: Record<AIMetaFeedbackKind, number>;
  /** fraction of applies that had at least one warn */
  applyWithWarnRate: number;
  windowSize: number;
};

/** Summarize local telemetry for console / future admin panel. */
export function summarizeAIMetaStats(events?: AIMetaEvent[]): AIMetaStatsSummary {
  const list = events ?? readEvents();
  const feedback: Record<AIMetaFeedbackKind, number> = {
    useful: 0,
    needs_edit: 0,
    unusable: 0,
  };
  let opens = 0;
  let generateOk = 0;
  let generateErr = 0;
  let applies = 0;
  let dismisses = 0;
  let applyWithWarn = 0;

  for (const e of list) {
    switch (e.type) {
      case "open":
        opens += 1;
        break;
      case "generate_ok":
        generateOk += 1;
        break;
      case "generate_err":
        generateErr += 1;
        break;
      case "apply":
        applies += 1;
        if ((e.warnCount || 0) > 0) applyWithWarn += 1;
        break;
      case "dismiss":
        dismisses += 1;
        break;
      case "feedback":
        if (e.feedback && feedback[e.feedback] !== undefined) {
          feedback[e.feedback] += 1;
        }
        break;
      default:
        break;
    }
  }

  return {
    opens,
    generateOk,
    generateErr,
    applies,
    dismisses,
    applyRate: generateOk > 0 ? applies / generateOk : 0,
    feedback,
    applyWithWarnRate: applies > 0 ? applyWithWarn / applies : 0,
    windowSize: list.length,
  };
}

/** Expose on window for local debugging: `__inklessAIMetaStats()` */
export function installAIMetaStatsDebug(): void {
  if (typeof window === "undefined") return;
  (window as unknown as { __inklessAIMetaStats?: () => AIMetaStatsSummary }).__inklessAIMetaStats =
    () => summarizeAIMetaStats();
}
