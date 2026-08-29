import { Modal, type ModalProps } from "antd";
import type { KeyboardEvent as ReactKeyboardEvent, PointerEvent as ReactPointerEvent, ReactNode } from "react";
import { useRef, useState } from "react";

export interface AkModalSize {
  width: number;
  height: number;
}

export interface AkModalResizeConfig extends AkModalSize {
  minWidth?: number;
  minHeight?: number;
  maxWidth?: number;
  maxHeight?: number;
  viewportPadding?: number;
  disabled?: boolean;
  onResize: (size: AkModalSize) => void;
}

export interface AkModalProps extends Omit<ModalProps, "width"> {
  width?: ModalProps["width"];
  resizable?: AkModalResizeConfig;
  resizeHandleLabel?: string;
}

interface ResizeSession {
  pointerId: number;
  startX: number;
  startY: number;
  size: AkModalSize;
  maximumWidth: number;
  maximumHeight: number;
}

const clamp = (value: number, minimum: number, maximum: number) => Math.min(Math.max(value, minimum), maximum);

export function AkModal({ modalRender, resizable, resizeHandleLabel, width, ...modalProps }: AkModalProps) {
  const frameRef = useRef<HTMLDivElement | null>(null);
  const resizeSessionRef = useRef<ResizeSession | null>(null);
  const [resizing, setResizing] = useState(false);
  const resizeEnabled = Boolean(resizable && !resizable.disabled);

  const commitResize = (widthValue: number, heightValue: number, maximumWidth?: number, maximumHeight?: number) => {
    if (!resizable || resizable.disabled) return;
    const minWidth = resizable.minWidth ?? 320;
    const minHeight = resizable.minHeight ?? 240;
    const maxWidth = Math.max(minWidth, maximumWidth ?? resizable.maxWidth ?? Number.MAX_SAFE_INTEGER);
    const maxHeight = Math.max(minHeight, maximumHeight ?? resizable.maxHeight ?? Number.MAX_SAFE_INTEGER);
    resizable.onResize({
      width: clamp(widthValue, Math.min(minWidth, maxWidth), maxWidth),
      height: clamp(heightValue, Math.min(minHeight, maxHeight), maxHeight),
    });
  };

  const startResize = (event: ReactPointerEvent<HTMLButtonElement>) => {
    if (!resizable || resizable.disabled || event.button !== 0) return;
    event.preventDefault();
    event.stopPropagation();
    event.currentTarget.setPointerCapture(event.pointerId);
    const bounds = frameRef.current?.getBoundingClientRect();
    const viewportPadding = resizable.viewportPadding ?? 8;
    const viewportWidth = typeof window === "undefined" ? resizable.width : window.innerWidth;
    const viewportHeight = typeof window === "undefined" ? resizable.height : window.innerHeight;
    const maximumWidth = resizable.maxWidth ?? Math.max(resizable.minWidth ?? 320, viewportWidth - (bounds?.left ?? 0) - viewportPadding);
    const maximumHeight = resizable.maxHeight ?? Math.max(resizable.minHeight ?? 240, viewportHeight - (bounds?.top ?? 0) - viewportPadding);
    resizeSessionRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      size: { width: resizable.width, height: resizable.height },
      maximumWidth,
      maximumHeight,
    };
    setResizing(true);
  };

  const dragResize = (event: ReactPointerEvent<HTMLButtonElement>) => {
    const session = resizeSessionRef.current;
    if (session?.pointerId !== event.pointerId) return;
    commitResize(
      session.size.width + event.clientX - session.startX,
      session.size.height + event.clientY - session.startY,
      session.maximumWidth,
      session.maximumHeight,
    );
  };

  const endResize = (event: ReactPointerEvent<HTMLButtonElement>) => {
    if (resizeSessionRef.current?.pointerId !== event.pointerId) return;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
    resizeSessionRef.current = null;
    setResizing(false);
  };

  const resizeWithKeyboard = (event: ReactKeyboardEvent<HTMLButtonElement>) => {
    if (!resizable || resizable.disabled) return;
    const step = event.shiftKey ? 32 : 16;
    let nextWidth = resizable.width;
    let nextHeight = resizable.height;
    if (event.key === "ArrowLeft") nextWidth -= step;
    else if (event.key === "ArrowRight") nextWidth += step;
    else if (event.key === "ArrowUp") nextHeight -= step;
    else if (event.key === "ArrowDown") nextHeight += step;
    else return;
    event.preventDefault();
    commitResize(nextWidth, nextHeight);
  };

  const renderFrame = (node: ReactNode) => {
    const content = modalRender ? modalRender(node) : node;
    if (!resizable) return content;
    return <div ref={frameRef} className="ak-modal-resizable-frame" style={{ height: resizable.height }}>
      {content}
      {resizeEnabled ? <button
        type="button"
        className="ak-modal-resize-handle"
        aria-label={resizeHandleLabel}
        data-resizing={resizing ? "true" : "false"}
        onPointerDown={startResize}
        onPointerMove={dragResize}
        onPointerUp={endResize}
        onPointerCancel={endResize}
        onKeyDown={resizeWithKeyboard}
      /> : null}
    </div>;
  };

  const modalWidth = resizable?.width ?? width;
  return <Modal {...modalProps} {...(modalWidth === undefined ? {} : { width: modalWidth })} modalRender={renderFrame} />;
}
