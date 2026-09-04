import { ReloadOutlined, UndoOutlined } from "@ant-design/icons";
import { Button, Tooltip, Typography } from "antd";
import {
  forwardRef,
  useEffect,
  useId,
  useImperativeHandle,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type PointerEvent,
} from "react";
import { useTranslation } from "react-i18next";

import type {
  AdminLoginCaptchaAnswer,
  AdminLoginCaptchaImage,
  AdminLoginCaptchaPoint,
  AdminLoginCaptchaResponse,
  AdminLoginCaptchaType,
} from "../generated/api/types.gen";
import enCaptcha from "../locales/en-US/captcha.json";
import zhCaptcha from "../locales/zh-CN/captcha.json";
import { registerAdminTranslationCatalog } from "../shared/i18n";
import "./AkInteractiveCaptcha.css";

registerAdminTranslationCatalog("zh-CN", zhCaptcha);
registerAdminTranslationCatalog("en-US", enCaptcha);

export type AkCaptchaType = AdminLoginCaptchaType;
export type AkCaptchaPoint = AdminLoginCaptchaPoint;
export type AkCaptchaImage = AdminLoginCaptchaImage;
export type AkInteractiveCaptchaChallenge = AdminLoginCaptchaResponse["data"];
export type AkInteractiveCaptchaResponse = AdminLoginCaptchaAnswer;

export interface AkInteractiveCaptchaHandle {
  focus: () => void;
}

export interface AkInteractiveCaptchaProps {
  autoFocus?: boolean;
  challenge: AkInteractiveCaptchaChallenge;
  disabled?: boolean;
  error?: string | undefined;
  onChange: (response: AkInteractiveCaptchaResponse | null) => void;
  onRefresh: () => void;
  value: AkInteractiveCaptchaResponse | null;
}

interface CaptchaViewport {
  height: number;
  left: number;
  top: number;
  width: number;
}

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(Math.max(value, minimum), maximum);
}

export function captchaPointFromClient(
  clientX: number,
  clientY: number,
  viewport: CaptchaViewport,
  image: Pick<AkCaptchaImage, "height" | "width">,
  item: Pick<AkCaptchaImage, "height" | "width"> = { height: 0, width: 0 },
): AkCaptchaPoint {
  const scaleX = viewport.width > 0 ? image.width / viewport.width : 1;
  const scaleY = viewport.height > 0 ? image.height / viewport.height : 1;
  return {
    x: clamp(
      Math.round((clientX - viewport.left) * scaleX - item.width / 2),
      0,
      Math.max(0, image.width - item.width),
    ),
    y: clamp(
      Math.round((clientY - viewport.top) * scaleY - item.height / 2),
      0,
      Math.max(0, image.height - item.height),
    ),
  };
}

function imageSource(image: AkCaptchaImage) {
  return `data:${image.mime_type};base64,${image.base64}`;
}

function pointStyle(point: AkCaptchaPoint, image: AkCaptchaImage): CSSProperties {
  return {
    left: `${String((point.x / Math.max(1, image.width)) * 100)}%`,
    top: `${String((point.y / Math.max(1, image.height)) * 100)}%`,
  };
}

function tileStyle(
  point: AkCaptchaPoint,
  tile: AkCaptchaImage,
  image: AkCaptchaImage,
): CSSProperties {
  return {
    height: `${String((tile.height / Math.max(1, image.height)) * 100)}%`,
    left: `${String((point.x / Math.max(1, image.width)) * 100)}%`,
    top: `${String((point.y / Math.max(1, image.height)) * 100)}%`,
    width: `${String((tile.width / Math.max(1, image.width)) * 100)}%`,
  };
}

function imageFrameStyle(image: AkCaptchaImage): CSSProperties {
  return {
    aspectRatio: `${String(Math.max(1, image.width))} / ${String(Math.max(1, image.height))}`,
    maxWidth: image.width,
  };
}

interface ModeProps<TChallenge extends AkInteractiveCaptchaChallenge> {
  challenge: TChallenge;
  disabled: boolean;
  instructionId: string;
  onChange: AkInteractiveCaptchaProps["onChange"];
  value: AkInteractiveCaptchaResponse | null;
}

function ClickCaptcha({
  challenge,
  disabled,
  instructionId,
  onChange,
  value,
}: ModeProps<Extract<AkInteractiveCaptchaChallenge, { type: "click" }>>) {
  const { t } = useTranslation();
  const requiredPoints = clamp(Math.round(challenge.required_points), 1, 4);
  const [cursor, setCursor] = useState<AkCaptchaPoint>(() => ({
    x: Math.floor(challenge.image.width / 2),
    y: Math.floor(challenge.image.height / 2),
  }));
  const [points, setPoints] = useState<AkCaptchaPoint[]>(() =>
    value?.type === "click" ? value.points.slice(0, requiredPoints) : [],
  );

  useEffect(() => {
    if (value?.type === "click") {
      setPoints(value.points.slice(0, requiredPoints));
    }
  }, [requiredPoints, value]);

  const setNextPoints = (next: AkCaptchaPoint[]) => {
    setPoints(next);
    onChange(
      next.length === requiredPoints
        ? { points: next, type: "click" }
        : null,
    );
  };
  const addPoint = (point: AkCaptchaPoint) => {
    if (disabled || points.length >= requiredPoints) return;
    setCursor(point);
    setNextPoints([...points, point]);
  };
  const updateCursor = (x: number, y: number) => {
    setCursor({
      x: clamp(x, 0, Math.max(0, challenge.image.width - 1)),
      y: clamp(y, 0, Math.max(0, challenge.image.height - 1)),
    });
  };
  const handlePointer = (event: PointerEvent<HTMLDivElement>, commit: boolean) => {
    if (disabled) return;
    const rect = event.currentTarget.getBoundingClientRect();
    const point = captchaPointFromClient(event.clientX, event.clientY, rect, challenge.image);
    updateCursor(point.x, point.y);
    if (commit) addPoint(point);
  };
  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (disabled) return;
    const step = event.shiftKey ? 10 : 1;
    switch (event.key) {
      case "ArrowLeft":
        event.preventDefault();
        updateCursor(cursor.x - step, cursor.y);
        break;
      case "ArrowRight":
        event.preventDefault();
        updateCursor(cursor.x + step, cursor.y);
        break;
      case "ArrowUp":
        event.preventDefault();
        updateCursor(cursor.x, cursor.y - step);
        break;
      case "ArrowDown":
        event.preventDefault();
        updateCursor(cursor.x, cursor.y + step);
        break;
      case "Enter":
      case " ":
        event.preventDefault();
        addPoint(cursor);
        break;
      case "Backspace":
      case "Delete":
        event.preventDefault();
        setNextPoints(points.slice(0, -1));
        break;
    }
  };

  return (
    <div className="ak-captcha-mode ak-captcha-click-mode">
      <div className="ak-captcha-prompt">
        <Typography.Text>{t("auth.login.captcha.click.prompt")}</Typography.Text>
        <img
          alt={t("auth.login.captcha.click.prompt_alt")}
          className="ak-captcha-prompt-image"
          draggable={false}
          src={imageSource(challenge.prompt_image)}
        />
      </div>
      <div
        aria-describedby={instructionId}
        aria-disabled={disabled}
        aria-keyshortcuts="ArrowLeft ArrowRight ArrowUp ArrowDown Enter Space Backspace Delete"
        aria-label={t("auth.login.captcha.click.control")}
        className="ak-captcha-image-frame ak-captcha-click-board"
        data-ak-captcha-control
        role="group"
        style={imageFrameStyle(challenge.image)}
        tabIndex={disabled ? -1 : 0}
        onKeyDown={handleKeyDown}
        onPointerDown={(event) => {
          event.currentTarget.focus();
          handlePointer(event, true);
        }}
        onPointerMove={(event) => {
          handlePointer(event, false);
        }}
      >
        <img
          alt={t("auth.login.captcha.click.image_alt")}
          className="ak-captcha-main-image"
          draggable={false}
          src={imageSource(challenge.image)}
        />
        {points.map((point, index) => (
          <span
            aria-hidden="true"
            className="ak-captcha-click-marker"
            key={`${String(point.x)}-${String(point.y)}-${String(index)}`}
            style={pointStyle(point, challenge.image)}
          >
            {index + 1}
          </span>
        ))}
        <span
          aria-hidden="true"
          className="ak-captcha-keyboard-cursor"
          style={pointStyle(cursor, challenge.image)}
        />
      </div>
      <div className="ak-captcha-mode-footer">
        <span aria-live="polite" role="status">
          {points.length === requiredPoints
            ? t("auth.login.captcha.completed")
            : t("auth.login.captcha.click.progress", {
                count: points.length,
                total: requiredPoints,
              })}
        </span>
        <Tooltip title={t("auth.login.captcha.click.undo")}>
          <Button
            aria-label={t("auth.login.captcha.click.undo")}
            disabled={disabled || points.length === 0}
            icon={<UndoOutlined aria-hidden="true" />}
            onClick={() => {
              setNextPoints(points.slice(0, -1));
            }}
          />
        </Tooltip>
      </div>
    </div>
  );
}

function SlideCaptcha({
  challenge,
  disabled,
  instructionId,
  onChange,
  value,
}: ModeProps<Extract<AkInteractiveCaptchaChallenge, { type: "slide" }>>) {
  const { t } = useTranslation();
  const maximumX = Math.max(0, challenge.image.width - challenge.tile_image.width);
  const fixedY = clamp(
    challenge.initial_point.y,
    0,
    Math.max(0, challenge.image.height - challenge.tile_image.height),
  );
  const [position, setPosition] = useState<AkCaptchaPoint>(() =>
    value?.type === "slide"
      ? value.point
      : { x: clamp(challenge.initial_point.x, 0, maximumX), y: fixedY },
  );

  useEffect(() => {
    setPosition(
      value?.type === "slide"
        ? value.point
        : { x: clamp(challenge.initial_point.x, 0, maximumX), y: fixedY },
    );
  }, [challenge.initial_point.x, fixedY, maximumX, value]);
  const commit = (x: number) => {
    const point = { x: clamp(Math.round(x), 0, maximumX), y: fixedY };
    setPosition(point);
    onChange({ point, type: "slide" });
  };

  return (
    <div className="ak-captcha-mode">
      <div
        className="ak-captcha-image-frame"
        style={imageFrameStyle(challenge.image)}
      >
        <img
          alt={t("auth.login.captcha.slide.image_alt")}
          className="ak-captcha-main-image"
          draggable={false}
          src={imageSource(challenge.image)}
        />
        <img
          alt=""
          aria-hidden="true"
          className="ak-captcha-tile-image"
          draggable={false}
          src={imageSource(challenge.tile_image)}
          style={tileStyle(position, challenge.tile_image, challenge.image)}
        />
      </div>
      <label className="ak-captcha-range-label">
        <span>{t("auth.login.captcha.slide.control")}</span>
        <input
          aria-describedby={instructionId}
          className="ak-captcha-range"
          data-ak-captcha-control
          disabled={disabled || maximumX === 0}
          max={maximumX}
          min={0}
          step={1}
          type="range"
          value={position.x}
          onChange={(event) => {
            commit(event.currentTarget.valueAsNumber);
          }}
        />
      </label>
      <span aria-live="polite" role="status">
        {value?.type === "slide"
          ? t("auth.login.captcha.completed")
          : t("auth.login.captcha.pending")}
      </span>
    </div>
  );
}

function DragCaptcha({
  challenge,
  disabled,
  instructionId,
  onChange,
  value,
}: ModeProps<Extract<AkInteractiveCaptchaChallenge, { type: "drag" }>>) {
  const { t } = useTranslation();
  const maximumX = Math.max(0, challenge.image.width - challenge.tile_image.width);
  const maximumY = Math.max(0, challenge.image.height - challenge.tile_image.height);
  const [position, setPosition] = useState<AkCaptchaPoint>(() =>
    value?.type === "drag"
      ? value.point
      : {
          x: clamp(challenge.initial_point.x, 0, maximumX),
          y: clamp(challenge.initial_point.y, 0, maximumY),
      },
  );

  useEffect(() => {
    setPosition(
      value?.type === "drag"
        ? value.point
        : {
            x: clamp(challenge.initial_point.x, 0, maximumX),
            y: clamp(challenge.initial_point.y, 0, maximumY),
          },
    );
  }, [challenge.initial_point.x, challenge.initial_point.y, maximumX, maximumY, value]);
  const activePointer = useRef<number | null>(null);
  const commit = (point: AkCaptchaPoint) => {
    const next = {
      x: clamp(Math.round(point.x), 0, maximumX),
      y: clamp(Math.round(point.y), 0, maximumY),
    };
    setPosition(next);
    onChange({ point: next, type: "drag" });
  };
  const pointForEvent = (event: PointerEvent<HTMLDivElement>) =>
    captchaPointFromClient(
      event.clientX,
      event.clientY,
      event.currentTarget.getBoundingClientRect(),
      challenge.image,
      challenge.tile_image,
    );
  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (disabled) return;
    const step = event.shiftKey ? 10 : 1;
    switch (event.key) {
      case "ArrowLeft":
        event.preventDefault();
        commit({ x: position.x - step, y: position.y });
        break;
      case "ArrowRight":
        event.preventDefault();
        commit({ x: position.x + step, y: position.y });
        break;
      case "ArrowUp":
        event.preventDefault();
        commit({ x: position.x, y: position.y - step });
        break;
      case "ArrowDown":
        event.preventDefault();
        commit({ x: position.x, y: position.y + step });
        break;
    }
  };

  return (
    <div className="ak-captcha-mode">
      <div
        aria-describedby={instructionId}
        aria-disabled={disabled}
        aria-keyshortcuts="ArrowLeft ArrowRight ArrowUp ArrowDown"
        aria-label={t("auth.login.captcha.drag.control")}
        className="ak-captcha-image-frame ak-captcha-drag-board"
        data-ak-captcha-control
        role="group"
        style={imageFrameStyle(challenge.image)}
        tabIndex={disabled ? -1 : 0}
        onKeyDown={handleKeyDown}
        onPointerCancel={(event) => {
          if (activePointer.current === event.pointerId) activePointer.current = null;
        }}
        onPointerDown={(event) => {
          if (disabled) return;
          activePointer.current = event.pointerId;
          event.currentTarget.focus();
          if (typeof event.currentTarget.setPointerCapture === "function") {
            event.currentTarget.setPointerCapture(event.pointerId);
          }
          commit(pointForEvent(event));
        }}
        onPointerMove={(event) => {
          if (!disabled && activePointer.current === event.pointerId) {
            commit(pointForEvent(event));
          }
        }}
        onPointerUp={(event) => {
          if (activePointer.current !== event.pointerId) return;
          commit(pointForEvent(event));
          activePointer.current = null;
          if (typeof event.currentTarget.releasePointerCapture === "function") {
            event.currentTarget.releasePointerCapture(event.pointerId);
          }
        }}
      >
        <img
          alt={t("auth.login.captcha.drag.image_alt")}
          className="ak-captcha-main-image"
          draggable={false}
          src={imageSource(challenge.image)}
        />
        <img
          alt=""
          aria-hidden="true"
          className="ak-captcha-tile-image"
          draggable={false}
          src={imageSource(challenge.tile_image)}
          style={tileStyle(position, challenge.tile_image, challenge.image)}
        />
      </div>
      <div className="ak-captcha-drag-ranges">
        <label className="ak-captcha-range-label">
          <span>{t("auth.login.captcha.drag.x")}</span>
          <input
            aria-describedby={instructionId}
            className="ak-captcha-range"
            disabled={disabled || maximumX === 0}
            max={maximumX}
            min={0}
            step={1}
            type="range"
            value={position.x}
            onChange={(event) => {
              commit({ x: event.currentTarget.valueAsNumber, y: position.y });
            }}
          />
        </label>
        <label className="ak-captcha-range-label">
          <span>{t("auth.login.captcha.drag.y")}</span>
          <input
            aria-describedby={instructionId}
            className="ak-captcha-range"
            disabled={disabled || maximumY === 0}
            max={maximumY}
            min={0}
            step={1}
            type="range"
            value={position.y}
            onChange={(event) => {
              commit({ x: position.x, y: event.currentTarget.valueAsNumber });
            }}
          />
        </label>
      </div>
      <span aria-live="polite" role="status">
        {value?.type === "drag"
          ? t("auth.login.captcha.completed")
          : t("auth.login.captcha.pending")}
      </span>
    </div>
  );
}

function RotateCaptcha({
  challenge,
  disabled,
  instructionId,
  onChange,
  value,
}: ModeProps<Extract<AkInteractiveCaptchaChallenge, { type: "rotate" }>>) {
  const { t } = useTranslation();
  const [angle, setAngle] = useState(() =>
    value?.type === "rotate" ? clamp(Math.round(value.angle), 0, 359) : 0,
  );

  useEffect(() => {
    setAngle(
      value?.type === "rotate" ? clamp(Math.round(value.angle), 0, 359) : 0,
    );
  }, [value]);
  const commit = (nextAngle: number) => {
    const next = clamp(Math.round(nextAngle), 0, 359);
    setAngle(next);
    onChange({ angle: next, type: "rotate" });
  };

  return (
    <div className="ak-captcha-mode">
      <div
        className="ak-captcha-image-frame ak-captcha-rotate-board"
        style={imageFrameStyle(challenge.image)}
      >
        <img
          alt={t("auth.login.captcha.rotate.image_alt")}
          className="ak-captcha-main-image"
          draggable={false}
          src={imageSource(challenge.image)}
        />
        <img
          alt=""
          aria-hidden="true"
          className="ak-captcha-rotate-thumb"
          draggable={false}
          src={imageSource(challenge.thumb_image)}
          style={{ transform: `translate(-50%, -50%) rotate(${String(angle)}deg)` }}
        />
      </div>
      <label className="ak-captcha-range-label">
        <span>{t("auth.login.captcha.rotate.control")}</span>
        <input
          aria-describedby={instructionId}
          className="ak-captcha-range"
          data-ak-captcha-control
          disabled={disabled}
          max={359}
          min={0}
          step={1}
          type="range"
          value={angle}
          onChange={(event) => {
            commit(event.currentTarget.valueAsNumber);
          }}
        />
      </label>
      <span aria-live="polite" role="status">
        {value?.type === "rotate"
          ? t("auth.login.captcha.completed")
          : t("auth.login.captcha.pending")}
      </span>
    </div>
  );
}

export const AkInteractiveCaptcha = forwardRef<
  AkInteractiveCaptchaHandle,
  AkInteractiveCaptchaProps
>(function AkInteractiveCaptcha(
  {
    autoFocus = false,
    challenge,
    disabled = false,
    error,
    onChange,
    onRefresh,
    value,
  },
  ref,
) {
  const { t } = useTranslation();
  const rootRef = useRef<HTMLElement>(null);
  const onChangeRef = useRef(onChange);
  const [expired, setExpired] = useState(false);
  const id = useId();
  const headingId = `${id}-heading`;
  const instructionId = `${id}-instruction`;
  onChangeRef.current = onChange;

  useImperativeHandle(ref, () => ({
    focus: () => {
      rootRef.current
        ?.querySelector<HTMLElement>("[data-ak-captcha-control]")
        ?.focus();
    },
  }));

  useEffect(() => {
    if (!autoFocus) return undefined;
    const animationFrame = window.requestAnimationFrame(() => {
      rootRef.current
        ?.querySelector<HTMLElement>("[data-ak-captcha-control]")
        ?.focus();
    });
    return () => {
      window.cancelAnimationFrame(animationFrame);
    };
  }, [autoFocus, challenge.captcha_id]);

  useEffect(() => {
    setExpired(false);
    const timeout = window.setTimeout(() => {
      setExpired(true);
      onChangeRef.current(null);
      rootRef.current?.querySelector<HTMLButtonElement>(".ak-captcha-heading button")?.focus();
    }, Math.max(0, challenge.expires_in_seconds) * 1000);
    return () => {
      window.clearTimeout(timeout);
    };
  }, [challenge.captcha_id, challenge.expires_in_seconds]);

  const interactionDisabled = disabled || expired;

  let modeContent;
  switch (challenge.type) {
    case "click":
      modeContent = (
        <ClickCaptcha
          challenge={challenge}
          disabled={interactionDisabled}
          instructionId={instructionId}
          key={challenge.captcha_id}
          value={value}
          onChange={onChange}
        />
      );
      break;
    case "slide":
      modeContent = (
        <SlideCaptcha
          challenge={challenge}
          disabled={interactionDisabled}
          instructionId={instructionId}
          key={challenge.captcha_id}
          value={value}
          onChange={onChange}
        />
      );
      break;
    case "drag":
      modeContent = (
        <DragCaptcha
          challenge={challenge}
          disabled={interactionDisabled}
          instructionId={instructionId}
          key={challenge.captcha_id}
          value={value}
          onChange={onChange}
        />
      );
      break;
    case "rotate":
      modeContent = (
        <RotateCaptcha
          challenge={challenge}
          disabled={interactionDisabled}
          instructionId={instructionId}
          key={challenge.captcha_id}
          value={value}
          onChange={onChange}
        />
      );
      break;
  }

  return (
    <section
      aria-busy={disabled}
      aria-labelledby={headingId}
      className="ak-interactive-captcha"
      ref={rootRef}
    >
      <div className="ak-captcha-heading">
        <Typography.Text id={headingId} strong>
          {t("auth.login.captcha.label")}
        </Typography.Text>
        <Tooltip title={t("auth.login.captcha.refresh")}>
          <Button
            aria-label={t("auth.login.captcha.refresh")}
            disabled={disabled}
            icon={<ReloadOutlined aria-hidden="true" />}
            onClick={onRefresh}
          />
        </Tooltip>
      </div>
      <Typography.Paragraph id={instructionId} type="secondary">
        {t(`auth.login.captcha.${challenge.type}.instruction`)}
      </Typography.Paragraph>
      {modeContent}
      {error || expired ? (
        <div className="ak-captcha-error" role="alert">
          {error ?? t("auth.login.captcha.error.expired")}
        </div>
      ) : null}
    </section>
  );
});
