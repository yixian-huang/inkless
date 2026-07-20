import { useMemo } from "react";
import { diffLines, htmlToPlainText, type DiffLine } from "@/lib/textDiff";

export function DiffView({ lines }: { lines: DiffLine[] }) {
  if (lines.length === 0) {
    return <div className="text-sm text-gray-400 py-8 text-center">无差异</div>;
  }
  const changed = lines.filter((l) => l.op !== "equal").length;
  return (
    <div className="font-mono text-xs leading-5 border border-gray-200 rounded-lg overflow-hidden">
      <div className="px-3 py-1.5 bg-gray-50 border-b border-gray-200 text-gray-500 flex justify-between">
        <span>行级差异</span>
        <span>{changed === 0 ? "完全相同" : `${changed} 处变更行`}</span>
      </div>
      <div className="max-h-[50vh] overflow-auto">
        {lines.map((line, idx) => {
          const bg =
            line.op === "add"
              ? "bg-green-50 text-green-900"
              : line.op === "remove"
                ? "bg-red-50 text-red-900"
                : "bg-white text-gray-700";
          const prefix = line.op === "add" ? "+" : line.op === "remove" ? "−" : " ";
          return (
            <div key={idx} className={`flex ${bg} border-b border-gray-50 last:border-0`}>
              <span className="w-10 flex-shrink-0 text-right pr-2 text-gray-400 select-none tabular-nums">
                {line.leftLine ?? ""}
              </span>
              <span className="w-10 flex-shrink-0 text-right pr-2 text-gray-400 select-none tabular-nums">
                {line.rightLine ?? ""}
              </span>
              <span className="w-4 flex-shrink-0 text-center select-none opacity-70">{prefix}</span>
              <pre className="flex-1 whitespace-pre-wrap break-all py-0.5 pr-2 m-0 font-mono text-xs">
                {line.text || " "}
              </pre>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function FieldDiff({
  label,
  left,
  right,
  asHtml,
}: {
  label: string;
  left: string;
  right: string;
  asHtml?: boolean;
}) {
  const leftText = asHtml ? htmlToPlainText(left) : left;
  const rightText = asHtml ? htmlToPlainText(right) : right;
  const same = leftText === rightText;
  const lines = useMemo(
    () => (same ? [] : diffLines(leftText, rightText)),
    [leftText, rightText, same],
  );

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <h4 className="text-sm font-semibold text-gray-800">{label}</h4>
        {same ? (
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-gray-100 text-gray-500">相同</span>
        ) : (
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-amber-100 text-amber-800">有变更</span>
        )}
      </div>
      {same ? (
        <div className="text-xs text-gray-500 bg-gray-50 rounded p-2 max-h-24 overflow-auto whitespace-pre-wrap">
          {leftText || <span className="italic text-gray-400">（空）</span>}
        </div>
      ) : (
        <DiffView lines={lines} />
      )}
    </div>
  );
}

