import type {
  ExtensionAPI,
  ExtensionContext,
} from "@earendil-works/pi-coding-agent";
import { readFileSync } from "node:fs";
import { mkdir, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { Ajv2020 } from "ajv/dist/2020.js";
import type { SessionRecord } from "./generated/session-record.js";

const SCHEMA_PATH = join(
  dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "pi-session-schema",
  "schema",
  "session-record.schema.json",
);
const recordSchema = JSON.parse(readFileSync(SCHEMA_PATH, "utf8"));

const ajv = new Ajv2020({
  allErrors: true,
  strict: true,
  strictRequired: false,
});
const validateRecord = ajv.compile<SessionRecord>(recordSchema);

type Rating = Omit<
  SessionRecord["outcome"],
  "rated" | "outcome" | "ratedAt" | "source"
> &
  Required<Pick<SessionRecord["outcome"], "outcome" | "ratedAt" | "source">>;

const SCHEMA_VERSION = 2;
const RECORD_DIR =
  process.env.PI_SESSION_RECORDS ?? join(homedir(), ".pi", "agent", "records");

const RATING_ENTRY = "session-rating";

const MIN_TURNS_TO_ASK = Number(process.env.PI_RATE_PROMPT_MIN ?? 3);

const RATING_CHOICES = [
  { label: "5: fully, no corrections", value: 5 },
  { label: "4: yes, minor fixes", value: 4 },
  { label: "3: partly, needed rework", value: 3 },
  { label: "2: mostly not", value: 2 },
  { label: "1: no", value: 1 },
] as const;

export default function (pi: ExtensionAPI) {
  pi.registerCommand("rate", {
    description:
      "Rate this session (stored in the session, written out on exit)",
    handler: async (args, ctx) => {
      const scale = ["1", "2", "3", "4", "5"];
      const outcome = await ctx.ui.select(
        "Did it accomplish the task? (1 no - 5 fully)",
        scale,
      );
      if (!outcome) return;

      const rework = await ctx.ui.select(
        "Rework needed? (1 lots - 5 none)",
        scale,
      );
      if (!rework) return;

      const autonomy = await ctx.ui.select(
        "Intervention needed? (1 constant - 5 none)",
        scale,
      );
      if (!autonomy) return;

      const note = args?.trim() || (await ctx.ui.input("Note (optional)", ""));

      pi.appendEntry(RATING_ENTRY, {
        outcome: Number(outcome),
        rework: Number(rework),
        autonomy: Number(autonomy),
        note: note || undefined,
        ratedAt: Date.now(),
        source: "command",
      } satisfies Rating);

      ctx.ui.notify("Rating recorded", "info");
    },
  });

  async function promptForRating(ctx: ExtensionContext) {
    const choice = await ctx.ui.select(
      "Did this session accomplish the task?",
      [...RATING_CHOICES.map((c) => c.label), "Skip"],
    );

    const selected = RATING_CHOICES.find((c) => c.label === choice);
    if (!selected) return;

    pi.appendEntry(RATING_ENTRY, {
      outcome: selected.value,
      ratedAt: Date.now(),
      source: "idle-prompt",
    } satisfies Rating);
  }

  function findRating(ctx: ExtensionContext): Rating | undefined {
    let found: Rating | undefined;
    for (const entry of ctx.sessionManager.getEntries() as any[]) {
      if (entry.type === "custom" && entry.customType === RATING_ENTRY) {
        found = entry.data as Rating;
      }
    }

    return found;
  }

  async function writeRecord(ctx: ExtensionContext, endReason: string) {
    const sessionFile = ctx.sessionManager.getSessionFile();
    if (!sessionFile) return;

    const sessionId = ctx.sessionManager.getSessionId();
    const entries = ctx.sessionManager.getEntries();

    const record: SessionRecord = {
      schemaVersion: SCHEMA_VERSION,
      sessionId,
      sessionFile,
      cwd: ctx.cwd,
      endedAt: Date.now(),
      endReason,

      agentVersion: process.env.PI_AGENT_VERSION ?? null,
      taskId: process.env.PI_TASK_ID ?? null,

      models: [],

      usage: {
        input: 0,
        output: 0,
        cacheRead: 0,
        cacheWrite: 0,
        costUsd: 0,
        unmeteredMessages: 0,
      },

      activity: {
        assistantMessages: 0,
        userMessages: 0,
        toolCalls: 0,
        toolErrors: 0,
        byTool: {},
        compactions: 0,
      },

      outcome: { rated: false },
    };

    const modelCounts = new Map<string, number>();
    let firstTimestamp: number | undefined;
    let lastRating: Rating | undefined;
    for (const entry of entries as any[]) {
      if (typeof entry.timestamp === "number") {
        firstTimestamp = Math.min(
          firstTimestamp ?? entry.timestamp,
          entry.timestamp,
        );
      }

      if (entry.type === "custom" && entry.customType === RATING_ENTRY) {
        lastRating = entry.data as Rating;
        continue;
      }

      if (entry.type === "compaction" || entry.customType === "compaction") {
        record.activity.compactions += 1;
        continue;
      }

      if (entry.type !== "message") continue;

      const message = entry.message;
      if (message.role === "user") {
        record.activity.userMessages += 1;
      } else if (message.role === "toolResult") {
        record.activity.toolCalls += 1;
        if (message.isError) record.activity.toolErrors += 1;

        const name = message.toolName ?? "unknown";
        record.activity.byTool[name] = (record.activity.byTool[name] ?? 0) + 1;
      } else if (message.role === "assistant") {
        record.activity.assistantMessages += 1;

        const key =
          message.provider && message.model
            ? `${message.provider}/${message.model}`
            : (message.model ?? "unknown");
        modelCounts.set(key, (modelCounts.get(key) ?? 0) + 1);

        const usage = message.usage;
        if (!usage) {
          record.usage.unmeteredMessages += 1;
          continue;
        }

        record.usage.input += usage.input ?? 0;
        record.usage.output += usage.output ?? 0;
        record.usage.cacheRead += usage.cacheRead ?? 0;
        record.usage.cacheWrite += usage.cacheWrite ?? 0;
        record.usage.costUsd += usage.cost?.total ?? 0;
      }
    }

    record.models = [...modelCounts].map(([key, assistantMessages]) => ({
      key,
      assistantMessages,
    }));
    record.startedAt = firstTimestamp;
    if (firstTimestamp) record.durationMs = record.endedAt - firstTimestamp;
    if (lastRating) record.outcome = { rated: true, ...lastRating };

    if (!validateRecord(record)) {
      throw new Error(
        `record failed schema validation: ${ajv.errorsText(validateRecord.errors)}`,
      );
    }

    await mkdir(RECORD_DIR, { recursive: true });
    await writeFile(
      join(RECORD_DIR, `${sessionId}.json`),
      `${JSON.stringify(record, null, 2)}\n`,
      "utf8",
    );
  }

  let settledCount = 0;
  let asked = false;

  pi.on("agent_settled", async (_event, ctx) => {
    if (asked) return;
    if (!ctx.hasUI || ctx.mode !== "tui") return;

    settledCount += 1;
    if (settledCount < MIN_TURNS_TO_ASK) return;
    if (findRating(ctx)) return;

    asked = true;

    try {
      await promptForRating(ctx);
    } catch {}
  });

  pi.on("session_shutdown", async (event, ctx) => {
    if (event.reason === "reload") return;

    try {
      await writeRecord(ctx, event.reason);
    } catch (error) {
      ctx.ui.notify(
        `pi-session-record failed: ${(error as Error).message}`,
        "error",
      );
    }
  });
}
