import { useState, useRef, useCallback, useEffect, type RefObject } from 'react';

interface UseDropdownPositionOptions {
  minWidth?: number;
  onClose?: () => void;
}

interface UseDropdownPositionReturn {
  open: boolean;
  pos: { top: number; left: number; width: number };
  triggerRef: RefObject<HTMLButtonElement | null>;
  dropdownRef: RefObject<HTMLDivElement | null>;
  openDropdown: () => void;
  closeDropdown: () => void;
  toggleDropdown: () => void;
}

export function useDropdownPosition(
  options: UseDropdownPositionOptions = {},
): UseDropdownPositionReturn {
  const { minWidth = 260, onClose } = options;
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ top: 0, left: 0, width: minWidth });
  const triggerRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const openDropdown = useCallback(() => {
    if (triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      const w = Math.max(rect.width, minWidth);
      let left = rect.left;
      if (left + w > window.innerWidth - 8) left = window.innerWidth - w - 8;
      setPos({ top: rect.bottom + 4, left, width: w });
    }
    setOpen(true);
  }, [minWidth]);

  const closeDropdown = useCallback(() => {
    setOpen(false);
    onClose?.();
    setTimeout(() => triggerRef.current?.focus(), 0);
  }, [onClose]);

  const toggleDropdown = useCallback(() => {
    if (open) closeDropdown();
    else openDropdown();
  }, [open, closeDropdown, openDropdown]);

  useEffect(() => {
    if (!open) return;
    const onMouseDown = (e: MouseEvent) => {
      const t = e.target as Node;
      if (triggerRef.current?.contains(t) || dropdownRef.current?.contains(t)) return;
      closeDropdown();
    };
    const onScroll = (e: Event) => {
      if (dropdownRef.current?.contains(e.target as Node)) return;
      closeDropdown();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') closeDropdown();
    };
    document.addEventListener('mousedown', onMouseDown);
    document.addEventListener('keydown', onKey);
    window.addEventListener('scroll', onScroll, true);
    return () => {
      document.removeEventListener('mousedown', onMouseDown);
      document.removeEventListener('keydown', onKey);
      window.removeEventListener('scroll', onScroll, true);
    };
  }, [open, closeDropdown]);

  return { open, pos, triggerRef, dropdownRef, openDropdown, closeDropdown, toggleDropdown };
}
