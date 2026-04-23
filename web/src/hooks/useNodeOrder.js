import { useState, useEffect, useCallback } from 'react';
import { arrayMove } from '@dnd-kit/sortable';

const LS_ORDER = 'mon_node_order';
const LS_PINNED = 'mon_node_pinned';

export function useNodeOrder(nodes) {
  const [order, setOrder] = useState(() => {
    try { return JSON.parse(localStorage.getItem(LS_ORDER)) || []; } catch { return []; }
  });
  const [pinned, setPinned] = useState(() => {
    try { return JSON.parse(localStorage.getItem(LS_PINNED)) || []; } catch { return []; }
  });

  useEffect(() => {
    localStorage.setItem(LS_ORDER, JSON.stringify(order));
  }, [order]);

  useEffect(() => {
    localStorage.setItem(LS_PINNED, JSON.stringify(pinned));
  }, [pinned]);

  const sorted = useCallback(() => {
    if (!nodes) return [];
    const names = nodes.map(n => n.name);
    // Merge saved order with current nodes (new nodes go to end)
    const known = order.filter(n => names.includes(n));
    const newOnes = names.filter(n => !known.includes(n));
    const fullOrder = [...known, ...newOnes];
    const nodeMap = Object.fromEntries(nodes.map(n => [n.name, n]));
    const pinnedSet = new Set(pinned);
    const pinnedNodes = fullOrder.filter(n => pinnedSet.has(n) && nodeMap[n]).map(n => nodeMap[n]);
    const unpinnedNodes = fullOrder.filter(n => !pinnedSet.has(n) && nodeMap[n]).map(n => nodeMap[n]);
    return [...pinnedNodes, ...unpinnedNodes];
  }, [nodes, order, pinned]);

  const handleDragEnd = useCallback((event) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    setOrder(prev => {
      const names = nodes?.map(n => n.name) || [];
      const known = prev.filter(n => names.includes(n));
      const newOnes = names.filter(n => !known.includes(n));
      const full = [...known, ...newOnes];
      const oldIdx = full.indexOf(active.id);
      const newIdx = full.indexOf(over.id);
      if (oldIdx === -1 || newIdx === -1) return prev;
      return arrayMove(full, oldIdx, newIdx);
    });
  }, [nodes]);

  const togglePin = useCallback((name) => {
    setPinned(prev =>
      prev.includes(name) ? prev.filter(n => n !== name) : [...prev, name]
    );
  }, []);

  return { sorted: sorted(), handleDragEnd, togglePin, pinned };
}
