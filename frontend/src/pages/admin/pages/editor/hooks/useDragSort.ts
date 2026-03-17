import { useRef, useCallback } from "react";

export function useDragSort(onReorder: (from: number, to: number) => void) {
  const dragIndexRef = useRef<number | null>(null);

  const makeDragHandlers = useCallback(
    (index: number) => ({
      onDragStart: (e: React.DragEvent) => {
        dragIndexRef.current = index;
        e.dataTransfer.effectAllowed = "move";
      },
      onDragOver: (e: React.DragEvent) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
      },
      onDrop: (e: React.DragEvent) => {
        e.preventDefault();
        const from = dragIndexRef.current;
        if (from !== null && from !== index) onReorder(from, index);
        dragIndexRef.current = null;
      },
      onDragEnd: () => {
        dragIndexRef.current = null;
      },
    }),
    [onReorder],
  );

  return { makeDragHandlers };
}
