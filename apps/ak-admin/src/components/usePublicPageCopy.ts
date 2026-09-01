import { useRef, useState } from "react";

export function usePublicPageCopy() {
  const [result, setResult] = useState<{ url: string; ok: boolean } | null>(null);
  const latest = useRef(0);
  const copy = async (url: string) => {
    const attempt = ++latest.current;
    try {
      await navigator.clipboard.writeText(url);
      if (attempt === latest.current) setResult({ url, ok: true });
    } catch {
      if (attempt === latest.current) setResult({ url, ok: false });
    }
  };
  return { result, copy };
}
